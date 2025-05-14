package tui

import (
	"suitop/internal/types"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the UI state
type Model struct {
	epoch         uint64
	checkpointSeq uint64
	totalWithSig  uint64
	committee     []types.ValidatorInfo
	stats         map[string]types.ValidatorStats
	progressBar   progress.Model
	checkpoints   map[uint64]types.CheckpointInfo
	latestSeq     uint64
	width         int // Terminal width
	height        int // Terminal height
	ready         bool
}

// New creates a new bubble tea model
func New(epochID uint64, validators []types.ValidatorInfo) Model {
	progressBar := progress.New(progress.WithDefaultGradient())

	// Initialize with zero values for width/height
	// They'll be updated when WindowSizeMsg is received
	return Model{
		epoch:       epochID,
		committee:   validators,
		stats:       make(map[string]types.ValidatorStats),
		progressBar: progressBar,
		checkpoints: make(map[uint64]types.CheckpointInfo),
		width:       0,
		height:      0,
		ready:       false,
	}
}

// Init initializes the bubble tea model
func (m Model) Init() tea.Cmd {
	return nil
}

// applySnapshot updates the model's state with new snapshot data
func (m *Model) applySnapshot(msg SnapshotMsg) {
	m.epoch = msg.Epoch
	m.checkpointSeq = msg.CheckpointSeq
	m.totalWithSig = msg.TotalWithSig
	m.committee = msg.Committee
	m.stats = msg.Stats
}

// SubscribeToStateUpdates creates a command that listens for state updates
func SubscribeToStateUpdates(sub chan types.SnapshotMsg) tea.Cmd {
	return func() tea.Msg {
		msg := <-sub
		return SnapshotMsg(msg)
	}
}
