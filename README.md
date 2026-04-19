# tolink

tolink is a local-only URL shortcut service. Add a shortcut like `gh → https://github.com` and then type `to/gh` in your browser to be instantly redirected. It runs as a lightweight Go binary on your machine with no external dependencies, stores shortcuts in a plain JSON file, and provides a simple web UI for managing them.

## Prerequisites

- Go 1.21+
- Linux with systemd

## One-time `/etc/hosts` setup

Add the following line to `/etc/hosts` so your browser resolves `to` to localhost:

```
127.0.0.1 to
```

## One-time port redirect setup

The service listens on port 4080, but browsers connect to port 80 for plain `http://` URLs. Redirect port 80 to 4080 with iptables:

```bash
sudo iptables -t nat -A OUTPUT -d 127.0.0.1 -p tcp --dport 80 -j REDIRECT --to-ports 4080
```

To persist the rule across reboots:

```bash
sudo apt install iptables-persistent
sudo netfilter-persistent save
```

After this, `http://to/gh` works in your browser. Once you have visited any `http://to/…` URL, Chrome learns that `to` is a real host and you can drop the `http://` prefix — bare `to/gh` will navigate directly without searching.

## Build and install

```bash
make build    # compiles to ./bin/tolink
make install  # installs binary and systemd service, then enables it
```

## Start the service

```bash
sudo systemctl start tolink
```

## Admin UI

Open `http://to/.tolink/` in your browser to add, edit, and delete shortcuts.

## CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `80` | Port to listen on |
| `--data` | `~/.config/tolink/links.json` | Path to the links JSON file |
