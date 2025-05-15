package tui

import (
	"suitop/internal/types"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the UI state
type Model struct {
	epoch                              uint64
	checkpointSeq                      uint64
	totalWithSig                       uint64
	signedPower                        int
	totalPower                         int
	committee                          []types.ValidatorInfo
	stats                              map[string]types.ValidatorStats
	validatorBar                       progress.Model
	votingPowerBar                     progress.Model
	checkpoints                        map[uint64]types.CheckpointInfo
	latestSeq                          uint64
	width                              int // Terminal width
	height                             int // Terminal height
	ready                              bool
	leftWidth, rightWidth, middleWidth int
	NetworkName                        string // Added to display the current network

	// Calculated fields for progress bars
	signedValidators  int
	totalValidators   int
	signedVotingPower int
	totalVotingPower  int
}

// New creates a new bubble tea model
func New(epochID uint64, validators []types.ValidatorInfo, networkName string) Model {
	// Create progress bars with different colors
	validatorBar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithScaledGradient("#5fff87", "#48BB78"),
	)
	// Set percentage format to show one decimal place
	validatorBar.PercentFormat = " %.1f%%"

	votingPowerBar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithScaledGradient("#ffdf5d", "#ECC94B"),
	)
	// Set percentage format to show one decimal place
	votingPowerBar.PercentFormat = " %.1f%%"

	return Model{
		epoch:             epochID,
		committee:         validators,
		stats:             make(map[string]types.ValidatorStats),
		validatorBar:      validatorBar,
		votingPowerBar:    votingPowerBar,
		checkpoints:       make(map[uint64]types.CheckpointInfo),
		width:             0,
		height:            0,
		ready:             false,
		totalValidators:   len(validators),
		signedValidators:  0,
		totalVotingPower:  0,
		signedVotingPower: 0,
		NetworkName:       networkName,
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
	m.signedPower = msg.SignedPower
	m.totalPower = msg.TotalPower
	m.committee = msg.Committee
	m.stats = msg.Stats

	// Update calculated fields
	m.totalValidators = len(m.committee)
	m.signedValidators = countSignaturesForCheckpoint(*m)
	m.signedVotingPower = m.signedPower
	m.totalVotingPower = m.totalPower
}

// SubscribeToStateUpdates creates a command that listens for state updates
func SubscribeToStateUpdates(sub chan types.SnapshotMsg) tea.Cmd {
	return func() tea.Msg {
		msg := <-sub
		return SnapshotMsg(msg)
	}
}
