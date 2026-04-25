# tolink — macOS Support Design Spec

**Date:** 2026-04-25
**Status:** Approved

## Overview

Extend `tolink` to run on macOS with the same user-facing behavior it has on Linux: type `to/gh` in any browser and get redirected. The daemon (`pkg/store`, `pkg/server`, `cmd/tolink`, `cmd/tolinkctl`) is already pure Go and runs unmodified on macOS. All work is in service supervision, port redirect, and install tooling.

## Goals

- `make install` works on both Linux and macOS, auto-detecting the host
- Mac install runs as a per-user LaunchAgent — auto-starts at login, no `sudo` for the daemon itself (only for the one-time `/usr/local/bin` copy and the one-time pf rule, mirroring the Linux iptables story)
- README documents macOS prerequisites parallel to the existing Linux ones
- No source-code changes to the daemon or `tolinkctl`

## Non-Goals

- System-wide LaunchDaemon scope (multi-user, survives logout) — a per-user LaunchAgent is sufficient for the single-user local tool tolink is
- Automating the pf redirect inside `make install` — it's a system-level change the user should see and approve, same as iptables on Linux today
- Migrating macOS users away from `~/.config/tolink/links.json` toward `~/Library/Application Support/...` — keeping the existing default avoids divergence between OSes
- Logging beyond `/tmp/tolink.{out,err}.log` — Linux relies on journald and isn't configured either; symmetry is good enough for this iteration

---

## Architecture

The Linux install has three OS-specific pieces; the Mac install replaces each with the macOS analogue:

| Concern | Linux | macOS |
|---|---|---|
| Service supervision | systemd unit (`tolink.service`) | LaunchAgent plist (`tolink.plist`) |
| Port 80 → 4080 redirect | `iptables -t nat ... REDIRECT` | `pf` anchor loaded from `/etc/pf.conf` |
| Hostname `to` → `127.0.0.1` | `/etc/hosts` entry | Same `/etc/hosts` entry |
| Binary install | `sudo cp ... /usr/local/bin/` | Same `sudo cp ... /usr/local/bin/` |
| `make install` dispatch | systemd path | launchctl path, selected by `uname -s` |

The daemon binds to `127.0.0.1:4080` on both platforms (its existing default) and serves redirects + the admin UI exactly as on Linux.

### Files added/changed

- **New:** `tolink.plist` (repo root, sibling of `tolink.service`)
- **Modified:** `Makefile` — `install` and `uninstall` branch on `uname -s`
- **Modified:** `README.md` — Prerequisites change to "Linux or macOS"; new "One-time port redirect setup (macOS)" section parallel to the existing iptables block

No changes to `cmd/`, `pkg/`, `web/`, or `go.mod`.

---

## LaunchAgent plist

The daemon's existing flag defaults already match what we want on macOS:

- `-addr` defaults to `127.0.0.1:4080` ✓
- `-data` defaults to `$HOME/.config/tolink/links.json` ✓ (Go's `os.Getenv("HOME")` returns the right path; LaunchAgents run as the user, so `~/.config/tolink/` is writable without privileges)

The plist therefore just points at the binary and lets defaults do the rest:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.charleszheng44.tolink</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/tolink</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <dict>
    <key>SuccessfulExit</key>
    <false/>
  </dict>
  <key>StandardOutPath</key>
  <string>/tmp/tolink.out.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/tolink.err.log</string>
</dict>
</plist>
```

**Key choices:**

- **Label** `com.charleszheng44.tolink` follows the reverse-DNS convention and matches the GitHub username already used in `go.mod`.
- **`KeepAlive` with `SuccessfulExit=false`** mirrors systemd's `Restart=on-failure`: relaunch only on crash, not on intentional exit.
- **Logs go to `/tmp`** because launchd plists do not expand `~`. Linux configures no logging today (relies on journald), so this matches the level of effort and avoids needing to template `$HOME` into the installed plist.

The plist file is committed verbatim and copied at install time — no substitution needed.

---

## Makefile

Add `UNAME := $(shell uname -s)` and branch `install` and `uninstall`:

```makefile
.PHONY: build install clean uninstall

UNAME := $(shell uname -s)

build:
	go build -o bin/tolink ./cmd/tolink
	go build -o bin/tolinkctl ./cmd/tolinkctl

install: build
ifeq ($(UNAME),Darwin)
	sudo cp bin/tolink /usr/local/bin/tolink
	sudo cp bin/tolinkctl /usr/local/bin/tolinkctl
	mkdir -p $(HOME)/Library/LaunchAgents
	cp tolink.plist $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist
	launchctl unload $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist 2>/dev/null || true
	launchctl load $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist
else
	sudo cp bin/tolink /usr/local/bin/tolink
	sudo cp bin/tolinkctl /usr/local/bin/tolinkctl
	sudo cp tolink.service /etc/systemd/system/tolink.service
	sudo systemctl daemon-reload
	sudo systemctl enable tolink
	sudo systemctl is-active --quiet tolink && sudo systemctl restart tolink || sudo systemctl start tolink
endif

clean:
	rm -rf bin/

uninstall:
ifeq ($(UNAME),Darwin)
	launchctl unload $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist 2>/dev/null || true
	rm -f $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist
	sudo rm -f /usr/local/bin/tolink /usr/local/bin/tolinkctl
else
	sudo systemctl stop tolink || true
	sudo systemctl disable tolink || true
	sudo rm -f /usr/local/bin/tolink /usr/local/bin/tolinkctl /etc/systemd/system/tolink.service
	sudo systemctl daemon-reload
endif
```

**Notes:**

- `launchctl unload ... 2>/dev/null || true` before `launchctl load` makes `make install` idempotent — it works whether or not the agent is already loaded (e.g. on upgrade).
- The Linux branch is byte-identical to today's `Makefile` body; the change is purely additive.
- `sudo` is still required on macOS for the binary copy into `/usr/local/bin` (same as Linux). No `sudo` is required for the LaunchAgent steps themselves.

---

## README additions

Two changes to `README.md`:

**1. Update Prerequisites section:**

```markdown
## Prerequisites

- Go 1.21+
- Linux with systemd, or macOS
```

**2. Rename the existing `## One-time port redirect setup` section to `## One-time port redirect setup (Linux)` for symmetry, and add a sibling macOS block after it:**

````markdown
## One-time port redirect setup (macOS)

Create the anchor file `/etc/pf.anchors/com.tolink`:

```
rdr pass on lo0 inet proto tcp from any to 127.0.0.1 port 80 -> 127.0.0.1 port 4080
```

Append to `/etc/pf.conf`:

```
rdr-anchor "com.tolink"
load anchor "com.tolink" from "/etc/pf.anchors/com.tolink"
```

Enable pf and load the rules:

```bash
sudo pfctl -ef /etc/pf.conf
```

These persist across reboots because pf reads `/etc/pf.conf` at boot.
````

The `/etc/hosts` instruction and the "Build and install" / "Admin UI" / "CLI flags" / "tolinkctl" sections need no changes — they apply identically on macOS.

---

## Verification

- `make build` produces working `bin/tolink` and `bin/tolinkctl` on macOS (Go cross-platform; no expected friction)
- After `make install` on macOS: `launchctl list | grep tolink` shows the agent loaded and running
- `curl -I http://127.0.0.1:4080/.tolink/` returns 200 (daemon is alive without the redirect)
- After the pf step: `curl -I http://to/.tolink/` returns 200 (full redirect path works)
- After adding `gh → https://github.com` via the admin UI: `curl -I http://to/gh` returns 302 to `https://github.com`
- The existing `tolinkctl` end-to-end smoke test runs green on macOS unmodified

`make uninstall` on macOS leaves no residue: no `~/Library/LaunchAgents/com.charleszheng44.tolink.plist`, no `/usr/local/bin/tolink{,ctl}`. The pf rule and `/etc/hosts` entry are user-managed (not added by `make install`) and so are not removed by `make uninstall` — the README will note this for symmetry with the Linux iptables story.
