package tui

import (
	"suitop/internal/types"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles the model updates based on messages received
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle key presses
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		gap := 2                      // 1‑char gap between panels + border rounding
		m.leftWidth = m.width/3 - gap // integer division -> left column for main content
		m.rightWidth = m.leftWidth    // for main content
		m.middleWidth = m.leftWidth   // for main content

		interior := func(boxWidth int) int {
			border := 2 // lipgloss border adds 2 chars (1 left, 1 right for RoundedBorder)
			pad := 4    // Padding(1,2) means 2 units on left and 2 on right for content = 4
			return boxWidth - border - pad
		}

		// Calculate width for progress bars. They are in progressPanelStyle.
		// progressPanelStyle gets width (m.width - 4) / 2 from AdjustStyles.
		progressPanelHostWidth := (m.width - 4) / 2
		barWidth := interior(progressPanelHostWidth)

		m.validatorBar.Width = barWidth
		m.votingPowerBar.Width = barWidth

		// update style widths
		AdjustStyles(m.width, m.leftWidth, m.middleWidth, m.rightWidth)

		m.ready = true

	case SnapshotMsg:
		// Apply the snapshot to the model state
		m.applySnapshot(msg)
	}

	// Handle progress bar updates
	var validatorBarCmd, votingPowerBarCmd tea.Cmd
	validatorModel, validatorBarCmd := m.validatorBar.Update(msg)
	votingPowerModel, votingPowerBarCmd := m.votingPowerBar.Update(msg)

	m.validatorBar = validatorModel.(progress.Model)
	m.votingPowerBar = votingPowerModel.(progress.Model)

	if validatorBarCmd != nil {
		cmd = validatorBarCmd
	}
	if votingPowerBarCmd != nil {
		cmd = tea.Batch(cmd, votingPowerBarCmd)
	}

	return m, cmd
}

// Listen for state updates from a channel
func (m Model) Listen(sub chan types.SnapshotMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-sub
		if !ok {
			// Channel closed
			return nil
		}
		return SnapshotMsg(msg)
	}
}
