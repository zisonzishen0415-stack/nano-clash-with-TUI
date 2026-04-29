package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"clashtui/internal/clash"
	"clashtui/internal/clipboard"
	"clashtui/internal/config"
	"clashtui/internal/proxy"
	"clashtui/internal/settings"
	"clashtui/internal/tui"
)

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
	setIdx    int // settings menu index
}

func New() Model {
	client := clash.NewClient()
	core := clash.NewCore()
	nodes := tui.NewNodesModel(client)
	logs := tui.NewLogsModel()
	s := settings.Load()

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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.subInput {
		return m.handleSubInput(msg)
	}

	switch msg := msg.(type) {
	case tui.MsgProxiesLoaded, tui.MsgProxySwitched, tui.MsgDelayTested, tui.MsgRetryLoad, tui.MsgTestProgress:
		cmd := m.nodes.Update(msg)
		return m, cmd

	case tui.MsgLogLine:
		m.logs.Update(msg)
		return m, nil

	case tui.MsgRefresh:
		m.running = true
		cmd := m.nodes.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		k := msg.String()

		// Tab switching
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

		// Global keys
		switch k {
		case "x":
			m.core.Stop()
			proxy.UnsetSystemProxy()
			m.running = false
			return m, func() tea.Msg { return tui.MsgLogLine("Stopped core, proxy disabled") }
		case "s":
			m.subInput = true
			m.subURL = ""
			return m, nil
		case "c":
			return m, m.importFromClipboard()
		case "r":
			return m, m.toggleCore()
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		// Tab-specific keys
		if m.tab == 0 {
			cmd := m.nodes.Update(msg)
			return m, cmd
		}

		if m.tab == 1 {
			return m.handleSettingsKeys(k)
		}
	}

	return m, nil
}

func (m Model) handleSettingsKeys(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "j", "down":
		if m.setIdx < 4 {
			m.setIdx++
		}
		return m, nil
	case "k", "up":
		if m.setIdx > 0 {
			m.setIdx--
		}
		return m, nil
	case "enter", " ":
		switch m.setIdx {
		case 0:
			m.settings.AutoStart = !m.settings.AutoStart
			settings.Save(m.settings)
			// Enable/disable systemd service
			home, _ := os.UserHomeDir()
			serviceDir := filepath.Join(home, ".config", "systemd", "user")
			serviceFile := filepath.Join(serviceDir, "clashtui.service")

			if m.settings.AutoStart {
				// Create directory and write service file
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
		case 1:
			m.settings.AutoTestDelay = !m.settings.AutoTestDelay
			settings.Save(m.settings)
			return m, nil
		case 2:
			m.settings.AutoSelectBest = !m.settings.AutoSelectBest
			settings.Save(m.settings)
			return m, nil
		}
	}
	return m, nil
}

func (m Model) handleSubInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "enter":
			m.subInput = false
			if m.subURL != "" {
				return m, m.downloadSub(m.subURL)
			}
			return m, nil
		case "esc":
			m.subInput = false
			m.subURL = ""
			return m, nil
		case "backspace":
			if len(m.subURL) > 0 {
				m.subURL = m.subURL[:len(m.subURL)-1]
			}
			return m, nil
		default:
			if len(k) == 1 {
				m.subURL += k
			}
			return m, nil
		}
	}
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

	if m.subInput {
		content = fmt.Sprintf("\n  Enter subscription URL:\n  > %s\n\n  enter: submit | esc: cancel", m.subURL)
	}

	status := fmt.Sprintf("\n\n  Core: %s | Current: %s",
		m.coreStatus(), m.nodes.GetCurrent())

	if m.err != "" {
		status += " | " + tui.StatusErr.Render(m.err)
	}

	help := "\n  1/2/3 or h/l: switch tabs | q: quit"

	return tabs + "\n" + content + status + tui.Help.Render(help)
}

func (m Model) configView() string {
	s := "\n"
	sub, err := config.LoadSubscription()
	if err != nil {
		s += "  Subscription: none\n\n"
	} else {
		s += "  Subscription: " + sub + "\n\n"
	}

	// Settings menu
	opts := []struct {
		name  string
		value bool
	}{
		{"Auto start on boot", m.settings.AutoStart},
		{"Auto test delay", m.settings.AutoTestDelay},
		{"Auto select fastest", m.settings.AutoSelectBest},
	}

	s += "  Settings:\n"
	for i, opt := range opts {
		prefix := "  "
		if i == m.setIdx {
			prefix = "> "
		}
		check := "[ ]"
		if opt.value {
			check = "[x]"
		}
		s += fmt.Sprintf("%s%s %s\n", prefix, check, opt.name)
	}

	s += "\n  j/k: select | enter: toggle\n"
	s += "  c: import clipboard | s: enter URL | r: refresh\n"
	return s
}

func (m Model) coreStatus() string {
	if m.running || m.client.IsConnected() {
		return tui.StatusOK.Render("running")
	}
	return tui.StatusErr.Render("stopped")
}

func (m Model) toggleCore() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			m.core.Stop()
			proxy.UnsetSystemProxy()
			return tui.MsgLogLine("Stopped existing core")
		},
		func() tea.Msg {
			sub, err := config.LoadSubscription()
			if err != nil || sub == "" {
				return tui.MsgLogLine("Error: no subscription, press c to import")
			}
			_, err = clash.DownloadSubscription(sub)
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

		return m.downloadSub(url)()
	}
}

func (m Model) downloadSub(url string) tea.Cmd {
	return func() tea.Msg {
		m.logs.AddLine("Downloading: " + url)
		_, err := clash.DownloadSubscription(url)
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
		return tui.MsgRefresh{}
	}
}