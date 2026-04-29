package tui

import (
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type LogsModel struct {
	lines []string
	mu    sync.Mutex
}

type MsgLogLine string

func NewLogsModel() *LogsModel {
	return &LogsModel{lines: []string{}}
}

func (m *LogsModel) Init() tea.Cmd {
	return nil
}

func (m *LogsModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case MsgLogLine:
		m.mu.Lock()
		m.lines = append(m.lines, string(msg))
		if len(m.lines) > 100 {
			m.lines = m.lines[len(m.lines)-100:]
		}
		m.mu.Unlock()
	}
	return nil
}

func (m *LogsModel) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.lines) == 0 {
		return "  No logs yet."
	}
	return strings.Join(m.lines, "\n")
}

func (m *LogsModel) AddLine(line string) {
	m.mu.Lock()
	m.lines = append(m.lines, line)
	if len(m.lines) > 100 {
		m.lines = m.lines[len(m.lines)-100:]
	}
	m.mu.Unlock()
}