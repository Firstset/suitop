package tui

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// View renders the current model as a string
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	AdjustStyles(m.width, m.leftWidth, m.rightWidth)

	headerRow := renderHeaderRow(m)
	mainContent := renderMainContent(m)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerRow,
		mainContent,
	)
}

// renderHeaderRow creates the top row with two panels side by side
func renderHeaderRow(m Model) string {
	// Left panel with info
	infoPanel := renderInfoPanel(m)

	// Right panel with progress bars
	progressPanel := renderProgressPanel(m)

	// Join them horizontally
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		infoPanel,
		progressPanel,
	)
}

// renderInfoPanel creates the info panel with epoch, checkpoint, and stats
func renderInfoPanel(m Model) string {
	// Format the basic information
	epochInfo := fmt.Sprintf("Epoch: %d", m.epoch)
	checkpointInfo := fmt.Sprintf("Checkpoint: %d", m.checkpointSeq)
	committeeSize := fmt.Sprintf("Committee Size: %d validators", len(m.committee))
	totalCheckpoints := fmt.Sprintf("Checkpoint samples: %d", m.totalWithSig)

	// Format time
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	timeInfo := fmt.Sprintf("Time: %s", currentTime)

	// Join vertically
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		epochInfo,
		checkpointInfo,
		totalCheckpoints,
		committeeSize,
		timeInfo,
	)

	return infoPanelStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			"SUI Network Statistics",
			content,
		),
	)
}

// renderProgressPanel creates the progress bars panel
func renderProgressPanel(m Model) string {
	// Calculate percentages
	validatorPercent := 0.0
	if m.totalValidators > 0 {
		validatorPercent = float64(m.signedValidators) / float64(m.totalValidators) * 100
	}

	votingPercent := 0.0
	if m.totalVotingPower > 0 {
		votingPercent = float64(m.signedVotingPower) / float64(m.totalVotingPower) * 100
	}

	// Create compact labels
	valLabel := fmt.Sprintf("✓ Validators:")
	voteLabel := fmt.Sprintf("🗳 Voting power:")

	// Render the progress bars with the calculated percentages
	validatorContent := m.validatorBar.ViewAs(validatorPercent / 100) // ViewAs expects 0.0-1.0
	votingContent := m.votingPowerBar.ViewAs(votingPercent / 100)     // ViewAs expects 0.0-1.0

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(valLabel),
		validatorContent,
		"", // Add space between bars
		lipgloss.NewStyle().Bold(true).Render(voteLabel),
		votingContent,
	)

	return progressPanelStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			content,
		),
	)
}

func renderBar(uptime float64) string {
	// Define the total width of the bar content (excluding brackets)
	const barWidth = 10

	// Calculate how many filled characters to show based on uptime percentage
	filled := int(uptime * barWidth)
	if filled > barWidth {
		filled = barWidth // Cap at maximum width
	}

	// Build the bar string with 'x' for filled parts and spaces for empty parts
	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "▓"
		} else {
			bar += " "
		}
	}
	bar += "]"

	return bar
}

// renderMainContent creates the main body with the validator table in two columns
func renderMainContent(m Model) string {
	// Only initialize the table if we have committee data
	if m.committee == nil || len(m.committee) == 0 {
		return boxStyle.Render("Waiting for validator data...")
	}

	// Build all validator rows
	var allRows []table.Row
	for _, validator := range m.committee {
		id := validator.SuiAddress
		stats, ok := m.stats[id]

		// Default values if stats not found
		status := "❓"
		uptimePercent := "N/A"
		//signedRatio := "N/A"
		var uptime float64 = 0

		if ok {
			// Calculate uptime percentage
			if m.totalWithSig > 0 {
				uptime = float64(stats.AttestedCount) / float64(m.totalWithSig)
			}
			uptimePercent = fmt.Sprintf("%.2f%%", uptime*100)

			// Format signed ratio
			//signedRatio = fmt.Sprintf("%d/%d", stats.AttestedCount, m.totalWithSig)

			// Set status emoji based on signature presence for current checkpoint
			if stats.SignedCurrent {
				status = "✅"
			} else {
				status = "❌"
			}
		}

		// Add the row
		allRows = append(allRows, table.Row{
			status,
			validator.Name,
			renderBar(uptime) + " " + uptimePercent,
			//signedRatio,
			//renderBar(uptime),
		})
	}

	// Sort rows by validator name
	sort.Slice(allRows, func(i, j int) bool {
		return allRows[i][1] < allRows[j][1]
	})

	// Split rows into two groups for two columns
	leftRows := allRows[:len(allRows)/2]
	rightRows := allRows[len(allRows)/2:]

	// Table columns definition
	columns := []table.Column{
		{Title: "Status", Width: 6},
		{Title: "Validator", Width: 25},
		{Title: "Signed %", Width: 25},
		//{Title: "Signed/Total", Width: 15},
		//{Title: "Bar", Width: 30},
	}

	// Create left table
	leftTable := table.New(
		table.WithColumns(columns),
		table.WithRows(leftRows),
		table.WithHeight(m.height-14), // Adjust height to fit in screen (14 fits exactly right now)
	)

	// Create right table
	rightTable := table.New(
		table.WithColumns(columns),
		table.WithRows(rightRows),
		table.WithHeight(m.height-14), // Adjust height to fit in screen (14 fits exactly right now)
	)

	// Style both tables
	leftTable.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	rightTable.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	// Render left and right tables
	leftTableView := leftPanelStyle.Render(leftTable.View())
	rightTableView := rightPanelStyle.Render(rightTable.View())

	// Join tables horizontally
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftTableView,
		rightTableView,
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
