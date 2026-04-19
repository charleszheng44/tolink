# tolinkctl — Command-Line Client Design

**Date:** 2026-04-19
**Status:** Approved (design phase)

## Overview

`tolinkctl` is a command-line client for managing tolink shortcuts. It reuses the daemon's HTTP API when the daemon is running, and falls back to direct JSON-file access when it isn't. Single binary, stdlib only, no external dependencies.

## Goals

- Manage shortcuts (`list`, `get`, `add`, `update`, `delete`) without opening the web UI.
- Work both when the daemon is running (via HTTP) and when it is not (via the JSON file).
- Match the zero-dependency style of the existing codebase.

## Non-goals

- Shell completion.
- Short aliases (`ls`, `rm`, `set`).
- `--json` output.
- Server-side per-shortcut `GET` endpoint.
- Bulk import / export.

## Architecture

New directory layout additions:

```
cmd/tolinkctl/
  main.go               arg parsing, subcommand dispatch, exit codes

pkg/client/
  client.go             Client interface + backend selection (HTTP vs file)
  http.go               HTTPClient: talks to /.tolink/api/links
  file.go               FileClient: wraps pkg/store.Store
  client_test.go        table-driven tests
```

The `Client` interface:

```go
type Client interface {
    List() (map[string]string, error)
    Get(shortcut string) (string, error)    // returns ErrNotFound if missing
    Set(shortcut, url string) error          // upsert
    Delete(shortcut string) error            // returns ErrNotFound if missing
}
```

`ErrNotFound` re-exports `store.ErrNotFound` so callers can use `errors.Is`.

Strict `add`/`update` semantics are implemented in the CLI layer on top of `Client`:

- `add`: call `Get`; if it returns a URL, fail with "already exists"; otherwise `Set`.
- `update`: call `Get`; if it returns `ErrNotFound`, fail; otherwise `Set`.

## Command Interface

```
tolinkctl [--url URL] [--data PATH] <command> [args]

  list                          print all shortcuts
  get <shortcut>                print the URL for <shortcut>
  add <shortcut> <url>          create; fails if <shortcut> exists
  update <shortcut> <url>       overwrite; fails if <shortcut> does not exist
  delete <shortcut>             remove; fails if <shortcut> does not exist
```

### Global flags

| Flag | Default | Purpose |
|------|---------|---------|
| `--url` | `http://127.0.0.1:4080` | Daemon base URL |
| `--data` | `~/.config/tolink/links.json` | Fallback file path |

### Output formats

- `list`: two-column aligned, `shortcut  url`, sorted by shortcut. Empty store prints nothing.
- `get`: URL alone on stdout, so `$(tolinkctl get gh)` is scriptable.
- `add`, `update`, `delete`: no stdout on success (Unix silent-success convention).
- Errors: go to stderr, exit non-zero.

## Backend Selection

`client.New(url, dataPath, urlExplicit bool)`:

1. Issue `GET <url>/.tolink/api/links` with a 500ms connect timeout.
2. If the request completes (any 2xx, 4xx, or 5xx — daemon is reachable), return `HTTPClient`.
3. If it fails with a dial / connection-refused error:
   - If `urlExplicit` is true (the user passed `--url` on the command line), return the error. The user asked for a specific daemon; silently writing to a local file would be surprising.
   - Otherwise return `FileClient(dataPath)`.

The probe is a real API call, not just a TCP dial, so "someone else is bound to the port" fails open toward HTTP rather than silently writing to a stale file.

### Concurrent writes in file mode

`pkg/store.save` writes via temp file + atomic rename, so two concurrent `tolinkctl` invocations cannot corrupt the JSON. They can, however, lose updates (last write wins). This is acceptable for a local single-user tool and is not guarded against.

## Error Handling & Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Generic failure (I/O, network, parse) |
| 2 | Shortcut not found (`get`, `update`, `delete`) |
| 3 | Shortcut already exists (`add`) |
| 64 | Usage error (bad args/flags) |

Error messages are short and go to stderr, for example:

- `tolinkctl: shortcut "gh" already exists`
- `tolinkctl: shortcut "gh" not found`
- `tolinkctl: url must start with http:// or https://`

## Input Validation

Match the server's existing validation in `pkg/server/server.go`:

- URL must parse and use `http` or `https` scheme.
- Shortcut must be non-empty.

`FileClient` applies the same validation before writing so both backends behave identically.

## Systemd / Permissions Caveat

The systemd service (`tolink.service`) uses `DynamicUser=yes` and writes to `/var/lib/tolink/links.json`. In fallback mode against the systemd install, the user typically cannot write that file. This is a known limitation: fallback mode is intended for users who run the daemon manually under their own user with `--data ~/.config/tolink/links.json`, or who explicitly `--data /var/lib/tolink/links.json` and run `tolinkctl` with `sudo`. The docs should note this.

## Testing Strategy

- **HTTPClient**: `httptest.NewServer` with a small mock mux covering the four methods and error cases.
- **FileClient**: temp directory; reuse `pkg/store`, which already has its own tests. Client tests focus on interface conformance and error translation.
- **`client.New` backend selection**: spin up an `httptest` server, test "reachable → HTTPClient"; shut it down, test "unreachable → FileClient".
- **`cmd/tolinkctl` end-to-end**: one smoke test that builds the binary and exercises each subcommand against a real `httptest` server.

## Open Questions

None — design is ready for implementation.
