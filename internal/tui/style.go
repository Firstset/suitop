package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Base colors
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#2D3748")
	successColor   = lipgloss.Color("#48BB78")
	errorColor     = lipgloss.Color("#F56565")
	warningColor   = lipgloss.Color("#ECC94B")
	textColor      = lipgloss.Color("#F7FAFC")
	mutedColor     = lipgloss.Color("#A0AEC0")

	// Header styles
	headerStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(primaryColor).
			Padding(1, 2).
			Bold(true).
			Width(100)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	// Info box style
	infoBoxStyle = boxStyle.Copy().
			Width(40)

	// Progress bar container style
	progressBarStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2)

	// Table styles
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Background(secondaryColor).
				Bold(true).
				Padding(0, 1)

	tableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Status indicator styles
	activeStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	inactiveStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)
)

// AdjustStyles updates style widths based on terminal dimensions
func AdjustStyles(width int) {
	// Adjust header to full width
	headerStyle = headerStyle.Width(width - 4)

	// Adjust boxes
	boxWidth := width - 4
	boxStyle = boxStyle.Width(boxWidth)
	infoBoxStyle = infoBoxStyle.Width(boxWidth / 2)
	progressBarStyle = progressBarStyle.Width(boxWidth)
}
