# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

See also AGENTS.md for compact guidance suitable for other AI assistants.

## Build Commands

```bash
# Build (Go 1.24.0 required)
go build -o clashtui .

# Run
go run .

# Install to PATH
go install .

# Or use Makefile
make build && make run
```

No tests or lint commands configured in this project.

## CLI Commands

| Command | Description |
|---------|-------------|
| `clashtui` | Launch TUI interface |
| `clashtui --status` | Output JSON status for Waybar |
| `clashtui --daemon` | Background mode (for systemd service) |
| `clashtui --stop` | Stop mihomo |
| `clashtui --toggle` | Toggle proxy on/off |
| `clashtui --restore-network` | Restore network after reboot (kills mihomo) |
| `clashtui --env` | Print proxy environment variables |

## Architecture Overview

ClashTUI is a terminal UI for managing Clash/mihomo proxy with **TUN mode only**. Uses BubbleTea framework with message-driven architecture.

**Default ports:** Proxy 7890, API 9090, Mihomo core v1.18.10

### Package Structure

- `main.go` - Entry point; CLI flag handling + single-instance check
- `internal/app/app.go` - Main Model with 3 tabs (Nodes/Config/Logs); handles key events, orchestrates components
- `internal/clash/` - Clash integration:
  - `core.go` - Process management (download, start/stop mihomo binary, geo data); subscription parsing; DNS config replacement; TUN config processing
  - `client.go` - REST API client for Clash external controller (127.0.0.1:9090)
  - `proxy.go` - ProxyInfo type and API methods (GetAllProxies, SwitchProxy, TestDelay)
- `internal/settings/settings.go` - User settings persistence (subscriptions, ports, toggles)
- `internal/config/config.go` - File paths (~/.config/clashtui/), config.yaml handling
- `internal/clipboard/clipboard.go` - Clipboard read via wl-paste (Wayland) or xclip/xsel (X11)
- `internal/tui/` - BubbleTea components:
  - `nodes.go` - Proxy list with selection, delay testing, auto-test on load
  - `logs.go` - Log display (thread-safe, max 100 lines)
  - `styles.go` - Lipgloss styling definitions
- `internal/singleinstance/singleinstance.go` - PID file mechanism (/tmp/clashtui.pid)
- `internal/state/state.go` - Network state tracking (ModeOff, ModeTUN)
- `internal/health/health.go` - Health checks for stale TUN mode

### Key Message Types (internal/tui/nodes.go)

- `MsgProxiesLoaded` - Proxy list loaded from API
- `MsgProxySwitched` - Proxy selection changed
- `MsgDelayTested` - Single delay test result
- `MsgRefresh` - Trigger proxy reload (core started)
- `MsgLogLine` - Add log entry
- `MsgStopCore` - Stop core signal

### Data Flow

1. User imports subscription (clipboard/manual) → `clash.DownloadSubscription()` parses base64 links, builds config.yaml
2. Core starts → `clash.Core.Start()` spawns mihomo process with `-d` pointing to config dir
3. Nodes tab → `clash.Client.GetAllProxies()` fetches from API, auto-starts sequential delay testing
4. Proxy switch → `clash.Client.SwitchProxy()` calls PUT /proxies/Auto
5. TUN mode → `clash.ProcessConfigForTUN()` adds TUN section to config.yaml

### TUN Mode

TUN mode creates a virtual network interface that captures all traffic at kernel level. Requires capability:

```bash
sudo setcap cap_net_admin+ep ~/.config/clashtui/core/clash
```

When enabled in settings, TUN section is added to config.yaml:
```yaml
tun:
  enable: true
  stack: system
  dns-hijack:
    - any:53
  auto-route: true
```

### DNS Configuration

Uses `redir-host` mode (not `fake-ip`) to prevent DNS hijacking issues. The `replaceDNSInConfig()` function in core.go enforces safe DNS settings when processing subscription configs.

Default DNS servers: 223.5.5.5, 119.29.29.29 (Chinese public DNS)
Fallback: 1.1.1.1, dns.google (for international resolution)

## Config Directory

```
~/.config/clashtui/
├── core/clash          # mihomo binary
├── config.yaml         # Current Clash config (auto-generated)
├── settings.json       # User settings (subscriptions, ports, toggles)
├── clash.pid           # Mihomo process PID (for cleanup)
├── Country.mmdb        # GeoIP database
└── geosite.dat         # GeoSite data
└── network-state.json  # Network state tracking
```

## Settings Structure (settings.json)

```json
{
  "subscriptions": [{"name": "...", "url": "...", "traffic": "...", "expiry": "..."}],
  "active_sub_idx": 0,
  "auto_start": false,
  "auto_test_delay": true,
  "auto_select_best": true,
  "tun_mode": true,
  "proxy_port": 7890,
  "api_port": 9090
}
```

## Waybar Integration

`clashtui --status` outputs JSON for Waybar status display:

```json
{"text":"🟢","tooltip":"Proxy: DIRECT","class":"running"}
{"text":"🔴","tooltip":"Proxy: stopped","class":"stopped"}
```

Waybar config example:
```json
"custom/clashtui": {
  "exec": "clashtui --status",
  "on-click": "clashtui",
  "on-click-right": "clashtui --toggle",
  "interval": 5
}
```

## Runtime Requirements

- Go 1.24.0+
- Clipboard: `wl-clipboard` (Wayland) or `xclip`/`xsel` (X11)
- TUN mode: `sudo setcap cap_net_admin+ep ~/.config/clashtui/core/clash`

## Protocol Support

Subscription parsing in `clash/core.go:parseNodeConfig()` handles:
- Trojan, VLESS, VMess, Shadowsocks, ShadowsocksR
- Hysteria2/hy2, Hysteria
- SOCKS5, HTTP/HTTPS proxy
- WireGuard, TUIC, SSH

## Known Issue: Network Recovery After Reboot

If network is broken after reboot (TUN persists but mihomo doesn't run):
- Run `clashtui --restore-network` to kill stale mihomo
- If DNS still broken, may need: `sudo systemctl restart systemd-resolved`

## Single Instance & IPC Architecture

ClashTUI uses **flock + Unix Socket** for robust single-instance control:

**flock (`/tmp/clashtui.pid`):**
- Kernel-level atomic lock acquired via `syscall.Flock(LOCK_EX|LOCK_NB)`
- Only one process can hold the lock at any time
- Lock auto-releases on process exit (even if crashed)
- Prevents race conditions when multiple processes try to start simultaneously

**Unix Socket (`/tmp/clashtui.sock`):**
- IPC channel between CLI commands and running TUI/daemon
- Commands like `--stop`, `--toggle` delegate to running instance via socket
- Running instance executes operations and returns result
- Ensures state changes are coordinated by the process holding the lock

**Core mutex:**
- `Core.mu` protects `Start()`/`Stop()` from concurrent access
- Prevents socket handler and TUI from conflicting operations

**Command behavior:**
| Command | TUI running | TUI not running |
|---------|-------------|-----------------|
| `--stop` | Socket → TUI executes | Direct operation |
| `--toggle` | Socket → TUI executes | Direct operation |
| `--restore-network` | Kill TUI first (emergency) | Direct operation |
| `--status` | Read mihomo API directly | Same |