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

	AdjustStyles(m.width, m.leftWidth, m.middleWidth, m.rightWidth)

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
	networkInfo := fmt.Sprintf("Network: %s", m.NetworkName)
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
		networkInfo,
		epochInfo,
		checkpointInfo,
		totalCheckpoints,
		committeeSize,
		timeInfo,
	)

	return infoPanelStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
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
		return mainContentContainerStyle.Render("Waiting for validator data...")
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

	// Split rows into three groups for three columns
	totalRows := len(allRows)
	rowsPerColumn := totalRows / 3
	remainder := totalRows % 3

	leftRows := allRows[:rowsPerColumn]
	middleRows := allRows[rowsPerColumn : 2*rowsPerColumn]
	rightRows := allRows[2*rowsPerColumn:]

	// Adjust for remainder
	if remainder == 1 {
		rightRows = allRows[2*rowsPerColumn+1:]
		middleRows = allRows[rowsPerColumn : 2*rowsPerColumn+1]
	} else if remainder == 2 {
		rightRows = allRows[2*rowsPerColumn+2:]
		middleRows = allRows[rowsPerColumn : 2*rowsPerColumn+1]
		leftRows = allRows[:rowsPerColumn+1]
	}

	// Table columns definition
	columns := []table.Column{
		{Title: "Status", Width: 6},
		{Title: "Validator", Width: 20},
		{Title: "Signed %", Width: 20},
		//{Title: "Signed/Total", Width: 15},
		//{Title: "Bar", Width: 30},
	}

	// Calculate height for tables inside the container.
	// The container (mainContentContainerStyle) is a copy of boxStyle, which has Padding(1,2).
	// This means 1 line top padding and 1 line bottom padding.
	// So, tables should be 2 lines shorter than before to fit inside.
	tableHeight := m.height - 14

	// Create left table
	leftTable := table.New(
		table.WithColumns(columns),
		table.WithRows(leftRows),
		table.WithHeight(tableHeight),
	)

	// Create middle table
	middleTable := table.New(
		table.WithColumns(columns),
		table.WithRows(middleRows),
		table.WithHeight(tableHeight),
	)

	// Create right table
	rightTable := table.New(
		table.WithColumns(columns),
		table.WithRows(rightRows),
		table.WithHeight(tableHeight),
	)

	// Style tables (internal styling)
	leftTable.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	middleTable.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	rightTable.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	// Get raw string views of the tables
	leftTableView := leftTable.View()
	middleTableView := middleTable.View()
	rightTableView := rightTable.View()

	// Join table views horizontally with a single space separator
	joinedTablesView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftTableView,
		" ", // Spacer
		middleTableView,
		" ", // Spacer
		rightTableView,
	)

	// Render the joined tables inside the single main container
	return mainContentContainerStyle.Render(joinedTablesView)
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
