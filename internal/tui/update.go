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

		gap := 4                          // 1‑char gap between panels + border rounding
		m.leftWidth = (m.width - gap) / 2 // integer division -> left column
		m.rightWidth = m.width - m.leftWidth - gap

		interior := func(boxWidth int) int {
			border := 2 // lipgloss border adds 2 chars
			pad := 4    // we use Padding(1,2) → left+right = 4
			return boxWidth - border - pad
		}

		barWidth := interior(m.rightWidth)

		m.validatorBar.Width = barWidth
		m.votingPowerBar.Width = barWidth

		// update style widths
		AdjustStyles(m.width, m.leftWidth, m.rightWidth)

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
