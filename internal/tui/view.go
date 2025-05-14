package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// View renders the current model as a string
func (m Model) View() string {
	// Check if we have a valid window size yet
	if !m.ready {
		return "Initializing..."
	}

	// Adjust styles based on terminal size
	AdjustStyles(m.width)

	// Build the view components
	header := renderHeader(m)
	progressSection := renderProgressBar(m)
	tableSection := renderTable(m)

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		progressSection,
		tableSection,
	)
}

// renderHeader creates the header section with epoch and checkpoint info
func renderHeader(m Model) string {
	title := fmt.Sprintf("SUI Validator Dashboard - Epoch %d", m.epoch)
	info := fmt.Sprintf("Checkpoint: %d", m.checkpointSeq)

	// Combine title and info in header
	header := headerStyle.Render(title + " - " + info)

	return header
}

// renderProgressBar creates the progress bars showing validator and voting power signatures
func renderProgressBar(m Model) string {
	// Calculate percentage of validators that signed this checkpoint
	var validatorPercent float64 = 0
	if len(m.committee) > 0 {
		sigCount := countSignaturesForCheckpoint(m)
		validatorPercent = float64(sigCount) / float64(len(m.committee))
	}

	// Calculate percentage of voting power that signed this checkpoint
	var powerPercent float64 = 0
	if m.totalPower > 0 {
		powerPercent = float64(m.signedPower) / float64(m.totalPower)
	}

	// Create labels for the progress bars
	validatorLabel := fmt.Sprintf("✓ Validators signed: %.1f%%", validatorPercent*100)
	powerLabel := fmt.Sprintf("🗳 Voting-power signed: %.1f%%", powerPercent*100)

	// Render the validator progress bar
	validatorContent := m.validatorBar.ViewAs(validatorPercent)

	// Render the voting power progress bar
	powerContent := m.powerBar.ViewAs(powerPercent)

	return progressBarStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			validatorLabel,
			validatorContent,
			"", // Add a blank line for separation
			powerLabel,
			powerContent,
		),
	)
}

// renderTable creates the table of validators with their stats
func renderTable(m Model) string {
	// Set up table columns
	columns := []table.Column{
		{Title: "Status", Width: 6},
		{Title: "Validator", Width: 40},
		{Title: "Uptime %", Width: 10},
		{Title: "Signed/Total", Width: 15},
	}

	// Only initialize the table if we have committee data
	if m.committee == nil || len(m.committee) == 0 {
		return boxStyle.Render("Waiting for validator data...")
	}

	// Build table rows from committee and stats data
	var rows []table.Row
	for _, validator := range m.committee {
		id := validator.SuiAddress
		stats, ok := m.stats[id]

		// Default values if stats not found
		status := "❓"
		uptimePercent := "N/A"
		signedRatio := "N/A"

		if ok {
			// Calculate uptime percentage
			var uptime float64 = 0
			if m.totalWithSig > 0 {
				uptime = float64(stats.AttestedCount) / float64(m.totalWithSig)
			}
			uptimePercent = fmt.Sprintf("%.2f%%", uptime*100)

			// Format signed ratio
			signedRatio = fmt.Sprintf("%d/%d", stats.AttestedCount, m.totalWithSig)

			// Set status emoji based on signature presence for current checkpoint
			if stats.SignedCurrent {
				status = "✅"
			} else {
				status = "❌"
			}
		}

		// Add the row
		rows = append(rows, table.Row{
			status,
			validator.Name,
			uptimePercent,
			signedRatio,
		})
	}

	// Create and style the table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(m.height-15), // Adjust height to fit in screen
	)

	// Style the table
	t.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	return boxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			"Validator Status",
			t.View(),
		),
	)
}

// countSignaturesForCheckpoint counts how many validators have signed the current checkpoint
func countSignaturesForCheckpoint(m Model) int {
	count := 0
	for _, stats := range m.stats {
		if stats.SignedCurrent {
			count++
		}
	}
	return count
}
