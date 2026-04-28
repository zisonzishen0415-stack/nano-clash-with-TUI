package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"clashtui/internal/app"
)

func main() {
	p := tea.NewProgram(app.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}