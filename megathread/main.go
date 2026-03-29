package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	folders, err := ParseMegathread("megathread.json")
	if err != nil {
		fmt.Printf("Error parsing megathread.json: %v\n", err)
		os.Exit(1)
	}

	if len(folders) == 0 {
		fmt.Println("No folders found. Make sure megathread.json contains a valid \"content_md\" tree.")
		os.Exit(1)
	}

	m := initialModel(folders)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running Bubble Tea program: %v\n", err)
		os.Exit(1)
	}
}
