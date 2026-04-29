# ClashTUI Feature Improvements Design

## Overview

Enhance ClashTUI with UI improvements, implement unused settings, add multi-subscription support, and parse subscription traffic info.

## Goals

1. **UI/UX Improvements**: Color-coded delays, node sorting by latency
2. **Settings Implementation**: AutoTestDelay, AutoSelectBest, ProxyPort, APIPort, DefaultNode
3. **Multi-subscription Support**: Manage multiple subscriptions with switch functionality
4. **Subscription Info**: Display traffic usage and expiry date

## Data Model Changes

### Subscription (new)

```go
type Subscription struct {
    Name       string    `json:"name"`        // User-defined name or extracted from URL
    URL        string    `json:"url"`
    Traffic    string    `json:"traffic"`     // e.g., "234GB/500GB" or empty
    Expiry     string    `json:"expiry"`      // e.g., "2025-05-01" or empty
    LastUpdate time.Time `json:"last_update"`
}
```

### Settings (updated)

```go
type Settings struct {
    Subscriptions  []Subscription `json:"subscriptions"`
    ActiveSubIdx   int            `json:"active_sub_idx"`
    AutoStart      bool           `json:"auto_start"`
    AutoTestDelay  bool           `json:"auto_test_delay"`
    AutoSelectBest bool           `json:"auto_select_best"`
    UseDefaultNode bool           `json:"use_default_node"`
    DefaultNode    string         `json:"default_node"`
    ProxyPort      int            `json:"proxy_port"`
    APIPort        int            `json:"api_port"`
}
```

## UI Changes

### Config Tab Layout

```
┌─ Subscriptions ──────────────────────────────┐
│ > [1] AirportA   234GB/500GB  exp:2025-05-01 │  <- Active (green highlight)
│   [2] AirportB   ---          ---            │  <- Inactive (gray)
│   [3] + Add subscription                     │
│                                              │
│   j/k: select | enter: switch | d: delete   │
└──────────────────────────────────────────────┘

┌─ Settings ───────────────────────────────────┐
│ > [x] Auto start on boot                     │
│   [x] Auto test delay on load               │
│   [x] Auto select fastest node              │
│   [ ] Use default node: ---                  │
│                                              │
│   Proxy port: 7890  |  API port: 9090        │
│   p: edit proxy | a: edit api               │
└──────────────────────────────────────────────┘

  c: import clipboard | s: enter URL | r: refresh
```

### Nodes Tab Improvements

**Delay Color Coding:**
- Green (`#10B981`): < 100ms
- Yellow (`#FBBF24`): 100-299ms
- Red (`#EF4444`): >= 300ms
- Gray (`#6B7280`): timeout/0

**Node Sorting:**
- Sort by delay ascending (fastest first)
- Nodes with delay=0 (untested/timeout) go to end

**Display:**
```
> * HongKong-01      45ms    <- green
  * HongKong-02      120ms   <- yellow
  o Japan-01         350ms   <- red
  o US-01            -       <- gray

  [Testing: 5/20]

  j/k: select | enter: switch | t: test | T: test all
```

## Component Changes

### 1. internal/settings/settings.go

- Add `Subscription` struct
- Update `Settings` struct with new fields
- Update `Save()` and `Load()` to handle new fields
- Add `GetActiveSubscription()` helper
- Add `AddSubscription()`, `RemoveSubscription()`, `SwitchSubscription()` helpers

### 2. internal/app/app.go

**Config Tab:**
- Split view: subscriptions list + settings menu
- Track subscription index (`subIdx`) and settings index (`setIdx`)
- Handle subscription switching with confirmation
- Handle subscription deletion
- Handle port editing with input mode

**Settings Integration:**
- Use `AutoTestDelay` to control whether to auto-test on load
- Use `AutoSelectBest` to trigger auto-switch after all tests complete
- Use `ProxyPort` and `APIPort` when building config

### 3. internal/tui/nodes.go

**Sorting:**
- Sort proxies by delay after loading
- Keep untested/timeout nodes at end

**Color Coding:**
- Add delay color logic in `View()`
- Use styles from `styles.go`

**AutoSelectBest:**
- After all delays tested (MsgDelayTested for last index)
- If `AutoSelectBest` is true, switch to fastest node
- Only on initial load, not on manual test

### 4. internal/tui/styles.go

- Add delay color styles: `DelayGreen`, `DelayYellow`, `DelayRed`, `DelayGray`
- Add subscription highlight style: `SubActive`, `SubInactive`

### 5. internal/config/config.go

- Remove single `SaveSubscription`/`LoadSubscription`
- Subscriptions now stored in settings.json

### 6. internal/clash/core.go

**Port Configuration:**
- `buildConfig()` accepts `proxyPort` and `apiPort` parameters
- Use dynamic ports instead of hardcoded 7890/9090

### 7. internal/clash/client.go

**Dynamic API Port:**
- `NewClient()` accepts `apiPort` parameter
- Store port from settings

### 8. main.go

- Pass settings to client/core initialization
- Use configured ports

## Behavior Details

### AutoTestDelay

- When `true` (default): Auto-test all node delays on initial load
- When `false`: Skip auto-testing, user must press `t` or `T`

### AutoSelectBest

- When `true` (default): After all tests complete, switch to fastest node
- When `false`: Keep current selection
- Only triggers on initial proxy load, not on manual re-tests

### UseDefaultNode

- When `true`: Always use `DefaultNode` as the active proxy, ignore AutoSelectBest
- When `false`: Use AutoSelectBest logic to determine node
- User can set DefaultNode from Nodes tab: press `D` on selected node to mark as default
- If DefaultNode is empty and UseDefaultNode is true, fallback to AutoSelectBest behavior

### Subscription Naming

- When importing from clipboard or URL input, prompt for name (optional)
- If user skips naming, extract name from URL domain (e.g., `sub.example.com` → `example`)
- User can rename by pressing `e` on selected subscription in Config tab

### Multi-subscription Switching

1. User presses `j/k` to select subscription
2. User presses `enter` to switch
3. System:
   - Downloads new subscription
   - Rebuilds config.yaml with new nodes
   - Restarts clash core
   - Refreshes proxy list
   - Auto-tests if AutoTestDelay is enabled

### Port Editing

1. User presses `p` to edit proxy port or `a` to edit API port
2. Enter input mode (similar to subscription URL input)
3. Validate port number (1-65535)
4. Save to settings
5. Restart core to apply

### Subscription Info Parsing

Subscription URLs often contain info in the fragment:
```
https://example.com/sub#流量:234GB|过期:2025-05-01
```

Parse this after download and store in `Subscription.Traffic` and `Subscription.Expiry`.

Also check response headers:
- `subscription-userinfo: upload=xxx; download=xxx; total=xxx; expire=xxx`

## Migration

On first run with new version:
- Check for existing `subscriptions.txt`
- If exists, migrate content to `settings.json` as first subscription
- Delete `subscriptions.txt`

## File Changes Summary

| File | Changes |
|------|---------|
| `internal/settings/settings.go` | Add Subscription struct, update Settings |
| `internal/app/app.go` | Config tab redesign, subscription management |
| `internal/tui/nodes.go` | Sorting, colors, AutoSelectBest |
| `internal/tui/styles.go` | New color styles |
| `internal/config/config.go` | Remove single subscription functions |
| `internal/clash/core.go` | Dynamic port configuration |
| `internal/clash/client.go` | Dynamic API port |
| `main.go` | Wire up settings to components |

## Bug Fixes Included

- Fix `setIdx < 4` hardcoded bound → use dynamic length