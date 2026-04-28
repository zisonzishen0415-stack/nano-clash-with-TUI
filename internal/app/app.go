package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"clashtui/internal/clash"
	"clashtui/internal/clipboard"
	"clashtui/internal/config"
	"clashtui/internal/proxy"
	"clashtui/internal/tui"
)

type Model struct {
	tab      int
	nodes    tui.NodesModel
	logs     tui.LogsModel
	core     *clash.Core
	client   *clash.Client
	running  bool
	err      string
	subInput bool
	subURL   string
}

func New() Model {
	client := clash.NewClient()
	core := clash.NewCore()
	nodes := tui.NewNodesModel(client)
	logs := tui.NewLogsModel()
	
	return Model{
		tab:    0,
		nodes:  nodes,
		logs:   logs,
		core:   core,
		client: client,
	}
}

func (m Model) Init() tea.Cmd {
	return m.nodes.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// 处理订阅输入模式
	if m.subInput {
		return m.handleSubInput(msg)
	}

	switch msg := msg.(type) {
	case tui.MsgProxiesLoaded, tui.MsgProxySwitched, tui.MsgDelayTested:
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

		// 切换标签: 1/2/3 或 h/l 或 left/right
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

		// 全局按键
		switch k {
		case "x":
			m.core.Stop()
			proxy.UnsetSystemProxy()
			m.running = false
			return m, func() tea.Msg { return tui.MsgLogLine("Stopped core, proxy disabled") }
		}

		// Nodes 标签页的按键
		if m.tab == 0 {
			cmd := m.nodes.Update(msg)
			return m, cmd
		}

		// Config 标签页按键
		if m.tab == 1 {
			switch k {
			case "s":
				m.subInput = true
				m.subURL = ""
				return m, nil
			case "c":
				return m, m.importFromClipboard()
			case "r":
				return m, m.toggleCore()
			}
		}

		// 全局
		switch k {
		case "q", "ctrl+c":
			return m, tea.Quit
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
	// 标签栏
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

	// 内容
	var content string
	switch m.tab {
	case 0:
		content = m.nodes.View()
		if content == "" {
			content = "  No proxies loaded.\n\n  Press 2/l → Config tab, then 'c' to import from clipboard"
		}
	case 1:
		content = m.configView()
	case 2:
		content = m.logs.View()
	}

	// 输入模式
	if m.subInput {
		content = fmt.Sprintf("\n  Enter subscription URL:\n  > %s\n\n  enter: submit | esc: cancel", m.subURL)
	}

	// 状态栏
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
		s += "  No subscription saved.\n\n"
	} else {
		s += "  Subscription: " + sub + "\n\n"
	}

	s += "  c: import from clipboard | s: enter subscription URL\n"
	s += "  r: refresh sub & start/stop core\n"
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

func (m Model) refreshSubscription() tea.Cmd {
	return func() tea.Msg {
		sub, err := config.LoadSubscription()
		if err != nil {
			m.err = "no subscription saved"
			return nil
		}

		m.logs.AddLine("Refreshing subscription...")
		_, err = clash.DownloadSubscription(sub)
		if err != nil {
			m.logs.AddLine("Error: " + err.Error())
			m.err = err.Error()
			return nil
		}

		m.core.Stop()
		m.core.Start()
		m.running = true
		m.logs.AddLine("Subscription refreshed")
		m.err = ""

		return tui.MsgRefresh{}
	}
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

		// 检查内核
		if !m.core.IsInstalled() {
			m.logs.AddLine("Installing core...")
			m.core.Install()
			m.core.DownloadGeoData()
		}

		m.core.Stop()
		m.core.Start()
		m.running = true
		m.logs.AddLine("Core started")

		m.err = ""
		return tui.MsgRefresh{}
	}
}