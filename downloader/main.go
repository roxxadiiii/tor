package main

import (
	"fmt"
	"os"

	"downloader/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m := tui.InitialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
