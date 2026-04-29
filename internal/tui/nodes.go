package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"clashtui/internal/clash"

	tea "github.com/charmbracelet/bubbletea"
)

type NodesModel struct {
	proxies        []clash.ProxyInfo
	selected       int
	current        string
	loading        bool
	client         *clash.Client
	testing        bool
	testIdx        int
	retries        int
	autoSelectBest bool
	initialLoad    bool
}

type MsgProxiesLoaded []clash.ProxyInfo
type MsgProxySwitched string
type MsgDelayTested struct {
	Index int
	Delay int
}
type MsgRefresh struct{}
type MsgRetryLoad struct{}
type MsgTestProgress struct {
	Index int
	Total int
}
type MsgStopCore struct{}

func NewNodesModel(client *clash.Client) NodesModel {
	return NodesModel{
		client:         client,
		loading:        true,
		testing:        false,
		retries:        0,
		autoSelectBest: true,
		initialLoad:    true,
	}
}

func (m NodesModel) Init() tea.Cmd {
	return m.loadProxies
}

func (m *NodesModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case MsgProxiesLoaded:
		m.proxies = msg
		m.loading = false
		m.retries = 0
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
		if len(m.proxies) > 0 {
			if m.selected >= len(m.proxies) {
				m.selected = 0
			}
			for i, p := range m.proxies {
				if p.Name == m.current {
					m.selected = i
					break
				}
			}
			m.testing = true
			m.testIdx = 0
			return tea.Sequence(
				func() tea.Msg { return MsgTestProgress{Index: 0, Total: len(m.proxies)} },
				m.testDelay(0),
			)
		}
		return nil

	case MsgRetryLoad:
		m.retries++
		if m.retries < 10 {
			return tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return MsgRefresh{}
			})
		}
		m.loading = false
		return nil

	case MsgProxySwitched:
		m.current = string(msg)
		if m.client != nil {
			m.client.SwitchProxy(m.current)
		}
		return nil

	case MsgTestProgress:
		m.testing = true
		m.testIdx = msg.Index
		return nil

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

		if m.initialLoad && m.autoSelectBest && len(m.proxies) > 0 {
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
		m.initialLoad = false
		return nil

	case MsgRefresh:
		m.loading = true
		return m.loadProxies

	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "j", "down":
			if len(m.proxies) > 0 && m.selected < len(m.proxies)-1 {
				m.selected++
			}
		case "k", "up":
			if len(m.proxies) > 0 && m.selected > 0 {
				m.selected--
			}
		case "enter":
			if len(m.proxies) > 0 {
				name := m.proxies[m.selected].Name
				m.current = name
				if m.client != nil {
					m.client.SwitchProxy(name)
				}
				return func() tea.Msg { return MsgProxySwitched(name) }
			}
		case "x":
			return func() tea.Msg { return MsgStopCore{} }
		case "t":
			if len(m.proxies) > 0 {
				return m.testDelay(m.selected)
			}
		case "T":
			if len(m.proxies) > 0 && !m.testing {
				m.testing = true
				m.testIdx = 0
				return tea.Sequence(
					func() tea.Msg { return MsgTestProgress{Index: 0, Total: len(m.proxies)} },
					m.testDelay(0),
				)
			}
		}
	}

	return nil
}

func (m NodesModel) View() string {
	var b strings.Builder

	if m.testing {
		b.WriteString(fmt.Sprintf("  Testing: %d/%d\n\n", m.testIdx+1, len(m.proxies)))
	}

	if m.loading {
		return "  Loading proxies..."
	}

	if len(m.proxies) == 0 {
		return "  No proxies loaded. Press 'r' to refresh."
	}

	maxShow := 12
	start := 0
	if m.selected > maxShow/2 {
		start = m.selected - maxShow/2
	}
	end := start + maxShow
	if end > len(m.proxies) {
		end = len(m.proxies)
		start = end - maxShow
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		p := m.proxies[i]
		prefix := "  "
		if i == m.selected {
			prefix = "> "
		}

		status := "*"
		if !p.Alive {
			status = "o"
		}

		delay := fmt.Sprintf("%dms", p.Delay)
		if p.Delay == 0 {
			delay = "-"
		}
		delayStyled := DelayStyle(p.Delay).Render(delay)

		curr := ""
		if p.Name == m.current {
			curr = " [current]"
		}

		b.WriteString(fmt.Sprintf("%s%s %s%s | %s\n", prefix, status, p.Name, curr, delayStyled))
	}

	b.WriteString("\n  j/k: select | enter: switch | t: test | T: test all")
	return b.String()
}

func (m NodesModel) loadProxies() tea.Msg {
	if m.client == nil {
		return MsgProxiesLoaded{}
	}
	proxies, err := m.client.GetAllProxies()
	if err != nil || len(proxies) == 0 {
		return MsgRetryLoad{}
	}
	current, _ := m.client.GetCurrentProxy()
	m.current = current
	return MsgProxiesLoaded(proxies)
}

func (m NodesModel) testDelay(idx int) tea.Cmd {
	return func() tea.Msg {
		if idx < len(m.proxies) && m.client != nil {
			delay, _ := m.client.TestDelay(m.proxies[idx].Name)
			return MsgDelayTested{Index: idx, Delay: delay}
		}
		return MsgDelayTested{Index: idx, Delay: 0}
	}
}

func (m NodesModel) GetCurrent() string {
	return m.current
}

func (m *NodesModel) SetAutoSelectBest(enabled bool) {
	m.autoSelectBest = enabled
}