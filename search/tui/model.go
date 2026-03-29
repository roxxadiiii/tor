package tui

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"search/fetch"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/skratchdot/open-golang/open"
)

type state int
type SortMode int

const (
	stateInput state = iota
	stateLoading
	stateResults
	stateDetails
)

const (
	SortDefault SortMode = iota // No sort
	SortSeeds
	SortSize
	SortDate
)

const banner = `
  _____                            _          
 |  __ \                          | |         
 | |  | | ___  ___ ___ _ __  _ __ | |_ ___    
 | |  | |/ _ \/ __/ _ \ '_ \| '_ \| __/ _ \   
 | |__| |  __/ (_|  __/ | | | | | | || (_) |  
 |_____/ \___|\___\___|_| |_|_| |_|\__\___/   
 `

type model struct {
	state       state
	input       textinput.Model
	spinner     spinner.Model
	table       table.Model
	results     []fetch.Torrent
	selectedIdx int
	message     string
	sortMode    SortMode
	width       int
	height      int
}

func InitialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Search magnet / IMDB ID (e.g. tt1234567)"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = HighlightStyle

	return model{
		state:    stateInput,
		input:    ti,
		spinner:  s,
		sortMode: SortDefault,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

type searchResultsMsg []fetch.Torrent
type errMsg error

func performSearch(query string) tea.Cmd {
	return func() tea.Msg {
		res, err := fetch.FetchConcurrently(query)
		if err != nil {
			return errMsg(err)
		}
		return searchResultsMsg(res)
	}
}

func (m *model) regenerateTable() {
	// Re-sort current results
	if m.sortMode == SortSeeds {
		sort.Slice(m.results, func(i, j int) bool {
			return m.results[i].Seeders > m.results[j].Seeders
		})
	} else if m.sortMode == SortSize {
		sort.Slice(m.results, func(i, j int) bool {
			return m.results[i].SizeRaw > m.results[j].SizeRaw
		})
	} else if m.sortMode == SortDate {
		sort.Slice(m.results, func(i, j int) bool {
			return m.results[i].UploadDate > m.results[j].UploadDate
		})
	}

	columns := []table.Column{
		{Title: "Name", Width: m.width - 60}, // Dynamic width taking bulk of space
		{Title: "Size", Width: 10},
		{Title: "SE", Width: 6},
		{Title: "LE", Width: 6},
		{Title: "Date", Width: 12},
		{Title: "Site", Width: 12},
	}
	var rows []table.Row
	for _, r := range m.results {
		rows = append(rows, table.Row{
			r.Name,
			r.Size,
			fmt.Sprintf("%d", r.Seeders),
			fmt.Sprintf("%d", r.Leechers),
			r.UploadDate,
			r.OriginSite,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.height-18),
	)
	st := table.DefaultStyles()
	st.Header = st.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBlue).
		BorderBottom(true).
		Bold(true).Foreground(ColorPurple)
	st.Selected = st.Selected.
		Foreground(ColorBg).
		Background(ColorGreen).
		Bold(false)
	t.SetStyles(st)
	m.table = t
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q": // Let table handle 'q' native quit inside results unless overriding
			if m.state == stateInput || m.state == stateLoading {
				return m, tea.Quit
			} else if m.state == stateResults {
				return m, tea.Quit
			}
		}

		switch m.state {
		case stateInput:
			switch msg.String() {
			case "enter":
				if m.input.Value() != "" {
					m.state = stateLoading
					m.sortMode = SortDefault
					m.message = ""
					return m, tea.Batch(m.spinner.Tick, performSearch(m.input.Value()))
				}
			}
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)

		case stateLoading:
			// Waiting internally

		case stateResults:
			switch msg.String() {
			case "esc", "b":
				m.state = stateInput
				m.input.SetValue("")
				m.message = ""
			case "enter":
				if len(m.results) > 0 {
					m.selectedIdx = m.table.Cursor()
					if m.selectedIdx >= 0 && m.selectedIdx < len(m.results) {
						m.state = stateDetails
						m.message = ""
					}
				}
			case "s":
				// Sort toggle
				m.sortMode = (m.sortMode + 1) % 4
				modeStr := []string{"Default", "Seeders", "Size", "Date"}[m.sortMode]
				m.message = "Sorted by: " + modeStr
				m.regenerateTable()
			}
			m.table, cmd = m.table.Update(msg)
			cmds = append(cmds, cmd)

		case stateDetails:
			switch msg.String() {
			case "esc", "b", "q":
				m.state = stateResults
				m.message = ""
			case "c":
				tor := m.results[m.selectedIdx]
				err := clipboard.WriteAll(tor.Magnet)
				if err != nil {
					cmd2 := exec.Command("wl-copy")
					cmd2.Stdin = strings.NewReader(tor.Magnet)
					if err2 := cmd2.Run(); err2 == nil {
						m.message = "Magnet link copied using wl-copy!"
					} else {
						m.message = "Failed to copy to clipboard (tried xclip/xsel/wl-copy)."
					}
				} else {
					m.message = "Magnet link copied to clipboard!"
				}
			case "o":
				tor := m.results[m.selectedIdx]
				err := open.Run(tor.Magnet)
				if err != nil {
					m.message = "Failed to open in default client: " + err.Error()
				} else {
					m.message = "Magnet link passed to default torrent client!"
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == stateResults && len(m.results) > 0 {
			m.regenerateTable()
		}

	case searchResultsMsg:
		m.state = stateResults
		m.results = msg
		if len(m.results) == 0 {
			m.message = "No results found."
			m.state = stateInput
			return m, nil
		}
		m.regenerateTable()
		m.message = fmt.Sprintf("Found %d torrents.", len(m.results))

	case errMsg:
		m.state = stateInput
		m.message = "Error: " + msg.Error()

	case spinner.TickMsg:
		if m.state == stateLoading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var s string

	s += TitleStyle.Render(banner) + "\n"

	if m.state == stateInput {
		s += SubtitleStyle.Render("Search Magnet Torrents") + "\n\n"
		s += m.input.View() + "\n\n"
		s += MutedStyle.Render("Press enter to search, q or ctrl+c to quit.") + "\n"
	} else if m.state == stateLoading {
		s += fmt.Sprintf("\n %s Searching the high seas for '%s'...\n", m.spinner.View(), m.input.Value())
	} else if m.state == stateResults {
		s += SubtitleStyle.Render(fmt.Sprintf("Results for '%s'", m.input.Value())) + "\n"
		s += m.table.View() + "\n"
		s += MutedStyle.Render("Press [Enter] to inspect, [s] to sort list, [b/esc] to search again, [q] to quit.") + "\n"
	} else if m.state == stateDetails {
		tor := m.results[m.selectedIdx]

		d := ""
		if len(tor.Magnet) > 50 {
			d = tor.Magnet[:50] + "..."
		} else {
			d = tor.Magnet
		}

		body := fmt.Sprintf(`%s

%s   %s
%s %d
%s %d
%s   %s
%s     %s

%s
%s
`,
			TitleStyle.Render(tor.Name),
			HighlightStyle.Render("Size:"), NormalStyle.Render(tor.Size),
			SuccessStyle.Render("Seeders:"), tor.Seeders,
			HighlightStyle.Render("Leechers:"), tor.Leechers,
			SubtitleStyle.Render("Date:"), NormalStyle.Render(tor.UploadDate),
			SubtitleStyle.Render("Site:"), NormalStyle.Render(tor.OriginSite),
			strings.Repeat("─", 40),
			MutedStyle.Render("Magnet Link: "+d),
		)

		s += BoxStyle.Render(body) + "\n\n"
		s += TitleStyle.Render("[c] Copy Magnet Link") + "  " + SuccessStyle.Render("[o] Open in Tor Client") + "\n"
		s += MutedStyle.Render("Press [b] or [esc] to go back to results.") + "\n"
	}

	if m.message != "" {
		s += "\n" + MutedStyle.Render(m.message) + "\n"
	}

	s += "\n" + MutedStyle.Render("Made with love by Github : roxxadiiii") + "\n"

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s)
	}
	return s
}
