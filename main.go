package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"clashtui/internal/app"
	"clashtui/internal/singleinstance"
)

func main() {
	acquired, err := singleinstance.Acquire()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if !acquired {
		fmt.Println("Another instance is running, exiting...")
		os.Exit(0)
	}

	defer singleinstance.Release()

	p := tea.NewProgram(
		app.NewModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}