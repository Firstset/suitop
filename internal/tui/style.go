package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Base colors
	primaryColor      = lipgloss.Color("#7D56F4")
	secondaryColor    = lipgloss.Color("#2D3748")
	successColor      = lipgloss.Color("#48BB78")
	errorColor        = lipgloss.Color("#F56565")
	warningColor      = lipgloss.Color("#ECC94B")
	textColor         = lipgloss.Color("#F7FAFC")
	mutedColor        = lipgloss.Color("#A0AEC0")
	validatorBarColor = lipgloss.Color("#5fff87") // Green for validator count bar
	powerBarColor     = lipgloss.Color("#ffdf5d") // Gold for voting power bar

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

	// Panel styles for the split layout
	leftPanelStyle   = boxStyle.Copy()
	middlePanelStyle = boxStyle.Copy()
	rightPanelStyle  = boxStyle.Copy()

	// Header panel styles
	headerBoxStyle = boxStyle.Copy().
			BorderForeground(primaryColor).
			Padding(1, 2)

	// Info panel specific style
	infoPanelStyle = boxStyle.Copy().
			BorderForeground(primaryColor)

	// Info box style
	infoBoxStyle = boxStyle.Copy()
	//.Width(40)

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

	// Progress bar style variants
	validatorBarStyle   = lipgloss.NewStyle().Foreground(validatorBarColor)
	votingPowerBarStyle = lipgloss.NewStyle().Foreground(powerBarColor)

	// Progress bars panel style
	progressPanelStyle = boxStyle.Copy().
				BorderForeground(primaryColor)
)

// AdjustStyles updates style widths based on terminal dimensions
func AdjustStyles(total, left, middle, right int) {
	// Adjust header to full width
	headerStyle = headerStyle.Width(total - 4)

	// Adjust boxes
	boxWidth := total - 4
	//boxStyle = boxStyle.Width(boxWidth / 2)
	infoBoxStyle = infoBoxStyle.Width(boxWidth / 2)

	// Set both panels to same height and width
	height := 8
	leftPanelStyle = leftPanelStyle.Width(left).Height(height)
	middlePanelStyle = middlePanelStyle.Width(middle).Height(height)
	rightPanelStyle = rightPanelStyle.Width(right).Height(height)

	// Make header panels same height and width
	//headerBoxStyle = headerBoxStyle.Width(total).Height(height)
	infoPanelStyle = infoPanelStyle.Width(boxWidth / 2).Height(height)
	progressPanelStyle = progressPanelStyle.Width(boxWidth / 2).Height(height)

	// Make the progress bar container narrower than its panel
	// to account for borders, padding, and label text.
	// The progress.Model's Width will be set in the Model.Update based on this.
	// validatorBarStyle and votingPowerBarStyle are for the bar's visual style (colors), not its width.
	// Remove Width(total) from these as it's incorrect.
	// validatorBarStyle = validatorBarStyle.Width(total)
	// votingPowerBarStyle = votingPowerBarStyle.Width(total)
}
