package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Tokyo Night High Contrast Colors
	ColorBg       = lipgloss.Color("#1a1b26")
	ColorFg       = lipgloss.Color("#c0caf5")
	ColorBlue     = lipgloss.Color("#7aa2f7")
	ColorCyan     = lipgloss.Color("#7dcfff")
	ColorGreen    = lipgloss.Color("#9ece6a")
	ColorRed      = lipgloss.Color("#f7768e")
	ColorPurple   = lipgloss.Color("#bb9af7")
	ColorMuted    = lipgloss.Color("#565f89")

	// Base Styles
	TitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(ColorBlue)
	SubtitleStyle = lipgloss.NewStyle().Foreground(ColorPurple).Bold(true)
	NormalStyle   = lipgloss.NewStyle().Foreground(ColorFg)
	MutedStyle    = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true)
	HighlightStyle = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	SuccessStyle  = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)

	// App Layout Styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBlue).
			Padding(1, 2)
)
