# tolink — Design Spec

**Date:** 2026-04-18
**Status:** Approved

## Overview

`tolink` is a local-only URL shortcut service inspired by [tailscale/golink](https://github.com/tailscale/golink). Type `to/gh` in any browser on your machine and get redirected to `https://github.com`. Shortcuts are managed via a web UI at `to/.tolink/`. The service runs as a systemd unit.

## Goals

- Zero-dependency single Go binary
- No Tailscale or network dependency — localhost only
- Human-editable JSON storage
- Simple web UI for CRUD (no frontend build step)
- Runs as a non-root systemd service on port 80

## Non-Goals

- Multi-user or networked access
- Authentication
- Analytics / click tracking
- Dynamic redirect patterns (e.g. `to/gh/{path}`)

---

## Architecture

One HTTP server on port 80, two responsibilities multiplexed on the same listener:

```
Browser: to/gh
  → /etc/hosts resolves "to" → 127.0.0.1
  → tolink server on :80
  → 302 → https://github.com

Browser: to/.tolink/
  → serves admin web UI (embedded HTML)
```

**Request routing (in order):**
1. Path starts with `/.tolink/` → admin handler
2. Path matches a known shortcut → 302 redirect
3. Otherwise → redirect to `to/.tolink/` (admin UI)

**Project layout:**
```
tolink/
├── cmd/tolink/main.go        # entry point, CLI flags
├── pkg/
│   ├── store/store.go        # JSON file + in-memory map + RWMutex
│   └── server/server.go      # HTTP handlers (redirect + admin API)
├── web/index.html            # embedded via embed.FS
├── tolink.service            # systemd unit file
├── Makefile                  # build, install targets
└── go.mod                    # module: github.com/charleszheng44/tolink
```

No external dependencies — standard library only.

---

## Storage

**File location:** `~/.config/tolink/links.json`

**Format:**
```json
{
  "gh": "https://github.com",
  "hn": "https://news.ycombinator.com"
}
```

**Implementation (`pkg/store`):**
- Load file into `map[string]string` at startup
- `sync.RWMutex` for concurrent read/write safety
- Atomic writes: marshal to a temp file in the same directory, then `os.Rename` into place
- Exported methods: `Get(shortcut)`, `List()`, `Set(shortcut, url)`, `Delete(shortcut)`

---

## HTTP API

All admin endpoints are under `/.tolink/api/`:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/.tolink/` | Serve admin web UI |
| `GET` | `/.tolink/api/links` | Return all links as JSON object |
| `POST` | `/.tolink/api/links` | Add or update a link. Body: `{"shortcut":"gh","url":"https://github.com"}` |
| `DELETE` | `/.tolink/api/links/{shortcut}` | Delete a link |

Redirect handler: `GET /{shortcut}` → `302` to target URL.

---

## Admin Web UI

Single HTML file embedded in the binary via `//go:embed web/index.html`.

**Layout:**
- Header: "tolink"
- Add/edit form: two text inputs (shortcut, URL) + Save button
- Table: shortcut | target URL | Edit | Delete

**Behavior:**
- On load: `GET /.tolink/api/links` → render table
- Add/Save: `POST /.tolink/api/links` → re-render table
- Edit: pre-fills the form with existing values
- Delete: `DELETE /.tolink/api/links/{shortcut}` → re-render table

Pure `fetch()`, no framework, no build step.

---

## Systemd Service

**Unit file:** `tolink.service` → installed to `/etc/systemd/system/`

```ini
[Unit]
Description=tolink - local URL shortcut service
After=network.target

[Service]
ExecStart=/usr/local/bin/tolink
Restart=on-failure
User=zc
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
```

- Runs as user `zc`, not root
- `CAP_NET_BIND_SERVICE` allows binding port 80 without sudo
- Data file at `~/.config/tolink/links.json` (relative to `zc`'s home)

**Makefile targets:**
- `make build` — compile binary to `./bin/tolink`
- `make install` — copy binary to `/usr/local/bin/tolink`, install and enable systemd unit

---

## One-Time Setup

Add to `/etc/hosts` (documented in README, not automated):
```
127.0.0.1 to
```

Then:
```bash
make install
sudo systemctl start tolink
```

---

## cmd/tolink/main.go Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `80` | Port to listen on |
| `--data` | `~/.config/tolink/links.json` | Path to links JSON file |
