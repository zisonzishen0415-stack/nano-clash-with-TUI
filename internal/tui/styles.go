package tui

import "github.com/charmbracelet/lipgloss"

var (
	TabActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		Background(lipgloss.Color("#1F2937")).
		Padding(0, 1)

	TabInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Padding(0, 1)

	StatusOK = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	StatusErr = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444"))

	Help = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))
)