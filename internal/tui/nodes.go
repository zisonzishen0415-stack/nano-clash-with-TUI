package tui

import (
	"fmt"
	"strings"

	"clashtui/internal/clash"

	tea "github.com/charmbracelet/bubbletea"
)

type NodesModel struct {
	proxies    []clash.ProxyInfo
	selected   int
	current    string
	loading    bool
	testing    bool
	client     *clash.Client
	width      int
	height     int
}

type proxiesLoadedMsg []clash.ProxyInfo
type proxySwitchedMsg string
type delayTestedMsg struct {
	index int
	delay int
}

func NewNodesModel(client *clash.Client) NodesModel {
	return NodesModel{client: client, loading: true}
}

func (m NodesModel) Init() tea.Cmd {
	return m.LoadProxies
}

func (m NodesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case proxiesLoadedMsg:
		m.proxies = msg
		m.loading = false
		if len(m.proxies) > 0 && m.selected >= len(m.proxies) {
			m.selected = 0
		}

	case proxySwitchedMsg:
		m.current = string(msg)

	case delayTestedMsg:
		if msg.index < len(m.proxies) {
			m.proxies[msg.index].Delay = msg.delay
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.proxies)-1 {
				m.selected++
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "enter":
			if len(m.proxies) > 0 {
				name := m.proxies[m.selected].Name
				return m, m.switchProxy(name)
			}
		case "r":
			if len(m.proxies) > 0 {
				return m, m.testSingleDelay(m.selected)
			}
		case "R":
			return m, m.testAllDelays()
		}
	}

	return m, nil
}

func (m NodesModel) View() string {
	if m.loading {
		return "\n  Loading proxies..."
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Nodes") + "\n\n")

	for i, p := range m.proxies {
		style := nodeStyle
		if i == m.selected {
			style = selectedStyle
		}
		if p.Name == m.current {
			style = currentStyle
		}

		aliveStr := aliveStyle.Render("●")
		if !p.Alive {
			aliveStr = deadStyle.Render("●")
		}

		delayStr := fmt.Sprintf("%dms", p.Delay)
		if p.Delay == 0 {
			delayStr = "-"
		}

		line := fmt.Sprintf("  %s %s | %s", aliveStr, p.Name, delayStr)
		b.WriteString(style.Render(line) + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("j/k: select | Enter: switch | r: test | R: test all"))

	return b.String()
}

func (m NodesModel) GetCurrent() string {
	return m.current
}

func (m NodesModel) LoadProxies() tea.Msg {
	proxies, err := m.client.GetAllProxies()
	if err != nil {
		return proxiesLoadedMsg{}
	}
	current, _ := m.client.GetCurrentProxy()
	m.current = current
	return proxiesLoadedMsg(proxies)
}

func (m NodesModel) switchProxy(name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.SwitchProxy(name); err != nil {
			return proxySwitchedMsg(m.current)
		}
		return proxySwitchedMsg(name)
	}
}

func (m NodesModel) testSingleDelay(index int) tea.Cmd {
	return func() tea.Msg {
		if index >= len(m.proxies) {
			return delayTestedMsg{index: index, delay: 0}
		}
		delay, _ := m.client.TestDelay(m.proxies[index].Name)
		return delayTestedMsg{index: index, delay: delay}
	}
}

func (m NodesModel) testAllDelays() tea.Cmd {
	if len(m.proxies) == 0 {
		return nil
	}
	m.testing = true

	cmds := make([]tea.Cmd, len(m.proxies))
	for i := range m.proxies {
		name := m.proxies[i].Name
		idx := i
		cmds[i] = func() tea.Msg {
			delay, _ := m.client.TestDelay(name)
			return delayTestedMsg{index: idx, delay: delay}
		}
	}

	return tea.Batch(cmds...)
}