# ClashTUI Feature Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enhance ClashTUI with UI improvements (colors, sorting), implement unused settings (AutoTestDelay, AutoSelectBest, ports), add multi-subscription support, and parse subscription traffic info.

**Architecture:** Extend settings module with Subscription type, update core/client for dynamic ports, enhance nodes.go with sorting/color logic, redesign app.go Config tab for subscription management.

**Tech Stack:** Go 1.21+, BubbleTea TUI framework, lipgloss styling

---

## Task 1: Update Settings Module

**Files:**
- Modify: `internal/settings/settings.go`

- [ ] **Step 1: Add Subscription struct**

```go
type Subscription struct {
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Traffic    string    `json:"traffic"`
	Expiry     string    `json:"expiry"`
	LastUpdate time.Time `json:"last_update"`
}
```

- [ ] **Step 2: Update Settings struct with new fields**

Replace the existing Settings struct:

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

- [ ] **Step 3: Update DefaultSettings**

```go
var DefaultSettings = Settings{
	Subscriptions:  []Subscription{},
	ActiveSubIdx:   0,
	AutoStart:      false,
	AutoTestDelay:  true,
	AutoSelectBest: true,
	UseDefaultNode: false,
	DefaultNode:    "",
	ProxyPort:      7890,
	APIPort:        9090,
}
```

- [ ] **Step 4: Update Load() to handle new fields**

Add defaults for new fields in Load():

```go
func Load() Settings {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return DefaultSettings
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultSettings
	}

	if s.ProxyPort == 0 {
		s.ProxyPort = DefaultSettings.ProxyPort
	}
	if s.APIPort == 0 {
		s.APIPort = DefaultSettings.APIPort
	}
	if s.Subscriptions == nil {
		s.Subscriptions = []Subscription{}
	}

	return s
}
```

- [ ] **Step 5: Add helper functions for subscription management**

```go
func GetActiveSubscription(s Settings) *Subscription {
	if len(s.Subscriptions) == 0 {
		return nil
	}
	if s.ActiveSubIdx < 0 || s.ActiveSubIdx >= len(s.Subscriptions) {
		return nil
	}
	return &s.Subscriptions[s.ActiveSubIdx]
}

func AddSubscription(s *Settings, name, url string) {
	sub := Subscription{
		Name: name,
		URL:  url,
	}
	s.Subscriptions = append(s.Subscriptions, sub)
	Save(*s)
}

func RemoveSubscription(s *Settings, idx int) {
	if idx < 0 || idx >= len(s.Subscriptions) {
		return
	}
	s.Subscriptions = append(s.Subscriptions[:idx], s.Subscriptions[idx+1:]...)
	if s.ActiveSubIdx >= len(s.Subscriptions) {
		s.ActiveSubIdx = len(s.Subscriptions) - 1
	}
	if s.ActiveSubIdx < 0 {
		s.ActiveSubIdx = 0
	}
	Save(*s)
}

func SwitchSubscription(s *Settings, idx int) {
	if idx < 0 || idx >= len(s.Subscriptions) {
		return
	}
	s.ActiveSubIdx = idx
	Save(*s)
}
```

- [ ] **Step 6: Build and verify**

Run: `go build -o clashtui .`

- [ ] **Step 7: Commit**

```bash
git add internal/settings/settings.go
git commit -m "feat(settings): add Subscription type and multi-subscription support"
```

---

## Task 2: Add Delay Color Styles

**Files:**
- Modify: `internal/tui/styles.go`

- [ ] **Step 1: Add delay color styles**

Add to the existing style definitions:

```go
var (
	DelayGreen = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	DelayYellow = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FBBF24"))

	DelayRed = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444"))

	DelayGray = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	SubActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	SubInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	SubHighlight = lipgloss.NewStyle().
		Bold(true)
)
```

- [ ] **Step 2: Add helper function for delay color**

```go
func DelayStyle(delay int) lipgloss.Style {
	if delay == 0 {
		return DelayGray
	}
	if delay < 100 {
		return DelayGreen
	}
	if delay < 300 {
		return DelayYellow
	}
	return DelayRed
}
```

- [ ] **Step 3: Build and verify**

Run: `go build -o clashtui .`

- [ ] **Step 4: Commit**

```bash
git add internal/tui/styles.go
git commit -m "feat(styles): add delay color styles and helper function"
```

---

## Task 3: Update Client for Dynamic API Port

**Files:**
- Modify: `internal/clash/client.go`

- [ ] **Step 1: Update Client struct and NewClient**

Replace hardcoded apiBase with configurable port:

```go
type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(apiPort int) *Client {
	if apiPort == 0 {
		apiPort = 9090
	}
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", apiPort),
		client:  &http.Client{Timeout: timeout},
	}
}
```

- [ ] **Step 2: Remove hardcoded apiBase constant**

Delete the line: `const apiBase = "http://127.0.0.1:9090"`

- [ ] **Step 3: Build and verify**

Run: `go build -o clashtui .`

Expected: Build fails because callers need updating (that's expected, will fix in Task 7)

- [ ] **Step 4: Commit**

```bash
git add internal/clash/client.go
git commit -m "feat(client): support dynamic API port"
```

---

## Task 4: Update Core for Dynamic Port Configuration

**Files:**
- Modify: `internal/clash/core.go`

- [ ] **Step 1: Update buildConfig to accept ports**

Change function signature:

```go
func buildConfig(nodes []string, proxyPort, apiPort int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("mixed-port: %d\nallow-lan: true\nmode: rule\nlog-level: info\nexternal-controller: 127.0.0.1:%d\n", proxyPort, apiPort))
	// ... rest of the function unchanged
}
```

- [ ] **Step 2: Update DownloadSubscription signature**

```go
func DownloadSubscription(subURL string, proxyPort, apiPort int) ([]byte, SubscriptionInfo, error) {
```

For now, return empty SubscriptionInfo:

```go
func DownloadSubscription(subURL string, proxyPort, apiPort int) ([]byte, SubscriptionInfo, error) {
	resp, err := http.Get(subURL)
	if err != nil {
		return nil, SubscriptionInfo{}, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, SubscriptionInfo{}, err
	}
	
	// ... existing decode logic ...
	
	configContent := buildConfig(decoded, proxyPort, apiPort)
	// ... save config ...
	
	return body, SubscriptionInfo{}, nil
}
```

And pass ports to buildConfig:

```go
configContent := buildConfig(decoded, proxyPort, apiPort)
```

- [ ] **Step 3: Build and verify**

Run: `go build -o clashtui .`

Expected: Build fails because callers need updating (expected, will fix in Task 7)

- [ ] **Step 4: Commit**

```bash
git add internal/clash/core.go
git commit -m "feat(core): support dynamic port configuration"
```

---

## Task 5: Update NodesModel with Sorting and Colors

**Files:**
- Modify: `internal/tui/nodes.go`

- [ ] **Step 1: Add sort by delay in MsgProxiesLoaded handler**

After setting m.proxies in the MsgProxiesLoaded case:

```go
case MsgProxiesLoaded:
	m.proxies = msg
	m.loading = false
	m.retries = 0
	if len(m.proxies) > 0 {
		// Sort by delay (fastest first, untested at end)
		sort.Slice(m.proxies, func(i, j int) bool {
			if m.proxies[i].Delay == 0 && m.proxies[j].Delay == 0 {
				return m.proxies[i].Name < m.proxies[j].Name
			}
			if m.proxies[i].Delay == 0 {
				return false
			}
			if m.proxies[j].Delay == 0 {
				return true
			}
			return m.proxies[i].Delay < m.proxies[j].Delay
		})
		// ... rest of existing code
	}
```

Add import: `"sort"`

- [ ] **Step 2: Add AutoSelectBest logic after all delays tested**

In MsgDelayTested handler, after testing completes:

```go
case MsgDelayTested:
	if msg.Index < len(m.proxies) {
		m.proxies[msg.Index].Delay = msg.Delay
	}
	if m.testing && msg.Index < len(m.proxies)-1 {
		return tea.Sequence(
			func() tea.Msg { return MsgTestProgress{Index: msg.Index + 1, Total: len(m.proxies)} },
			m.testDelay(msg.Index+1),
		)
	}
	m.testing = false
	// Re-sort after all tests complete
	sort.Slice(m.proxies, func(i, j int) bool {
		if m.proxies[i].Delay == 0 && m.proxies[j].Delay == 0 {
			return m.proxies[i].Name < m.proxies[j].Name
		}
		if m.proxies[i].Delay == 0 {
			return false
		}
		if m.proxies[j].Delay == 0 {
			return true
		}
		return m.proxies[i].Delay < m.proxies[j].Delay
	})
	return nil
```

- [ ] **Step 3: Add Settings reference and AutoSelectBest field**

Add to NodesModel struct:

```go
type NodesModel struct {
	proxies       []clash.ProxyInfo
	selected      int
	current       string
	loading       bool
	client        *clash.Client
	testing       bool
	testIdx       int
	retries       int
	autoSelectBest bool
	initialLoad   bool
}
```

Update NewNodesModel:

```go
func NewNodesModel(client *clash.Client, autoSelectBest bool) NodesModel {
	return NodesModel{
		client:        client,
		loading:       true,
		testing:       false,
		retries:       0,
		autoSelectBest: autoSelectBest,
		initialLoad:   true,
	}
}
```

- [ ] **Step 4: Auto-select logic (AutoSelectBest or UseDefaultNode)**

In MsgDelayTested handler, after the re-sort and before `return nil`:

```go
	m.testing = false
	// Re-sort after all tests complete
	sort.Slice(m.proxies, func(i, j int) bool {
		// ... same sort logic
	})
	
	// Handle initial selection logic
	if m.initialLoad && len(m.proxies) > 0 {
		// UseDefaultNode takes priority
		if m.useDefaultNode && m.defaultNode != "" {
			for _, p := range m.proxies {
				if p.Name == m.defaultNode {
					m.current = p.Name
					m.client.SwitchProxy(p.Name)
					for i, proxy := range m.proxies {
						if proxy.Name == p.Name {
							m.selected = i
							break
						}
					}
					break
				}
			}
		} else if m.autoSelectBest {
			// Auto-select fastest node
			for _, p := range m.proxies {
				if p.Delay > 0 {
					m.current = p.Name
					m.client.SwitchProxy(p.Name)
					for i, proxy := range m.proxies {
						if proxy.Name == p.Name {
							m.selected = i
							break
						}
					}
					break
				}
			}
		}
	}
	m.initialLoad = false
	return nil
```

- [ ] **Step 5: Add UseDefaultNode handling**

Add to NodesModel struct:

```go
useDefaultNode bool
defaultNode    string
settings       *settings.Settings
```

Update NewNodesModel:

```go
func NewNodesModel(client *clash.Client, autoSelectBest, useDefaultNode bool, defaultNode string, s *settings.Settings) NodesModel {
	return NodesModel{
		client:         client,
		loading:        true,
		testing:        false,
		retries:        0,
		autoSelectBest: autoSelectBest,
		initialLoad:    true,
		useDefaultNode: useDefaultNode,
		defaultNode:    defaultNode,
		settings:       s,
	}
}
```

Add handling for UseDefaultNode in MsgProxiesLoaded (instead of AutoSelectBest):

```go
if m.initialLoad && len(m.proxies) > 0 {
	// If UseDefaultNode is enabled, use the default node
	if m.useDefaultNode && m.defaultNode != "" {
		for _, p := range m.proxies {
			if p.Name == m.defaultNode {
				m.current = p.Name
				m.client.SwitchProxy(p.Name)
				// ... update selected index
				break
			}
		}
	}
	m.initialLoad = false
}
```

Add key handler for "D" to set default node:

```go
case "D":
	if len(m.proxies) > 0 {
		name := m.proxies[m.selected].Name
		m.defaultNode = name
		if m.settings != nil {
			m.settings.DefaultNode = name
			m.settings.UseDefaultNode = true
			settings.Save(*m.settings)
		}
		return func() tea.Msg { return MsgLogLine("Default node set: " + name) }
	}
```

- [ ] **Step 6: Update View() to use color styles**

Replace the delay formatting line:

```go
delay := fmt.Sprintf("%dms", p.Delay)
if p.Delay == 0 {
	delay = "-"
}

// Apply color style
delayStyled := tui.DelayStyle(p.Delay).Render(delay)
```

- [ ] **Step 7: Add SetDefaultNode method to NodesModel**

```go
func (m *NodesModel) SetDefaultNode(useDefault bool, defaultNode string) {
	m.useDefaultNode = useDefault
	m.defaultNode = defaultNode
}
```

And use `delayStyled` in the output line instead of `delay`.

- [ ] **Step 8: Build and verify**

Run: `go build -o clashtui .`

Expected: Build fails because NewNodesModel signature changed (expected, will fix in Task 7)

- [ ] **Step 7: Commit**

```bash
git add internal/tui/nodes.go
git commit -m "feat(nodes): add sorting, color coding, and AutoSelectBest"
```

---

## Task 6: Redesign Config Tab for Subscription Management

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Add new state fields to Model**

```go
type Model struct {
	tab       int
	nodes     tui.NodesModel
	logs      *tui.LogsModel
	core      *clash.Core
	client    *clash.Client
	running   bool
	err       string
	subInput  bool
	subURL    string
	settings  settings.Settings
	setIdx    int
	subIdx    int    // subscription list index
	inputMode string // "sub_url", "sub_name", "proxy_port", "api_port", ""
	inputBuf  string
}
```

- [ ] **Step 2: Update New() to pass settings to NodesModel**

```go
func New() Model {
	s := settings.Load()
	client := clash.NewClient(s.APIPort)
	core := clash.NewCore()
	nodes := tui.NewNodesModel(client, s.AutoSelectBest, s.UseDefaultNode, s.DefaultNode, &s)
	logs := tui.NewLogsModel()

	running := client.IsConnected()

	return Model{
		tab:      0,
		nodes:    nodes,
		logs:     logs,
		core:     core,
		client:   client,
		running:  running,
		settings: s,
	}
}
```

- [ ] **Step 3: Add subscription list handling in Config tab**

In Update(), add handling for Config tab (tab 1):

```go
if m.tab == 1 {
	// Handle input mode first
	if m.inputMode != "" {
		return m.handleInputMode(msg)
	}
	// Handle settings keys
	return m.handleConfigKeys(msg)
}
```

- [ ] **Step 4: Add handleConfigKeys for Config tab navigation**

```go
func (m Model) handleConfigKeys(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "j", "down":
			// Navigate between subscription list and settings
			if m.subIdx < len(m.settings.Subscriptions) {
				m.subIdx++
			} else if m.setIdx < 4 {
				m.setIdx++
			}
			return m, nil
		case "k", "up":
			if m.setIdx > 0 {
				m.setIdx--
			} else if m.subIdx > 0 {
				m.subIdx--
			}
			return m, nil
		case "enter":
			return m.handleConfigEnter()
		case "d":
			if m.subIdx < len(m.settings.Subscriptions) {
				settings.RemoveSubscription(&m.settings, m.subIdx)
				return m, func() tea.Msg { return tui.MsgLogLine("Subscription deleted") }
			}
			return m, nil
		case "e":
			if m.subIdx < len(m.settings.Subscriptions) {
				m.inputMode = "sub_name"
				m.inputBuf = m.settings.Subscriptions[m.subIdx].Name
				return m, nil
			}
			return m, nil
		case "p":
			m.inputMode = "proxy_port"
			m.inputBuf = fmt.Sprintf("%d", m.settings.ProxyPort)
			return m, nil
		case "a":
			m.inputMode = "api_port"
			m.inputBuf = fmt.Sprintf("%d", m.settings.APIPort)
			return m, nil
		}
	}
	return m, nil
}
```

- [ ] **Step 5: Add handleConfigEnter for actions**

```go
func (m Model) handleConfigEnter() (tea.Model, tea.Cmd) {
	// If in subscription list area
	if m.subIdx < len(m.settings.Subscriptions) {
		// Switch to this subscription
		if m.subIdx != m.settings.ActiveSubIdx {
			settings.SwitchSubscription(&m.settings, m.subIdx)
			return m, m.switchSubscription()
		}
		return m, nil
	}
	// If in "Add subscription" row
	if m.subIdx == len(m.settings.Subscriptions) {
		m.inputMode = "sub_url"
		m.inputBuf = ""
		return m, nil
	}
	// If in settings area - toggle boolean settings
	settingsIdx := m.subIdx - len(m.settings.Subscriptions) - 1
	switch settingsIdx {
	case 0:
		m.settings.AutoStart = !m.settings.AutoStart
		settings.Save(m.settings)
		return m.handleAutoStartToggle()
	case 1:
		m.settings.AutoTestDelay = !m.settings.AutoTestDelay
		settings.Save(m.settings)
		return m, nil
	case 2:
		m.settings.AutoSelectBest = !m.settings.AutoSelectBest
		settings.Save(m.settings)
		return m, nil
	case 3:
		m.settings.UseDefaultNode = !m.settings.UseDefaultNode
		settings.Save(m.settings)
		// Update NodesModel
		m.nodes.SetDefaultNode(m.settings.UseDefaultNode, m.settings.DefaultNode)
		return m, nil
	}
	return m, nil
}
```

- [ ] **Step 6: Add handleAutoStartToggle for systemd service**

```go
func (m Model) handleAutoStartToggle() (tea.Model, tea.Cmd) {
	home, _ := os.UserHomeDir()
	serviceDir := filepath.Join(home, ".config", "systemd", "user")
	serviceFile := filepath.Join(serviceDir, "clashtui.service")

	if m.settings.AutoStart {
		os.MkdirAll(serviceDir, 0755)
		serviceContent := `[Unit]
Description=ClashTUI - Clash proxy manager
After=network.target

[Service]
Type=simple
ExecStart=` + filepath.Join(home, ".local", "bin", "clashtui") + ` --daemon
Restart=on-failure

[Install]
WantedBy=default.target`
		os.WriteFile(serviceFile, []byte(serviceContent), 0644)
		exec.Command("systemctl", "--user", "enable", "clashtui.service").Run()
		exec.Command("systemctl", "--user", "daemon-reload").Run()
		return m, func() tea.Msg { return tui.MsgLogLine("Auto start enabled") }
	} else {
		exec.Command("systemctl", "--user", "disable", "clashtui.service").Run()
		os.Remove(serviceFile)
		exec.Command("systemctl", "--user", "daemon-reload").Run()
		return m, func() tea.Msg { return tui.MsgLogLine("Auto start disabled") }
	}
}
```

- [ ] **Step 7: Add handleInputMode for text input**

```go
func (m Model) handleInputMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "enter":
			return m.finishInputMode()
		case "esc":
			m.inputMode = ""
			m.inputBuf = ""
			return m, nil
		case "backspace":
			if len(m.inputBuf) > 0 {
				m.inputBuf = m.inputBuf[:len(m.inputBuf)-1]
			}
			return m, nil
		default:
			if len(k) == 1 {
				m.inputBuf += k
			}
			return m, nil
		}
	}
	return m, nil
}
```

- [ ] **Step 8: Add finishInputMode to process input**

```go
func (m Model) finishInputMode() (tea.Model, tea.Cmd) {
	switch m.inputMode {
	case "sub_url":
		if m.inputBuf == "" {
			m.inputMode = ""
			return m, nil
		}
		m.inputMode = "sub_name"
		m.subURL = m.inputBuf
		m.inputBuf = ""
		return m, nil
	case "sub_name":
		name := m.inputBuf
		if name == "" {
			// Extract name from URL domain
			u, err := url.Parse(m.subURL)
			if err == nil {
				name = u.Hostname()
			} else {
				name = "Subscription"
			}
		}
		settings.AddSubscription(&m.settings, name, m.subURL)
		m.settings.ActiveSubIdx = len(m.settings.Subscriptions) - 1
		m.inputMode = ""
		return m, m.downloadAndStartSubscription()
	case "proxy_port":
		port, err := strconv.Atoi(m.inputBuf)
		if err == nil && port > 0 && port <= 65535 {
			m.settings.ProxyPort = port
			settings.Save(m.settings)
			m.inputMode = ""
			return m, func() tea.Msg { return tui.MsgLogLine("Proxy port updated, restart core to apply") }
		}
		m.err = "Invalid port"
		return m, nil
	case "api_port":
		port, err := strconv.Atoi(m.inputBuf)
		if err == nil && port > 0 && port <= 65535 {
			m.settings.APIPort = port
			settings.Save(m.settings)
			m.client = clash.NewClient(port)
			m.inputMode = ""
			return m, func() tea.Msg { return tui.MsgLogLine("API port updated, restart core to apply") }
		}
		m.err = "Invalid port"
		return m, nil
	}
	m.inputMode = ""
	return m, nil
}
```

Add imports: `"net/url", "strconv"`

- [ ] **Step 9: Add subscription switching command**

```go
func (m Model) switchSubscription() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			m.core.Stop()
			return tui.MsgLogLine("Switching subscription...")
		},
		m.downloadAndStartSubscription(),
	)
}

func (m Model) downloadAndStartSubscription() tea.Cmd {
	return func() tea.Msg {
		sub := settings.GetActiveSubscription(m.settings)
		if sub == nil {
			return tui.MsgLogLine("No subscription selected")
		}
		
		m.logs.AddLine("Downloading: " + sub.URL)
_, _, err := clash.DownloadSubscription(sub.URL, m.settings.ProxyPort, m.settings.APIPort)
		if err != nil {
			m.logs.AddLine("Error: " + err.Error())
			m.err = err.Error()
			return nil
		}
		
		m.logs.AddLine("Subscription updated")
		
		if !m.core.IsInstalled() {
			m.logs.AddLine("Installing core...")
			m.core.Install()
			m.core.DownloadGeoData()
		}
		
		m.core.Stop()
		m.core.Start()
		proxy.SetSystemProxy()
		m.running = true
		m.logs.AddLine("Core started")
		
		m.err = ""
		return tui.MsgRefresh{}
	}
}
```

- [ ] **Step 10: Update configView() for new layout**

```go
func (m Model) configView() string {
	var b strings.Builder
	b.WriteString("\n")
	
	// Subscription list
	b.WriteString("  Subscriptions:\n")
	for i, sub := range m.settings.Subscriptions {
		prefix := "  "
		if i == m.subIdx {
			prefix = "> "
		}
		style := tui.SubInactive
		if i == m.settings.ActiveSubIdx {
			style = tui.SubActive
		}
		if i == m.subIdx {
			style = tui.SubHighlight
		}
		
		name := style.Render(sub.Name)
		traffic := "---"
		if sub.Traffic != "" {
			traffic = sub.Traffic
		}
		expiry := "---"
		if sub.Expiry != "" {
			expiry = "exp:" + sub.Expiry
		}
		b.WriteString(fmt.Sprintf("%s[%d] %s  %s  %s\n", prefix, i+1, name, traffic, expiry))
	}
	
	// Add subscription row
	prefix := "  "
	if m.subIdx == len(m.settings.Subscriptions) {
		prefix = "> "
	}
	b.WriteString(fmt.Sprintf("%s[+] Add subscription\n", prefix))
	
	// Settings section
	b.WriteString("\n  Settings:\n")
	settingsIdx := m.subIdx - len(m.settings.Subscriptions) - 1
	opts := []struct {
		name  string
		value bool
	}{
		{"Auto start on boot", m.settings.AutoStart},
		{"Auto test delay on load", m.settings.AutoTestDelay},
		{"Auto select fastest node", m.settings.AutoSelectBest},
		{"Use default node", m.settings.UseDefaultNode},
	}
	
	for i, opt := range opts {
		prefix := "  "
		if i == settingsIdx {
			prefix = "> "
		}
		check := "[ ]"
		if opt.value {
			check = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, check, opt.name))
	}
	
	// Port settings
	b.WriteString(fmt.Sprintf("\n  Proxy port: %d  |  API port: %d\n", m.settings.ProxyPort, m.settings.APIPort))
	
	// Help text
	if m.inputMode != "" {
		b.WriteString(fmt.Sprintf("\n  Enter %s: %s\n  enter: submit | esc: cancel\n", m.inputMode, m.inputBuf))
	} else {
		b.WriteString("\n  j/k: navigate | enter: action | d: delete | e: rename\n")
		b.WriteString("  p: edit proxy port | a: edit api port\n")
		b.WriteString("  c: import clipboard | r: refresh\n")
	}
	
	return b.String()
}
```

- [ ] **Step 11: Update toggleCore to use dynamic ports**

```go
func (m Model) toggleCore() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			m.core.Stop()
			proxy.UnsetSystemProxy()
			return tui.MsgLogLine("Stopped existing core")
		},
		func() tea.Msg {
			sub := settings.GetActiveSubscription(m.settings)
			if sub == nil {
				return tui.MsgLogLine("Error: no subscription, add one first")
			}
_, _, err := clash.DownloadSubscription(sub.URL, m.settings.ProxyPort, m.settings.APIPort)
			if err != nil {
				return tui.MsgLogLine("Error: " + err.Error())
			}
			return tui.MsgLogLine("Subscription updated")
		},
		func() tea.Msg {
			if !m.core.IsInstalled() {
				m.core.Install()
				m.core.DownloadGeoData()
			}
			err := m.core.Start()
			if err != nil {
				return tui.MsgLogLine("Error starting: " + err.Error())
			}
			proxy.SetSystemProxy()
			return tui.MsgLogLine("Core started, proxy enabled")
		},
		func() tea.Msg {
			return tui.MsgRefresh{}
		},
	)
}
```

- [ ] **Step 12: Remove old handleSettingsKeys**

Delete the old `handleSettingsKeys` function (replaced by handleConfigKeys).

- [ ] **Step 13: Update importFromClipboard**

```go
func (m Model) importFromClipboard() tea.Cmd {
	return func() tea.Msg {
		url, err := clipboard.Read()
		if err != nil {
			m.err = "clipboard: " + err.Error()
			return nil
		}
		if len(url) < 4 || url[:4] != "http" {
			m.err = "no valid URL in clipboard"
			return nil
		}
		m.subURL = url
		m.inputMode = "sub_name"
		m.inputBuf = ""
		return nil
	}
}
```

- [ ] **Step 14: Build and verify**

Run: `go build -o clashtui .`

- [ ] **Step 15: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): redesign Config tab with subscription management"
```

---

## Task 7: Wire Settings in Main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Update client initialization with API port**

```go
func main() {
	args := os.Args[1:]
	s := settings.Load()
	
	cleanStaleProxySettings(s)
	
	// ... rest of main
}

func cleanStaleProxySettings(s settings.Settings) {
	client := clash.NewClient(s.APIPort)
	if !client.IsConnected() {
		proxy.UnsetSystemProxy()
	}
}
```

- [ ] **Step 2: Update printStatus**

```go
func printStatus() {
	s := settings.Load()
	client := clash.NewClient(s.APIPort)
	connected := client.IsConnected()
	// ... rest unchanged
}
```

- [ ] **Step 3: Update runDaemon**

```go
func runDaemon() {
	s := settings.Load()
	
	acquired, err := singleinstance.Acquire()
	// ... error handling unchanged
	
	defer singleinstance.Release()
	defer cleanupOnExit(s)
	
	if len(s.Subscriptions) > 0 {
		sub := settings.GetActiveSubscription(s)
		if sub != nil && sub.URL != "" {
			core := clash.NewCore()
			if !core.IsInstalled() {
				core.Install()
				core.DownloadGeoData()
			}
			_, _, _ = clash.DownloadSubscription(sub.URL, s.ProxyPort, s.APIPort)
			core.Start()
			proxy.SetSystemProxy()
		}
	}
	
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}

func cleanupOnExit(s settings.Settings) {
	core := clash.NewCore()
	core.Stop()
	proxy.UnsetSystemProxy()
}
```

- [ ] **Step 4: Update stopAll**

```go
func stopAll() {
	s := settings.Load()
	
	daemonPid, err := singleinstance.ReadPID()
	if err == nil && daemonPid > 0 {
		process, _ := os.FindProcess(daemonPid)
		process.Signal(syscall.SIGTERM)
		for i := 0; i < 10; i++ {
			if process.Signal(syscall.Signal(0)) != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	core := clash.NewCore()
	core.Stop()
	proxy.UnsetSystemProxy()
	fmt.Println("Stopped")
}
```

- [ ] **Step 5: Update toggleProxy**

```go
func toggleProxy() {
	s := settings.Load()
	client := clash.NewClient(s.APIPort)
	connected := client.IsConnected()
	
	if connected {
		stopAll()
	} else {
		sub := settings.GetActiveSubscription(s)
		if sub == nil || sub.URL == "" {
			fmt.Println("No subscription configured")
			os.Exit(1)
		}
		core := clash.NewCore()
		if !core.IsInstalled() {
			core.Install()
			core.DownloadGeoData()
		}
		_, _, _ = clash.DownloadSubscription(sub.URL, s.ProxyPort, s.APIPort)
		core.Start()
		proxy.SetSystemProxy()
		fmt.Println("Started")
	}
}
```

- [ ] **Step 6: Build and verify**

Run: `go build -o clashtui .`

- [ ] **Step 7: Commit**

```bash
git add main.go
git commit -m "feat(main): wire settings to all components"
```

---

## Task 8: Add Migration for Old Subscription

**Files:**
- Modify: `internal/settings/settings.go`
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add migration function in settings.go**

```go
func MigrateFromOldFormat() {
	// Check if old subscription file exists
	oldPath := config.GetOldSubscriptionPath()
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return // No old file to migrate
	}
	
	url := strings.TrimSpace(string(data))
	if url == "" {
		return
	}
	
	// Check if settings already has subscriptions
	s := Load()
	if len(s.Subscriptions) > 0 {
		// Already migrated, delete old file
		os.Remove(oldPath)
		return
	}
	
	// Add as first subscription
	AddSubscription(&s, "Migrated", url)
	s.ActiveSubIdx = 0
	Save(s)
	
	// Delete old file
	os.Remove(oldPath)
}
```

Add import: `"strings"`

- [ ] **Step 2: Add GetOldSubscriptionPath in config.go**

```go
func GetOldSubscriptionPath() string {
	return subscriptionPath
}
```

- [ ] **Step 3: Call migration in main.go**

Add at the beginning of main():

```go
func main() {
	settings.MigrateFromOldFormat()
	// ... rest of main
}
```

- [ ] **Step 4: Build and verify**

Run: `go build -o clashtui .`

- [ ] **Step 5: Commit**

```bash
git add internal/settings/settings.go internal/config/config.go main.go
git commit -m "feat: add migration from old subscription format"
```

---

## Task 9: Parse Subscription Traffic Info

**Files:**
- Modify: `internal/clash/core.go`

- [ ] **Step 1: Add parseSubscriptionInfo function**

```go
type SubscriptionInfo struct {
	Traffic string
	Expiry  string
}

func parseSubscriptionInfo(urlStr string, headers http.Header) SubscriptionInfo {
	info := SubscriptionInfo{}
	
	// Try to parse from URL fragment
	u, err := url.Parse(urlStr)
	if err == nil && u.Fragment != "" {
		// Common formats: 流量:234GB|过期:2025-05-01 or #traffic=234GB&expire=2025-05-01
		fragment := u.Fragment
		
		// Chinese format
		if strings.Contains(fragment, "流量") {
			parts := strings.Split(fragment, "|")
			for _, p := range parts {
				if strings.HasPrefix(p, "流量:") || strings.HasPrefix(p, "流量=") {
					info.Traffic = strings.TrimPrefix(strings.TrimPrefix(p, "流量:"), "流量=")
				}
				if strings.HasPrefix(p, "过期:") || strings.HasPrefix(p, "过期=") || 
				   strings.HasPrefix(p, "到期:") || strings.HasPrefix(p, "到期=") {
					info.Expiry = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(p, "过期:"), "过期="), "到期:"), "到期=")
				}
			}
		}
		
		// English format
		if strings.Contains(fragment, "traffic") {
			params := strings.Split(fragment, "&")
			for _, p := range params {
				if strings.HasPrefix(p, "traffic=") {
					info.Traffic = strings.TrimPrefix(p, "traffic=")
				}
				if strings.HasPrefix(p, "expire=") || strings.HasPrefix(p, "expiry=") {
					info.Expiry = strings.TrimPrefix(strings.TrimPrefix(p, "expire="), "expiry=")
				}
			}
		}
	}
	
	// Try to parse from response headers (subscription-userinfo)
	userInfo := headers.Get("subscription-userinfo")
	if userInfo != "" {
		// Format: upload=xxx; download=xxx; total=xxx; expire=xxx
		parts := strings.Split(userInfo, ";")
		var upload, download, total int64
		var expire int64
		
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "upload=") {
				upload, _ = strconv.ParseInt(strings.TrimPrefix(p, "upload="), 10, 64)
			}
			if strings.HasPrefix(p, "download=") {
				download, _ = strconv.ParseInt(strings.TrimPrefix(p, "download="), 10, 64)
			}
			if strings.HasPrefix(p, "total=") {
				total, _ = strconv.ParseInt(strings.TrimPrefix(p, "total="), 10, 64)
			}
			if strings.HasPrefix(p, "expire=") {
				expire, _ = strconv.ParseInt(strings.TrimPrefix(p, "expire="), 10, 64)
			}
		}
		
		if total > 0 {
			used := upload + download
			usedGB := used / 1024 / 1024 / 1024
			totalGB := total / 1024 / 1024 / 1024
			info.Traffic = fmt.Sprintf("%dGB/%dGB", usedGB, totalGB)
		}
		
		if expire > 0 {
			expiryTime := time.Unix(expire, 0)
			info.Expiry = expiryTime.Format("2006-01-02")
		}
	}
	
	return info
}
```

Add imports: `"net/url", "strconv", "time"`

- [ ] **Step 2: Update DownloadSubscription to return info**

```go
func DownloadSubscription(subURL string, proxyPort, apiPort int) ([]byte, SubscriptionInfo, error) {
	resp, err := http.Get(subURL)
	if err != nil {
		return nil, SubscriptionInfo{}, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, SubscriptionInfo{}, err
	}
	
	info := parseSubscriptionInfo(subURL, resp.Header)
	
	// ... existing decode and build logic
	
	return body, info, nil
}
```

- [ ] **Step 3: Update callers to handle new return type**

This will require updating `internal/app/app.go` to capture and save the info:

```go
func (m Model) downloadAndStartSubscription() tea.Cmd {
	return func() tea.Msg {
		sub := settings.GetActiveSubscription(m.settings)
		if sub == nil {
			return tui.MsgLogLine("No subscription selected")
		}
		
		m.logs.AddLine("Downloading: " + sub.URL)
		_, info, err := clash.DownloadSubscription(sub.URL, m.settings.ProxyPort, m.settings.APIPort)
		if err != nil {
			m.logs.AddLine("Error: " + err.Error())
			m.err = err.Error()
			return nil
		}
		
		// Update subscription info
		if info.Traffic != "" {
			m.settings.Subscriptions[m.settings.ActiveSubIdx].Traffic = info.Traffic
		}
		if info.Expiry != "" {
			m.settings.Subscriptions[m.settings.ActiveSubIdx].Expiry = info.Expiry
		}
		m.settings.Subscriptions[m.settings.ActiveSubIdx].LastUpdate = time.Now()
		settings.Save(m.settings)
		
		// ... rest unchanged
	}
}
```

Add import: `"time"`

- [ ] **Step 4: Build and verify**

Run: `go build -o clashtui .`

- [ ] **Step 5: Commit**

```bash
git add internal/clash/core.go internal/app/app.go
git commit -m "feat: parse and display subscription traffic info"
```

---

## Task 10: Final Verification and Testing

- [ ] **Step 1: Build final version**

Run: `go build -o clashtui .`

- [ ] **Step 2: Run basic smoke test**

Run the TUI:
```bash
./clashtui
```

Test navigation:
- Press `2` to go to Config tab
- Press `j/k` to navigate subscription list
- Press `3` to go to Logs tab

- [ ] **Step 3: Test subscription import**

- Press `c` with a subscription URL in clipboard
- Enter subscription name
- Verify core starts and nodes load

- [ ] **Step 4: Test node sorting and colors**

- Verify nodes are sorted by delay
- Verify delay colors are applied (green/yellow/red/gray)

- [ ] **Step 5: Test AutoSelectBest**

- With AutoSelectBest enabled, verify fastest node is auto-selected after tests

- [ ] **Step 6: Test port configuration**

- Press `p` to edit proxy port
- Press `a` to edit API port
- Enter new port numbers
- Verify settings saved

- [ ] **Step 7: Final commit if needed**

If any fixes were made:
```bash
git add -A
git commit -m "fix: final adjustments after testing"
```

---

## Summary

This plan implements:
1. Multi-subscription support with switch functionality
2. Delay color coding and node sorting
3. AutoSelectBest, AutoTestDelay, port configuration
4. Subscription traffic/expiry info parsing
5. Migration from old subscription format

All changes maintain backward compatibility and follow existing code patterns.