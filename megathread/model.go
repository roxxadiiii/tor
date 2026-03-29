package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/skratchdot/open-golang/open"
)

var (
	colorBg      = lipgloss.Color("#1a1b26")
	colorFg      = lipgloss.Color("#c0caf5")
	colorBlue    = lipgloss.Color("#7aa2f7")
	colorMagenta = lipgloss.Color("#bb9af7")
	colorGreen   = lipgloss.Color("#9ece6a")
	colorMuted   = lipgloss.Color("#565f89")

	titleStyle = lipgloss.NewStyle().
			Background(colorBlue).
			Foreground(colorBg).
			Padding(0, 1).
			Bold(true)

	statusStyle = lipgloss.NewStyle().Foreground(colorGreen).PaddingLeft(2)
	errorStyle  = lipgloss.NewStyle().Foreground(colorMagenta).PaddingLeft(2)
)

type state int
const (
	stateFolders state = iota
	stateBookmarks
)

type model struct {
	folders    []Folder
	list       list.Model
	state      state
	activeFolder int
	message    string
}

func initialModel(folders []Folder) model {
	var items []list.Item
	for _, f := range folders {
		// Custom Description string for list interface
		items = append(items, folderItem{f})
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(colorBlue).BorderLeftForeground(colorBlue)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(colorMagenta).BorderLeftForeground(colorBlue)

	l := list.New(items, delegate, 0, 0)
	l.Title = "Megathread Wiki (Folders)"
	l.Styles.Title = titleStyle

	return model{
		folders: folders,
		list:    l,
		state:   stateFolders,
	}
}

type folderItem struct { Folder }
func (f folderItem) Title() string       { return f.Name }
func (f folderItem) Description() string { return fmt.Sprintf("Contains %d bookmarks", len(f.Bookmarks)) }
func (f folderItem) FilterValue() string { return f.Name }

func (m *model) loadBookmarks(folderIdx int) {
	m.activeFolder = folderIdx
	var items []list.Item
	for _, b := range m.folders[folderIdx].Bookmarks {
		items = append(items, b)
	}
	m.list.SetItems(items)
	m.list.Title = "Directory: " + m.folders[folderIdx].Name
	m.list.ResetSelected()
	m.state = stateBookmarks
}

func (m *model) loadFolders() {
	var items []list.Item
	for _, f := range m.folders {
		items = append(items, folderItem{f})
	}
	m.list.SetItems(items)
	m.list.Title = "Megathread Wiki (Folders)"
	m.list.ResetSelected()
	m.state = stateFolders
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.list.FilterState() != list.Filtering {
				return m, tea.Quit
			}
		}

		if m.state == stateFolders {
			switch msg.String() {
			case "enter":
				if i, ok := m.list.SelectedItem().(folderItem); ok {
					// Find index
					for idx, f := range m.folders {
						if f.Name == i.Name {
							m.message = ""
							m.loadBookmarks(idx)
							break
						}
					}
				}
			}
		} else if m.state == stateBookmarks {
			switch msg.String() {
			case "esc", "b":
				if m.list.FilterState() != list.Filtering {
					m.message = ""
					m.loadFolders()
					return m, nil // intercept esc from closing bubbles filter if not active
				}
			case "c":
				if i, ok := m.list.SelectedItem().(BookmarkItem); ok {
					err := clipboard.WriteAll(i.URL)
					if err != nil {
						cmd2 := exec.Command("wl-copy")
						cmd2.Stdin = strings.NewReader(i.URL)
						if err2 := cmd2.Run(); err2 == nil {
							m.message = "Copied to Wayland clipboard: " + i.URL
						} else {
							m.message = "Failed to copy: " + err.Error()
						}
					} else {
						m.message = "Copied to clipboard: " + i.URL
					}
				}
			case "o":
				if i, ok := m.list.SelectedItem().(BookmarkItem); ok {
					err := open.Run(i.URL)
					if err != nil {
						m.message = "Failed to open browser: " + err.Error()
					} else {
						m.message = "Opened in default browser: " + i.URL
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-2) // leave room for message footer
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	s := lipgloss.NewStyle().Margin(1, 2).Render(m.list.View())
	
	footer := ""
	if m.state == stateBookmarks {
		footer = lipgloss.NewStyle().Foreground(colorMuted).PaddingLeft(2).Render("[c] Copy Link   [o] Open Browser   [b/esc] Go Back")
	} else {
		footer = lipgloss.NewStyle().Foreground(colorMuted).PaddingLeft(2).Render("[enter] Open Category   [q] Quit")
	}

	if m.message != "" {
		if strings.Contains(m.message, "Failed") {
			footer += "\n" + errorStyle.Render(m.message)
		} else {
			footer += "\n" + statusStyle.Render(m.message)
		}
	}

	return s + "\n" + footer
}
