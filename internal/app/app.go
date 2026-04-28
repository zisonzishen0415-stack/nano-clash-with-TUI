package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"clashtui/internal/clash"
	"clashtui/internal/clipboard"
	"clashtui/internal/config"
	"clashtui/internal/tui"
)

type tab int

const (
	nodesTab tab = iota
	configTab
	logsTab
)

type Model struct {
	tabs        []tab
	activeTab   tab
	nodes       tui.NodesModel
	core        *clash.Core
	client      *clash.Client
	coreRunning bool
	width       int
	height      int
}

type coreStartedMsg struct{}
type subscriptionImportedMsg struct{}

func NewModel() Model {
	client := clash.NewClient()
	core := clash.NewCore()
	return Model{
		tabs:      []tab{nodesTab, configTab, logsTab},
		activeTab: nodesTab,
		nodes:     tui.NewNodesModel(client),
		core:      core,
		client:    client,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.initCore,
		m.nodes.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		model, cmd := m.nodes.Update(msg)
		m.nodes = model.(tui.NodesModel)
		return m, cmd

	case coreStartedMsg:
		m.coreRunning = true
		return m, func() tea.Msg { return m.nodes.LoadProxies() }

	case subscriptionImportedMsg:
		m.core.Stop()
		m.core.Start()
		return m, func() tea.Msg { return m.nodes.LoadProxies() }

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % tab(len(m.tabs))
		case "s":
			return m, m.promptSubscription()
		case "c":
			return m, m.importFromClipboard()
		case "p":
			return m, m.toggleCore()
		}

	default:
		model, cmd := m.nodes.Update(msg)
		m.nodes = model.(tui.NodesModel)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	header := m.renderTabs()
	content := ""

	switch m.activeTab {
	case nodesTab:
		content = m.nodes.View()
	case configTab:
		content = m.renderConfigTab()
	case logsTab:
		content = "Logs tab - Press 's' to import subscription"
	}

	status := fmt.Sprintf("\n%s Core: %s | Proxy: %s",
		tui.StatusStyle.Render(""),
		m.coreStatus(),
		m.currentProxy(),
	)

	return header + "\n" + content + status
}

func (m Model) renderTabs() string {
	tabNames := []string{"Nodes", "Config", "Logs"}
	var tabs string
	for i, name := range tabNames {
		if tab(i) == m.activeTab {
			tabs += tui.ActiveTabStyle.Render(name)
		} else {
			tabs += tui.TabStyle.Render(name)
		}
		tabs += " "
	}
	return tabs
}

func (m Model) renderConfigTab() string {
	var b strings.Builder
	b.WriteString("Config Tab\n\n")

	sub, err := config.LoadSubscription()
	if err != nil {
		b.WriteString("No subscription saved\n")
	} else {
		b.WriteString("Current subscription:\n")
		b.WriteString(sub + "\n")
	}

	b.WriteString("\nPress 's' to enter new subscription\n")
	b.WriteString("Press 'c' to import from clipboard\n")

	return b.String()
}

func (m Model) coreStatus() string {
	if m.coreRunning {
		return tui.AliveStyle.Render("Running")
	}
	return tui.DeadStyle.Render("Stopped")
}

func (m Model) currentProxy() string {
	current := m.nodes.GetCurrent()
	if current != "" {
		return current
	}
	return "-"
}

func (m Model) initCore() tea.Msg {
	if !m.core.IsInstalled() {
		fmt.Println("Installing Clash core...")
		m.core.Install()
		m.core.DownloadGeoData()
	}

	if !m.core.IsRunning() {
		fmt.Println("Starting Clash core...")
		m.core.Start()
	}

	return coreStartedMsg{}
}

func (m Model) toggleCore() tea.Cmd {
	return func() tea.Msg {
		if m.coreRunning {
			m.core.Stop()
			m.coreRunning = false
		} else {
			m.core.Start()
			m.coreRunning = true
		}
		return nil
	}
}

func (m Model) promptSubscription() tea.Cmd {
	return func() tea.Msg {
		fmt.Print("\nEnter subscription URL: ")
		var url string
		fmt.Scanln(&url)
		if url == "" {
			return nil
		}

		_, err := clash.DownloadSubscription(url)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return nil
		}

		return subscriptionImportedMsg{}
	}
}

func (m Model) importFromClipboard() tea.Cmd {
	return func() tea.Msg {
		url, err := clipboard.Read()
		if err != nil {
			fmt.Printf("Clipboard error: %v\n", err)
			return nil
		}

		if url == "" || !strings.HasPrefix(url, "http") {
			fmt.Println("No valid URL in clipboard")
			return nil
		}

		_, err = clash.DownloadSubscription(url)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return nil
		}

		return subscriptionImportedMsg{}
	}
}