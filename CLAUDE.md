# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build
go build -o clashtui .

# Run
go run .

# Install to PATH
go install .
```

## Architecture Overview

ClashTUI is a terminal UI for managing Clash/mihomo proxy. Uses BubbleTea framework with message-driven architecture.

### Package Structure

- `main.go` - Entry point; sets up bubbletea program with single-instance check
- `internal/app/app.go` - Main Model with 3 tabs (Nodes/Config/Logs); handles key events and orchestrates all components
- `internal/clash/` - Clash integration:
  - `core.go` - Process management (download, start/stop mihomo binary, geo data)
  - `client.go` - REST API client for Clash external controller (127.0.0.1:9090)
  - `proxy.go` - ProxyInfo type and API methods (GetAllProxies, SwitchProxy, TestDelay)
- `internal/config/config.go` - File paths (~/.config/clashtui/), config/subscription persistence
- `internal/proxy/proxy.go` - System proxy via gsettings (GNOME) and kwriteconfig (KDE); creates ~/.config/clashtui/proxy.sh for terminal users
- `internal/clipboard/clipboard.go` - Clipboard read via wl-paste (Wayland) or xclip/xsel (X11)
- `internal/tui/` - BubbleTea components:
  - `nodes.go` - Proxy list with selection, delay testing, auto-test on load
  - `logs.go` - Log display (thread-safe, max 100 lines)
  - `styles.go` - Lipgloss styling definitions
- `internal/singleinstance/singleinstance.go` - PID file mechanism (/tmp/clashtui.pid)

### Key Message Types (internal/tui/nodes.go)

- `MsgProxiesLoaded` - Proxy list loaded from API
- `MsgProxySwitched` - Proxy selection changed
- `MsgDelayTested` - Single delay test result
- `MsgRefresh` - Trigger proxy reload
- `MsgLogLine` - Add log entry

### Data Flow

1. User imports subscription (clipboard/manual) → `clash.DownloadSubscription()` parses base64 links, builds config.yaml
2. Core starts → `clash.Core.Start()` spawns mihomo process with `-d` pointing to config dir
3. Nodes tab → `clash.Client.GetAllProxies()` fetches from API, auto-starts sequential delay testing
4. Proxy switch → `clash.Client.SwitchProxy()` calls PUT /proxies/Auto

### Runtime Requirements

- TUN mode: `sudo setcap cap_net_admin+ep ~/.config/clashtui/core/clash`
- Clipboard: `wl-clipboard` (Wayland) or `xclip`/`xsel` (X11)

## Protocol Support

Subscription parsing in `clash/core.go:parseNode()` handles:
- `trojan://`
- `vless://`
- `hysteria2://` and `hy2://`