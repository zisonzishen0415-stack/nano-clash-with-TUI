package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"clashtui/internal/clash"
	"clashtui/internal/clipboard"
	"clashtui/internal/proxy"
	"clashtui/internal/settings"
	"clashtui/internal/tui"
)

type Model struct {
	tab          int
	nodes        tui.NodesModel
	logs         *tui.LogsModel
	core         *clash.Core
	client       *clash.Client
	running      bool
	err          string
	settings     settings.Settings
	cursorIdx    int
	inputMode    string
	inputBuf     string
	addType      string
	urlBuf       string
	currentAction string  // 当前正在进行的操作
}

func New() Model {
	s := settings.Load()
	client := clash.NewClient(s.APIPort)
	core := clash.NewCore()
	nodes := tui.NewNodesModel(client)
	nodes.SetAutoSelectBest(s.AutoSelectBest)
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

func (m Model) Init() tea.Cmd {
	return m.nodes.Init()
}

// startAction 记录操作开始状态并写入日志
func (m *Model) startAction(action string) {
	m.currentAction = action
	m.logs.AddLine("→ " + action)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.inputMode != "" {
		return m.handleInputMode(msg)
	}

	switch msg := msg.(type) {
	case tui.MsgProxiesLoaded:
		cmd := m.nodes.Update(msg)
		if len(msg) > 0 {
			m.logs.AddLine(fmt.Sprintf("✓ Loaded %d proxies", len(msg)))
		}
		return m, cmd

	case tui.MsgProxySwitched:
		m.logs.AddLine(fmt.Sprintf("✓ Switched to: %s", string(msg)))
		cmd := m.nodes.Update(msg)
		return m, cmd

	case tui.MsgDelayTested, tui.MsgRetryLoad, tui.MsgTestProgress:
		cmd := m.nodes.Update(msg)
		return m, cmd

	case tui.MsgStopCore:
		m.startAction("Stopping core")
		m.core.Stop()
		proxy.UnsetSystemProxy()
		m.running = false
		return m, func() tea.Msg { return tui.MsgLogLine("✓ Core stopped, proxy disabled") }

	case tui.MsgLogLine:
		m.logs.Update(msg)
		m.currentAction = ""
		return m, nil

	case tui.MsgRefresh:
		if msg.Traffic != "" || msg.Expiry != "" {
			if m.settings.ActiveSubIdx >= 0 && m.settings.ActiveSubIdx < len(m.settings.Subscriptions) {
				m.settings.Subscriptions[m.settings.ActiveSubIdx].Traffic = msg.Traffic
				m.settings.Subscriptions[m.settings.ActiveSubIdx].Expiry = msg.Expiry
				m.settings.Subscriptions[m.settings.ActiveSubIdx].LastUpdate = time.Now()
				settings.Save(m.settings)
			}
		}
		m.running = true
		cmd := m.nodes.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		k := msg.String()

		switch k {
		case "h", "left":
			if m.tab > 0 {
				m.tab--
			} else {
				m.tab = 2
			}
			return m, nil
		case "l", "right":
			m.tab = (m.tab + 1) % 3
			return m, nil
		case "1":
			m.tab = 0
			return m, nil
		case "2":
			m.tab = 1
			return m, nil
		case "3":
			m.tab = 2
			return m, nil
		}

		switch k {
		case "x":
			m.startAction("Stopping core and clearing proxy")
			m.core.Stop()
			proxy.UnsetSystemProxy()
			m.running = false
			return m, func() tea.Msg { return tui.MsgLogLine("✓ Core stopped, proxy disabled") }
		case "s":
			m.startAction("Adding subscription")
			m.addType = "subscription"
			m.inputMode = "url"
			m.inputBuf = ""
			m.urlBuf = ""
			return m, nil
		case "c":
			m.startAction("Importing from clipboard")
			return m, m.importFromClipboard()
		case "r":
			m.startAction("Restarting core")
			return m, m.toggleCore()
		case "q", "ctrl+c":
			m.startAction("Exiting")
			return m, tea.Quit
		}

		if m.tab == 0 {
			cmd := m.nodes.Update(msg)
			return m, cmd
		}

		if m.tab == 1 {
			return m.handleConfigKeys(msg)
		}
	}

	return m, nil
}

func (m Model) handleConfigKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	totalRows := len(m.settings.Subscriptions) + 1 + 1 + 3 + 1 + 2

	switch k {
	case "j", "down":
		if m.cursorIdx < totalRows-1 {
			m.cursorIdx++
		}
		return m, nil
	case "k", "up":
		if m.cursorIdx > 0 {
			m.cursorIdx--
		}
		return m, nil
	case "enter":
		return m.handleConfigEnter()
	case "d":
		if m.cursorIdx < len(m.settings.Subscriptions) {
			settings.RemoveSubscription(&m.settings, m.cursorIdx)
			if m.cursorIdx >= len(m.settings.Subscriptions) && len(m.settings.Subscriptions) > 0 {
				m.cursorIdx = len(m.settings.Subscriptions)
			}
			return m, func() tea.Msg { return tui.MsgLogLine("Deleted subscription") }
		}
		return m, nil
	case "D":
		m.settings.Subscriptions = []settings.Subscription{}
		m.settings.ActiveSubIdx = 0
		settings.Save(m.settings)
		m.cursorIdx = 0
		return m, func() tea.Msg { return tui.MsgLogLine("Deleted all subscriptions") }
	}
	return m, nil
}

func (m Model) handleConfigEnter() (tea.Model, tea.Cmd) {
	row := m.cursorIdx

	if row < len(m.settings.Subscriptions) {
		if row != m.settings.ActiveSubIdx {
			settings.SwitchSubscription(&m.settings, row)
			return m, m.switchSubscription()
		}
		return m, nil
	}

	addRow := len(m.settings.Subscriptions)
	if row == addRow {
		m.inputMode = "add_type"
		m.inputBuf = ""
		return m, nil
	}

	settingsStart := addRow + 1
	settingsIdx := row - settingsStart

	if settingsIdx >= 0 && settingsIdx < 3 {
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
			m.nodes.SetAutoSelectBest(m.settings.AutoSelectBest)
			return m, nil
		}
	}

	portsStart := settingsStart + 3
	portIdx := row - portsStart

	if portIdx == 0 {
		m.inputMode = "proxy_port"
		m.inputBuf = fmt.Sprintf("%d", m.settings.ProxyPort)
		return m, nil
	}
	if portIdx == 1 {
		m.inputMode = "api_port"
		m.inputBuf = fmt.Sprintf("%d", m.settings.APIPort)
		return m, nil
	}

	return m, nil
}

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
	}
	exec.Command("systemctl", "--user", "disable", "clashtui.service").Run()
	os.Remove(serviceFile)
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return m, func() tea.Msg { return tui.MsgLogLine("Auto start disabled") }
}

func (m Model) handleInputMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	k := keyMsg.String()

	switch k {
	case "enter":
		return m.finishInputMode()
	case "esc":
		m.inputMode = ""
		m.inputBuf = ""
		m.urlBuf = ""
		return m, nil
	case "backspace":
		if len(m.inputBuf) > 0 {
			m.inputBuf = m.inputBuf[:len(m.inputBuf)-1]
		}
		return m, nil
	case "c":
		if m.inputMode == "url" {
			url, err := clipboard.Read()
			if err != nil {
				m.err = "clipboard: " + err.Error()
				return m, nil
			}
			if len(url) < 4 || url[:4] != "http" {
				m.err = "no valid URL in clipboard"
				return m, nil
			}
			m.inputBuf = url
			m.err = ""
			return m, nil
		}
		m.inputBuf += "c"
		return m, nil
	default:
		if len(k) == 1 {
			m.inputBuf += k
		}
		return m, nil
	}
}

func (m Model) finishInputMode() (tea.Model, tea.Cmd) {
	switch m.inputMode {
	case "add_type":
		if m.inputBuf == "1" {
			m.addType = "subscription"
			m.inputMode = "url"
			m.inputBuf = ""
			return m, nil
		} else if m.inputBuf == "2" {
			m.addType = "subscription"
			url, err := clipboard.Read()
			if err != nil {
				m.err = "clipboard: " + err.Error()
				m.inputMode = ""
				return m, nil
			}
			if len(url) < 4 || url[:4] != "http" {
				m.err = "no valid URL in clipboard"
				m.inputMode = ""
				return m, nil
			}
			m.urlBuf = url
			m.inputMode = "name"
			m.inputBuf = ""
			m.err = ""
			return m, nil
		} else if m.inputBuf == "3" {
			m.addType = "manual"
			m.inputMode = "name"
			m.inputBuf = ""
			return m, nil
		}
		m.inputMode = ""
		return m, nil
	case "url":
		m.urlBuf = m.inputBuf
		m.inputMode = "name"
		m.inputBuf = ""
		return m, nil
	case "name":
		name := m.inputBuf
		if name == "" {
			name = "Subscription"
		}
		if m.addType == "subscription" {
			settings.AddSubscription(&m.settings, name, m.urlBuf)
			m.logs.AddLine("Added subscription: " + name)
			m.inputMode = ""
			m.inputBuf = ""
			m.urlBuf = ""
			return m, m.downloadSub(name, m.urlBuf)
		}
		m.inputMode = ""
		m.inputBuf = ""
		m.urlBuf = ""
		return m, nil
	case "proxy_port":
		port, err := strconv.Atoi(m.inputBuf)
		if err == nil && port > 0 && port <= 65535 {
			m.settings.ProxyPort = port
			settings.Save(m.settings)
		}
		m.inputMode = ""
		return m, nil
	case "api_port":
		port, err := strconv.Atoi(m.inputBuf)
		if err == nil && port > 0 && port <= 65535 {
			m.settings.APIPort = port
			settings.Save(m.settings)
			m.client = clash.NewClient(port)
		}
		m.inputMode = ""
		return m, nil
	}
	m.inputMode = ""
	return m, nil
}

func (m Model) View() string {
	tabs := ""
	tabNames := []string{"Nodes", "Config", "Logs"}
	for i, name := range tabNames {
		if i == m.tab {
			tabs += tui.TabActive.Render("[" + name + "]")
		} else {
			tabs += tui.TabInactive.Render(" " + name + " ")
		}
		tabs += " "
	}

	var content string
	switch m.tab {
	case 0:
		content = m.nodes.View()
		if content == "" {
			content = "  No proxies loaded.\n\n  Press 2/l -> Config tab, then 'c' to import from clipboard"
		}
	case 1:
		content = m.configView()
	case 2:
		content = m.logs.View()
	}

	if m.inputMode != "" {
		content = m.renderInputMode()
	}

	status := fmt.Sprintf("\n\n  Core: %s | Current: %s",
		m.coreStatus(), m.nodes.GetCurrent())

	if m.currentAction != "" {
		status += "\n  " + tui.StatusOK.Render("⏳ " + m.currentAction)
	}

	if m.err != "" {
		status += "\n  " + tui.StatusErr.Render("⚠ " + m.err)
	}

	help := "\n  1/2/3 or h/l: switch tabs | q: quit"

	return tabs + "\n" + content + status + tui.Help.Render(help)
}

func (m Model) renderInputMode() string {
	switch m.inputMode {
	case "add_type":
		return fmt.Sprintf("\n  Add new:\n  [1] Subscription - type URL\n  [2] Subscription - paste from clipboard\n  [3] Manual (nodes)\n\n  Enter choice: %s_\n  enter: submit | esc: cancel", m.inputBuf)
	case "url":
		return fmt.Sprintf("\n  Enter subscription URL:\n  > %s_\n\n  c: paste from clipboard | enter: submit | esc: cancel", m.inputBuf)
	case "name":
		return fmt.Sprintf("\n  Enter name:\n  > %s_\n\n  enter: submit | esc: cancel", m.inputBuf)
	case "proxy_port":
		return fmt.Sprintf("\n  Enter proxy port:\n  > %s_\n\n  enter: submit | esc: cancel", m.inputBuf)
	case "api_port":
		return fmt.Sprintf("\n  Enter API port:\n  > %s_\n\n  enter: submit | esc: cancel", m.inputBuf)
	}
	return ""
}

func (m Model) configView() string {
	var b strings.Builder
	b.WriteString("\n  Subscriptions:\n")

	for i, sub := range m.settings.Subscriptions {
		prefix := "  "
		if i == m.cursorIdx {
			prefix = "> "
		}
		style := tui.SubInactive
		if i == m.settings.ActiveSubIdx {
			style = tui.SubActive
		}

		name := style.Render(sub.Name)
		traffic := "---"
		if sub.Traffic != "" {
			traffic = sub.Traffic
		}
		b.WriteString(fmt.Sprintf("%s[%d] %s  %s\n", prefix, i+1, name, traffic))
	}

	addRow := len(m.settings.Subscriptions)
	prefix := "  "
	if m.cursorIdx == addRow {
		prefix = "> "
	}
	b.WriteString(fmt.Sprintf("%s[+] Add subscription/nodes\n", prefix))

	b.WriteString("\n  Settings:\n")

	settingsStart := addRow + 1
	opts := []string{"Auto start on boot", "Auto test delay", "Auto select fastest"}
	values := []bool{m.settings.AutoStart, m.settings.AutoTestDelay, m.settings.AutoSelectBest}

	for i, opt := range opts {
		row := settingsStart + i
		prefix := "  "
		if m.cursorIdx == row {
			prefix = "> "
		}
		check := "[ ]"
		if values[i] {
			check = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, check, opt))
	}

	portsStart := settingsStart + 3
	b.WriteString("\n  Ports:\n")

	proxyPrefix := "  "
	if m.cursorIdx == portsStart {
		proxyPrefix = "> "
	}
	b.WriteString(fmt.Sprintf("%sProxy: %d\n", proxyPrefix, m.settings.ProxyPort))

	apiPrefix := "  "
	if m.cursorIdx == portsStart+1 {
		apiPrefix = "> "
	}
	b.WriteString(fmt.Sprintf("%sAPI: %d\n", apiPrefix, m.settings.APIPort))

	b.WriteString("\n  j/k: navigate | enter: action | d: delete | D: delete all\n")
	b.WriteString("  c: import clipboard | r: refresh\n")

	return b.String()
}

func (m Model) coreStatus() string {
	if m.running || m.client.IsConnected() {
		return tui.StatusOK.Render("running")
	}
	return tui.StatusErr.Render("stopped")
}

func (m Model) switchSubscription() tea.Cmd {
	return func() tea.Msg {
		m.core.Stop()
		m.logs.AddLine("Switching subscription...")
		
		s := settings.Load()
		sub := settings.GetActiveSubscription(s)
		if sub == nil || sub.URL == "" {
			return tui.MsgLogLine("No subscription")
		}
		
		m.logs.AddLine("Downloading: " + sub.URL)
		_, info, err := clash.DownloadSubscription(sub.URL, s.ProxyPort, s.APIPort)
		if err != nil {
			return tui.MsgLogLine("Error: " + err.Error())
		}
		
		if info.Traffic != "" || info.Expiry != "" {
			s.Subscriptions[s.ActiveSubIdx].Traffic = info.Traffic
			s.Subscriptions[s.ActiveSubIdx].Expiry = info.Expiry
			s.Subscriptions[s.ActiveSubIdx].LastUpdate = time.Now()
			settings.Save(s)
		}
		
		if !m.core.IsInstalled() {
			m.core.Install()
			m.core.DownloadGeoData()
		}
		
		if err := m.core.Start(); err != nil {
			return tui.MsgLogLine("Error starting: " + err.Error())
		}
		
		proxy.SetSystemProxy()
		m.running = true
		m.logs.AddLine("Core started")
		
		return tui.MsgRefresh{Traffic: info.Traffic, Expiry: info.Expiry}
	}
}

func (m Model) toggleCore() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			m.core.Stop()
			proxy.UnsetSystemProxy()
			return tui.MsgLogLine("→ Stopping existing core...")
		},
		func() tea.Msg {
			sub := settings.GetActiveSubscription(m.settings)
			if sub == nil {
				return tui.MsgLogLine("⚠ Error: no subscription, press c to import")
			}
			return tui.MsgLogLine("→ Downloading subscription...")
		},
		func() tea.Msg {
			sub := settings.GetActiveSubscription(m.settings)
			if sub == nil {
				return nil
			}
			_, info, err := clash.DownloadSubscription(sub.URL, m.settings.ProxyPort, m.settings.APIPort)
			if err != nil {
				return tui.MsgLogLine("⚠ Error: " + err.Error())
			}
			return tui.MsgRefresh{Traffic: info.Traffic, Expiry: info.Expiry}
		},
		func() tea.Msg {
			if !m.core.IsInstalled() {
				return tui.MsgLogLine("→ Installing core...")
			}
			return nil
		},
		func() tea.Msg {
			if !m.core.IsInstalled() {
				m.core.Install()
				m.core.DownloadGeoData()
			}
			return tui.MsgLogLine("→ Starting core...")
		},
		func() tea.Msg {
			err := m.core.Start()
			if err != nil {
				return tui.MsgLogLine("⚠ Error starting: " + err.Error())
			}
			proxy.SetSystemProxy()
			return tui.MsgLogLine("✓ Core started, proxy enabled")
		},
	)
}

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

		m.addType = "subscription"
		m.urlBuf = url
		m.inputMode = "name"
		m.inputBuf = ""
		m.err = ""
		return nil
	}
}

func (m Model) downloadSub(name, url string) tea.Cmd {
	return func() tea.Msg {
		m.logs.AddLine("Downloading: " + url)
		_, info, err := clash.DownloadSubscription(url, m.settings.ProxyPort, m.settings.APIPort)
		if err != nil {
			m.logs.AddLine("Error: " + err.Error())
			m.err = err.Error()
			return nil
		}

		m.logs.AddLine("Subscription saved")

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
		return tui.MsgRefresh{Traffic: info.Traffic, Expiry: info.Expiry}
	}
}