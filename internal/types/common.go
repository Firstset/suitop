package types

// ValidatorInfo holds static information about a validator in a specific epoch's committee.
type ValidatorInfo struct {
	Name                string
	SuiAddress          string // Used as a persistent key for stats
	ProtocolPubkeyBytes string // BLS key from committee info / system state
	BitmapIndex         int    // Index from suix_getCommitteeInfo (0 to N-1), for bitmap lookup
}

// ValidatorStats tracks the uptime statistics for a validator.
type ValidatorStats struct {
	AttestedCount uint64
	SignedCurrent bool // Did they sign the most recently processed checkpoint?
}

// CheckpointInfo contains information about a processed checkpoint
type CheckpointInfo struct {
	Sequence        uint64
	Timestamp       int64
	SignaturesCount uint64
	ValidatorCount  uint64
}

// SnapshotMsg represents a state snapshot from the core logic that is sent to the UI
type SnapshotMsg struct {
	Epoch         uint64
	CheckpointSeq uint64
	TotalWithSig  uint64
	Committee     []ValidatorInfo
	Stats         map[string]ValidatorStats
}
