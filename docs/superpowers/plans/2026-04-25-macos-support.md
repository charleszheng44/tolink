# macOS Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `make install` produce a working `tolink` install on macOS that mirrors the Linux behavior — auto-starting daemon at login, port-80→4080 redirect, admin UI reachable at `http://to/.tolink/`.

**Architecture:** Pure tooling and docs change; no Go source modifications. Add a per-user LaunchAgent plist (`tolink.plist`) that calls `/usr/local/bin/tolink` with its existing default flags. Branch the `Makefile` on `uname -s` so `install`/`uninstall` dispatch to either systemd or launchctl. Document the macOS pf redirect parallel to the existing iptables block.

**Tech Stack:** launchd (`.plist`, `launchctl`), pf (`pfctl`, `/etc/pf.conf` anchors), GNU make conditionals (`ifeq ($(UNAME),Darwin)`).

**Spec:** [`docs/superpowers/specs/2026-04-25-macos-support-design.md`](../specs/2026-04-25-macos-support-design.md)

---

## Task 1: Add the LaunchAgent plist file

**Files:**
- Create: `tolink.plist`

- [ ] **Step 1: Create `tolink.plist` at the repo root**

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

- [ ] **Step 2: Validate the plist is well-formed**

Run (macOS only — skip if implementing on Linux):
```bash
plutil -lint tolink.plist
```
Expected: `tolink.plist: OK`

On Linux, instead verify the file exists and is readable:
```bash
test -f tolink.plist && head -1 tolink.plist
```
Expected: prints the `<?xml version=...?>` line.

- [ ] **Step 3: Commit**

```bash
git add tolink.plist
git commit -m "feat: add macOS LaunchAgent plist"
```

---

## Task 2: Branch the Makefile on `uname -s`

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Replace the entire `Makefile` with the OS-aware version**

Current contents (for reference):
```makefile
.PHONY: build install clean uninstall

build:
	go build -o bin/tolink ./cmd/tolink
	go build -o bin/tolinkctl ./cmd/tolinkctl

install: build
	sudo cp bin/tolink /usr/local/bin/tolink
	sudo cp bin/tolinkctl /usr/local/bin/tolinkctl
	sudo cp tolink.service /etc/systemd/system/tolink.service
	sudo systemctl daemon-reload
	sudo systemctl enable tolink
	sudo systemctl is-active --quiet tolink && sudo systemctl restart tolink || sudo systemctl start tolink

clean:
	rm -rf bin/

uninstall:
	sudo systemctl stop tolink || true
	sudo systemctl disable tolink || true
	sudo rm -f /usr/local/bin/tolink /usr/local/bin/tolinkctl /etc/systemd/system/tolink.service
	sudo systemctl daemon-reload
```

New contents:
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

- [ ] **Step 2: Verify `make build` still works on the host**

Run:
```bash
make clean && make build
```
Expected: produces `bin/tolink` and `bin/tolinkctl`, no errors.

- [ ] **Step 3: Verify `make` parses the conditional on the non-host branch by dry-running with overridden UNAME**

Run:
```bash
make -n install UNAME=Linux | head -10
make -n install UNAME=Darwin | head -10
```
Expected: each prints the matching branch's commands without errors. The Linux branch shows `sudo cp ... /etc/systemd/system/tolink.service`; the Darwin branch shows `cp tolink.plist ...LaunchAgents/...`.

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "feat(make): branch install/uninstall on uname for macOS"
```

---

## Task 3: Update README for macOS

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the Prerequisites section**

Replace lines 5–8:
```markdown
## Prerequisites

- Go 1.21+
- Linux with systemd
```
with:
```markdown
## Prerequisites

- Go 1.21+
- Linux with systemd, or macOS
```

- [ ] **Step 2: Rename the existing port-redirect section header to scope it to Linux**

Replace the line:
```markdown
## One-time port redirect setup
```
with:
```markdown
## One-time port redirect setup (Linux)
```

(The body of the section — iptables instructions and persistence note — stays unchanged.)

- [ ] **Step 3: Insert the macOS port-redirect section directly after the Linux one**

Add a new section immediately after the Linux iptables persistence block (after the `sudo netfilter-persistent save` block, before the "After this, `http://to/gh` works..." paragraph — the new section goes between the two):

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

- [ ] **Step 4: Verify the rendered structure**

Run:
```bash
grep -n '^##' README.md
```
Expected: among the headings, you see (in order):
- `## Prerequisites`
- `## One-time \`/etc/hosts\` setup`
- `## One-time port redirect setup (Linux)`
- `## One-time port redirect setup (macOS)`
- `## Build and install`

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: add macOS prerequisites and pf redirect instructions"
```

---

## Task 4: Verify Go tests pass on the host

**Files:** none modified

- [ ] **Step 1: Run the full test suite**

Run:
```bash
go test ./...
```
Expected: PASS for all packages, including `cmd/tolinkctl` (which contains `TestE2ESmoke` in `cmd/tolinkctl/e2e_test.go`). The smoke test is HTTP-based and platform-agnostic, so it should pass on macOS unmodified.

If tests fail, stop and investigate before continuing — they should pass without any changes from this plan.

---

## Task 5: Manual verification on a macOS host

> **Note:** Skip this task if the implementing engineer does not have a macOS machine. The plan owner (the user) will run this themselves. Mark each step done only when verified on the host.

**Files:** none modified

- [ ] **Step 1: Confirm prerequisites are in place**

```bash
grep -E '^127\.0\.0\.1\s+to(\s|$)' /etc/hosts
```
Expected: prints the `127.0.0.1 to` line. If not, add it (`sudo` required).

- [ ] **Step 2: Run `make install`**

```bash
make install
```
Expected: prompts for `sudo` password (for the `/usr/local/bin` copies), then completes without errors.

- [ ] **Step 3: Verify the LaunchAgent loaded**

```bash
launchctl list | grep tolink
```
Expected: a line whose third column is `com.charleszheng44.tolink` and whose first column is a numeric PID (not `-`), meaning the daemon is running.

- [ ] **Step 4: Verify the daemon serves the admin UI directly on 4080**

```bash
curl -sI http://127.0.0.1:4080/.tolink/ | head -1
```
Expected: `HTTP/1.1 200 OK`.

- [ ] **Step 5: Set up the pf redirect (one-time, per the README)**

```bash
echo 'rdr pass on lo0 inet proto tcp from any to 127.0.0.1 port 80 -> 127.0.0.1 port 4080' | sudo tee /etc/pf.anchors/com.tolink
sudo sh -c 'cat >> /etc/pf.conf <<EOF
rdr-anchor "com.tolink"
load anchor "com.tolink" from "/etc/pf.anchors/com.tolink"
EOF'
sudo pfctl -ef /etc/pf.conf
```
Expected: `pfctl: pf already enabled` or `pf enabled`, and rule loading messages with no syntax errors.

- [ ] **Step 6: Verify the redirect end-to-end**

```bash
curl -sI http://to/.tolink/ | head -1
```
Expected: `HTTP/1.1 200 OK`.

Open `http://to/.tolink/` in a browser, add a shortcut `gh → https://github.com`, then:
```bash
curl -sI http://to/gh | grep -iE '^(HTTP|Location)'
```
Expected:
```
HTTP/1.1 302 Found
Location: https://github.com
```

- [ ] **Step 7: Verify `make uninstall` is clean**

```bash
make uninstall
ls $HOME/Library/LaunchAgents/com.charleszheng44.tolink.plist 2>&1
ls /usr/local/bin/tolink /usr/local/bin/tolinkctl 2>&1
launchctl list | grep tolink
```
Expected:
- `ls ...plist` prints `No such file or directory`
- `ls /usr/local/bin/...` prints `No such file or directory` for both
- `launchctl list | grep tolink` prints nothing (exit code 1 is fine here)

- [ ] **Step 8: Reinstall to leave the host in a working state**

```bash
make install
```
Expected: same as Step 2.

---

## Task 6: Open the PR

**Files:** none modified — this task only affects git/GitHub state.

- [ ] **Step 1: Push the branch**

Run (replace `<branch>` with the actual branch name created at the start of work):
```bash
git push -u origin <branch>
```

- [ ] **Step 2: Open the PR with both docs and the implementation in one diff**

```bash
gh pr create --title "Add macOS support (LaunchAgent + pf redirect)" --body "$(cat <<'EOF'
## Summary
- Run tolink on macOS via a per-user LaunchAgent (`~/Library/LaunchAgents/com.charleszheng44.tolink.plist`) that auto-starts at login
- `make install` / `make uninstall` now branch on `uname -s` — Linux path unchanged, macOS path uses `launchctl`
- README documents the macOS pf redirect parallel to the existing Linux iptables block
- No Go source changes; daemon and `tolinkctl` already work on macOS

## Test plan
- [ ] `go test ./...` passes on macOS
- [ ] `make build` produces working binaries on macOS
- [ ] `make install` loads the LaunchAgent and the daemon serves `http://127.0.0.1:4080/.tolink/`
- [ ] After one-time pf setup from README, `http://to/gh` redirects to the configured target
- [ ] `make uninstall` removes the plist, the binaries, and stops the daemon with no residue
- [ ] On Linux, `make install`/`uninstall` behave identically to before this change

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Expected: prints the PR URL. Report it back to the user.

---

## Self-Review Notes

- **Spec coverage:** Tasks 1, 2, 3 implement the plist, Makefile, and README sections of the spec respectively. Tasks 4 and 5 cover the spec's Verification section. Task 6 is the user-requested PR step (not part of the spec but explicit user instruction).
- **Placeholders:** none — every step has full file content, exact commands, and expected output.
- **Type consistency:** plist `Label` `com.charleszheng44.tolink` matches the `~/Library/LaunchAgents/com.charleszheng44.tolink.plist` filename used in the Makefile and uninstall targets.
- **Frequent commits:** Tasks 1, 2, 3 each end with a focused commit. Tasks 4 and 5 are verification-only and commit nothing. Task 6 is the PR.
