package tui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		Padding(0, 1)

	TabStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Padding(0, 2)

	ActiveTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		Background(lipgloss.Color("#1F2937")).
		Padding(0, 2)

	NodeStyle = lipgloss.NewStyle().
		Padding(0, 1)

	SelectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#10B981")).
		Padding(0, 1)

	CurrentStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Padding(0, 1)

	AliveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	DeadStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444"))

	HelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Padding(0, 1)

	StatusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Padding(1, 0)

	titleStyle = TitleStyle
	tabStyle = TabStyle
	activeTabStyle = ActiveTabStyle
	nodeStyle = NodeStyle
	selectedStyle = SelectedStyle
	currentStyle = CurrentStyle
	aliveStyle = AliveStyle
	deadStyle = DeadStyle
	helpStyle = HelpStyle
)