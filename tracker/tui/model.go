package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tracker/fetch"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateMenu state = iota
	stateView
)

const banner = `
  _______                  _             
 |__   __|                | |            
    | |_ __ __ _  ___| | _____ _ __ 
    | | '__/ _ \ |/ __| |/ / _ \ '__|
    | | | | (_| | (__|   <  __/ |   
    |_|_|  \__,_|\___|_|\_\___|_|   
`

type model struct {
	state    state
	options  []string
	cursor   int
	trackers string
	kind     string
	message  string
	fetching bool
	width    int
	height   int
}

func InitialModel() model {
	return model{
		state:   stateMenu,
		options: []string{"All", "HTTP", "HTTPS", "IP", "Best"},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

type fetchedMsg string
type errMsg error

func fetchCmd(kind string) tea.Cmd {
	return func() tea.Msg {
		data, err := fetch.FetchTrackers(kind)
		if err != nil {
			return errMsg(err)
		}
		return fetchedMsg(data)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		if m.state == stateMenu {
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.options)-1 {
					m.cursor++
				}
			case "enter":
				m.kind = m.options[m.cursor]
				m.state = stateView
				m.fetching = true
				m.message = "Fetching " + m.kind + " trackers..."
				return m, fetchCmd(m.kind)
			}
		} else if m.state == stateView {
			switch msg.String() {
			case "esc", "b":
				m.state = stateMenu
				m.message = ""
			case "c":
				if m.trackers != "" {
					err := clipboard.WriteAll(m.trackers)
					if err != nil {
						// Fallback explicitly to Wayland's wl-copy
						cmd := exec.Command("wl-copy")
						cmd.Stdin = strings.NewReader(m.trackers)
						if err2 := cmd.Run(); err2 == nil {
							m.message = "Copied to clipboard (using wl-copy)!"
						} else {
							m.message = "Failed to copy to clipboard (tried xclip/xsel/wl-copy)"
						}
					} else {
						m.message = "Copied to clipboard!"
					}
				}
			case "s":
				if m.trackers != "" {
					filename := fmt.Sprintf("trackers_%s.txt", m.kind)
					err := os.WriteFile(filename, []byte(m.trackers), 0644)
					if err != nil {
						m.message = "Failed to save file: " + err.Error()
					} else {
						m.message = fmt.Sprintf("Saved to %s!", filename)
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case fetchedMsg:
		m.fetching = false
		m.trackers = string(msg)
		m.message = fmt.Sprintf("Fetched %s trackers! Press 'c' to copy, 's' to save. 'esc' to go back.", m.kind)

	case errMsg:
		m.fetching = false
		m.message = "Error: " + msg.Error()
	}

	return m, nil
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7")).MarginBottom(1)
	itemStyle   = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#c0caf5"))
	selected    = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("#f7768e")).Bold(true)
	msgStyle    = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#9ece6a")).MarginTop(1)
	textWrap    = lipgloss.NewStyle().Width(80).MarginTop(1).Foreground(lipgloss.Color("#c0caf5"))
	bannerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7")).Bold(true).MarginBottom(1)
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Italic(true).MarginTop(2)
)

func (m model) View() string {
	var s string

	s += bannerStyle.Render(banner) + "\n"

	if m.state == stateMenu {
		s += titleStyle.Render("Select Tracker Type:") + "\n"
		for i, choice := range m.options {
			cursor := "   "
			if m.cursor == i {
				cursor = " > "
				s += selected.Render(fmt.Sprintf("%s%s", cursor, choice)) + "\n"
			} else {
				s += itemStyle.Render(fmt.Sprintf("%s%s", cursor, choice)) + "\n"
			}
		}
		s += "\nPress q to quit.\n"
	} else {
		// State View
		s += titleStyle.Render(fmt.Sprintf("%s Trackers", m.kind)) + "\n"

		if m.fetching {
			s += "Fetching data online...\n"
		} else {
			lines := 0
			maxLines := 10
			preview := ""
			for _, char := range m.trackers {
				if char == '\n' {
					lines++
				}
				if lines >= maxLines {
					preview += "\n... (more trackers available)"
					break
				}
				preview += string(char)
			}

			if preview == "" {
				preview = "No trackers found."
			}
			s += textWrap.Render(preview) + "\n"
		}
	}

	s += msgStyle.Render(m.message) + "\n"
	s += footerStyle.Render("Made with love by Github : roxxadiiii") + "\n"

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s)
}
