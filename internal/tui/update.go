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
		// Update the model with the new window size
		m.width, m.height = msg.Width, msg.Height

		// Also update the progress bar
		m.progressBar.Width = msg.Width - 20 // Adjust for padding

		// Mark the model as ready to render once we have window dimensions
		m.ready = true

		// Update the table (will be fully implemented in view.go)
		// Adjust column widths based on the new window size

	case SnapshotMsg:
		// Apply the snapshot to the model state
		m.applySnapshot(msg)
	}

	// Handle progress bar updates
	progressModel, progressCmd := m.progressBar.Update(msg)
	m.progressBar = progressModel.(progress.Model)
	if progressCmd != nil {
		cmd = progressCmd
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
