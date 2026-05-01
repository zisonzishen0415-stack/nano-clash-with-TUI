# AGENTS.md

Compact guidance for OpenCode sessions working in this repository.

## Build & Run

```bash
make build       # Build (or: go build -o clashtui .)
make run         # Run (or: go run .)
make install     # Install to PATH
```

- Go 1.24.0 required
- No tests in this project
- No lint/typecheck commands configured

## Architecture

Terminal UI for Clash/mihomo proxy management using BubbleTea framework.

- `main.go` - Entry point, CLI flags (--status, --daemon, --stop, --toggle, --env)
- `internal/app/app.go` - Main TUI model (3 tabs: Nodes/Config/Logs)
- `internal/clash/` - Mihomo core process + REST API client
- `internal/config/` - File paths (~/.config/clashtui/)
- `internal/tui/` - BubbleTea components

Defaults: proxy port 7890, API port 9090, mihomo core v1.18.10

See CLAUDE.md for detailed architecture and data flow.

## Runtime Dependencies

- Clipboard: `wl-clipboard` (Wayland) or `xclip`/`xsel` (X11)
- TUN mode: `sudo setcap cap_net_admin+ep ~/.config/clashtui/core/clash`

## Config Directory

```
~/.config/clashtui/
├── core/clash      # mihomo binary
├── config.yaml     # Current Clash config
├── settings.json   # User settings
├── proxy.sh        # Terminal proxy script (auto-generated)
├── Country.mmdb    # GeoIP
└── geosite.dat     # GeoSite
```

## Key Behaviors

- Single instance: `/tmp/clashtui.pid` lock
- Auto-modifies `~/.bashrc` and `~/.zshrc` for terminal proxy loading
- Protocol parsing in `internal/clash/core.go:parseNodeConfig()` supports 11+ protocols (see README.md)