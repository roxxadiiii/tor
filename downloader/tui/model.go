package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"downloader/core"
	"downloader/rpc"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tokyo Night Palette
var (
	colorBg      = lipgloss.Color("#1a1b26")
	colorFg      = lipgloss.Color("#c0caf5")
	colorBlue    = lipgloss.Color("#7aa2f7")
	colorCyan    = lipgloss.Color("#7dcfff")
	colorGreen   = lipgloss.Color("#9ece6a")
	colorYellow  = lipgloss.Color("#e0af68")
	colorMagenta = lipgloss.Color("#bb9af7")
	colorRed     = lipgloss.Color("#f7768e")
	colorMuted   = lipgloss.Color("#565f89")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorBlue)

	sidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMagenta).
			Width(22).
			Padding(0, 1)

	tableFrameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBlue)
)

type Download struct {
	GID       string
	Name      string
	Total     int64
	Completed int64
	Speed     int64
	Status    string
}

type state int
type Category int

const (
	stateDash state = iota
	stateAdd
)

const (
	CatAll Category = iota
	CatActive
	CatCompleted
	CatPaused
	CatError
)

type model struct {
	client    *rpc.Aria2Client
	downloads []Download
	table     table.Model
	state     state
	input     textinput.Model
	category  Category
	width     int
	height    int
	message   string
	prog      progress.Model
}

func InitialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter URL (HTTP / Magnet / YouTube)"
	ti.CharLimit = 500
	ti.Width = 60

	client := rpc.New("http://127.0.0.1:6800/jsonrpc")

	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	return model{
		client:   client,
		state:    stateDash,
		input:    ti,
		prog:     prog,
		category: CatAll,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		startDaemonCmd(),
		tickCmd(),
	)
}

type tickMsg time.Time
type daemonStartedMsg struct{ err error }
type linkAddedMsg struct{ err error }

func startDaemonCmd() tea.Cmd {
	return func() tea.Msg {
		err := core.StartAriaDaemon()
		return daemonStartedMsg{err}
	}
}

// Tick at high refresh rate (400ms) for smooth IDM-like bars
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*400, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchDownloads(client *rpc.Aria2Client) []Download {
	var results []Download

	parse := func(res interface{}) {
		list, ok := res.([]interface{})
		if !ok { return }
		for _, v := range list {
			item := v.(map[string]interface{})
			gid := item["gid"].(string)
			status := item["status"].(string)
			total, _ := strconv.ParseInt(item["totalLength"].(string), 10, 64)
			comp, _ := strconv.ParseInt(item["completedLength"].(string), 10, 64)
			speedStr := "0"
			if sp, ok := item["downloadSpeed"].(string); ok { speedStr = sp }
			speed, _ := strconv.ParseInt(speedStr, 10, 64)

			name := "Unknown"
			if bt, ok := item["bittorrent"].(map[string]interface{}); ok {
				info, ok2 := bt["info"].(map[string]interface{})
				if ok2 {
					if n, _ := info["name"].(string); n != "" { name = n }
				}
			}
			if name == "Unknown" {
				if files, ok := item["files"].([]interface{}); ok && len(files) > 0 {
					f := files[0].(map[string]interface{})
					if path, ok := f["path"].(string); ok && path != "" {
						parts := strings.Split(path, "/")
						name = parts[len(parts)-1]
					} else if uris, ok := f["uris"].([]interface{}); ok && len(uris) > 0 {
						u := uris[0].(map[string]interface{})
						name = u["uri"].(string)
					}
				}
			}

			results = append(results, Download{
				GID:       gid,
				Name:      name,
				Total:     total,
				Completed: comp,
				Speed:     speed,
				Status:    status,
			})
		}
	}

	if res, err := client.Call("aria2.tellActive"); err == nil { parse(res) }
	if res, err := client.Call("aria2.tellWaiting", 0, 100); err == nil { parse(res) }
	if res, err := client.Call("aria2.tellStopped", 0, 100); err == nil { parse(res) }

	return results
}

func addLinkCmd(client *rpc.Aria2Client, url string) tea.Cmd {
	return func() tea.Msg {
		if core.IsYoutube(url) {
			urls, err := core.GetYTDLStream(url)
			if err != nil || len(urls) == 0 {
				return linkAddedMsg{fmt.Errorf("yt-dlp err: %v", err)}
			}
			for _, u := range urls {
				client.Call("aria2.addUri", []string{u})
			}
			return linkAddedMsg{nil}
		}

		_, err := client.Call("aria2.addUri", []string{url})
		return linkAddedMsg{err}
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit { return fmt.Sprintf("%d B", bytes) }
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (m *model) regenerateTable() {
	if m.width == 0 { return }
	
	// Dynamic table width bridging bounds
	tableWidth := m.width - 32
	if tableWidth < 50 { tableWidth = 50 }

	columns := []table.Column{
		{Title: "Name", Width: tableWidth / 3},
		{Title: "Size", Width: 8},
		{Title: "Status", Width: 9},
		{Title: "Speed", Width: 10},
		{Title: "Progress", Width: 22},
	}

	var rows []table.Row
	for _, d := range m.downloads {
		// Category Filtering
		if m.category == CatActive && d.Status != "active" { continue }
		if m.category == CatCompleted && d.Status != "complete" { continue }
		if m.category == CatPaused && d.Status != "paused" { continue }
		if m.category == CatError && d.Status != "error" { continue }

		percent := 0.0
		if d.Total > 0 { percent = float64(d.Completed) / float64(d.Total) }
		
		m.prog.Width = 14
		progStr := m.prog.ViewAs(percent) + fmt.Sprintf(" %.0f%%", percent*100)
		
		speedStr := "---"
		if d.Status == "active" { speedStr = formatBytes(d.Speed) + "/s" }

		rows = append(rows, table.Row{
			d.Name,
			formatBytes(d.Total),
			strings.ToUpper(d.Status),
			speedStr,
			progStr,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.height-14),
	)

	st := table.DefaultStyles()
	st.Header = st.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorBlue).
		BorderBottom(true).
		Bold(true).Foreground(colorMagenta)
	st.Selected = st.Selected.Foreground(colorBg).Background(colorGreen).Bold(false)
	
	t.SetStyles(st)
	
	// Restore cursor position if possible to prevent jumping
	if m.table.Cursor() >= 0 && m.table.Cursor() < len(rows) {
		t.SetCursor(m.table.Cursor())
	}
	
	m.table = t
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.regenerateTable()

	case tickMsg:
		m.downloads = fetchDownloads(m.client)
		m.regenerateTable()
		cmds = append(cmds, tickCmd())

	case daemonStartedMsg:
		if msg.err != nil {
			m.message = "Failed to start aria2c natively (is it installed?): " + msg.err.Error()
		} else {
			m.message = "Aria2 background process successfully bound to ~/Downloads"
		}

	case linkAddedMsg:
		if msg.err != nil {
			m.message = "Error adding link: " + msg.err.Error()
		} else {
			m.message = "Link successfully enqueued!"
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.client.Call("system.multicall", []interface{}{
				map[string]string{"methodName": "aria2.shutdown"},
			})
			return m, tea.Quit

		// Category Switching
		case "1": m.category = CatAll; m.regenerateTable()
		case "2": m.category = CatActive; m.regenerateTable()
		case "3": m.category = CatCompleted; m.regenerateTable()
		case "4": m.category = CatPaused; m.regenerateTable()
		case "5": m.category = CatError; m.regenerateTable()
		}

		if m.state == stateDash {
			switch msg.String() {
			case "q":
				m.client.Call("system.multicall", []interface{}{
					map[string]string{"methodName": "aria2.shutdown"},
				})
				return m, tea.Quit
			case "a":
				m.state = stateAdd
				m.input.Focus()
				m.input.SetValue("")
				m.message = ""
			case "p":
				// Pause/Resume toggler
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.downloads) { // Simplified correlation, might conflict with filters but fine for MVP
					filtered := m.getFiltered()
					if idx < len(filtered) {
						d := filtered[idx]
						if d.Status == "active" || d.Status == "waiting" {
							m.client.Call("aria2.pause", d.GID)
							m.message = "Paused " + d.Name
						} else if d.Status == "paused" {
							m.client.Call("aria2.unpause", d.GID)
							m.message = "Resumed " + d.Name
						}
					}
				}
			case "delete":
				idx := m.table.Cursor()
				filtered := m.getFiltered()
				if idx >= 0 && idx < len(filtered) {
					d := filtered[idx]
					m.client.Call("aria2.remove", d.GID)
					m.client.Call("aria2.removeDownloadResult", d.GID) // clear from stopped list
					m.message = "Removed " + d.Name
				}
			}
			m.table, cmd = m.table.Update(msg)
			cmds = append(cmds, cmd)

		} else if m.state == stateAdd {
			switch msg.String() {
			case "enter":
				if m.input.Value() != "" {
					m.message = "Resolving URI headers..."
					cmds = append(cmds, addLinkCmd(m.client, m.input.Value()))
					m.state = stateDash
				}
			case "esc":
				m.state = stateDash
			}
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) getFiltered() []Download {
	var filtered []Download
	for _, d := range m.downloads {
		if m.category == CatActive && d.Status != "active" { continue }
		if m.category == CatCompleted && d.Status != "complete" { continue }
		if m.category == CatPaused && d.Status != "paused" { continue }
		if m.category == CatError && d.Status != "error" { continue }
		filtered = append(filtered, d)
	}
	return filtered
}

func (m model) View() string {
	header := titleStyle.Render("⚡ Aria2 GUI Terminal ⚡")

	cat := func(idx Category, name string, num string) string {
		prefix := "  "
		if m.category == idx { prefix = lipgloss.NewStyle().Foreground(colorCyan).Render("❯ ") }
		return prefix + name + " [" + num + "]\n\n"
	}

	var act, comp, pau, errC int
	for _, d := range m.downloads {
		switch d.Status {
		case "active", "waiting": act++
		case "complete": comp++
		case "paused": pau++
		case "error": errC++
		}
	}

	sb := "\n"
	sb += lipgloss.NewStyle().Foreground(colorMagenta).Bold(true).Render(" FILTERS") + "\n\n"
	sb += cat(CatAll, "All Items", strconv.Itoa(len(m.downloads)))
	sb += cat(CatActive, "Active", strconv.Itoa(act))
	sb += cat(CatCompleted, "Completed", strconv.Itoa(comp))
	sb += cat(CatPaused, "Paused", strconv.Itoa(pau))
	sb += cat(CatError, "Errors", strconv.Itoa(errC))
	
	sb += lipgloss.NewStyle().Foreground(colorMagenta).Bold(true).Render(" HOTKEYS") + "\n\n"
	sb += "  [1-5] Filter\n\n  [a] Add URL\n\n  [p] Pause/Play\n\n  [Del] Remove\n"

	sidebar := sidebarStyle.Height(m.height - 12).Render(sb)

	var main string
	if len(m.downloads) == 0 {
		main = tableFrameStyle.Width(m.width - 32).Height(m.height - 12).Align(lipgloss.Center).Render("\n\n\nQueue is empty.")
	} else {
		main = tableFrameStyle.Render(m.table.View())
	}

	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, "  ", main)

	footer := ""
	if m.state == stateDash {
		footer = lipgloss.NewStyle().Foreground(colorMuted).Render("[Arrow Keys] Navigate Table   |   [q] Quit")
	} else if m.state == stateAdd {
		footer = lipgloss.NewStyle().Foreground(colorYellow).Render("Add Payload URI:") + "\n" +
			m.input.View() + "\n" +
			lipgloss.NewStyle().Foreground(colorMuted).Render("Press [Enter] to submit  |  [Esc] to cancel")
	}

	if m.message != "" {
		footer += "\n\n" + lipgloss.NewStyle().Foreground(colorCyan).Render(m.message)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header+"\n", layout, "\n"+footer)
}
