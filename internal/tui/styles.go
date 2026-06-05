package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorCyan   = lipgloss.Color("39")
	colorYellow = lipgloss.Color("214")
	colorGreen  = lipgloss.Color("42")
	colorRed    = lipgloss.Color("196")
	colorGray   = lipgloss.Color("245")
	colorWhite  = lipgloss.Color("255")

	headerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan)

	footerStyle = lipgloss.NewStyle().
			Background(colorCyan).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow).
			Background(lipgloss.Color("18"))

	normalRowStyle = lipgloss.NewStyle()

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan).
			Padding(1, 2).
			Background(lipgloss.Color("0"))

	dialogWarnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorYellow).
			Padding(1, 2).
			Background(lipgloss.Color("0"))

	dialogDangerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorRed).
			Padding(1, 2).
			Background(lipgloss.Color("0"))

	focusedFieldStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite)

	blurredFieldStyle = lipgloss.NewStyle().
				Foreground(colorGray)
)
