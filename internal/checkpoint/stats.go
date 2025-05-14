package checkpoint

import (
	"suitop/internal/types"
	valmodel "suitop/internal/validator"
)

// ValidatorStats tracks the uptime statistics for a validator.
// This is a copy of types.ValidatorStats for internal usage.
type ValidatorStats struct {
	AttestedCount uint64
	SignedCurrent bool // Did they sign the most recently processed checkpoint?
}

// ToTypesStats converts a ValidatorStats to types.ValidatorStats
func (v ValidatorStats) ToTypesStats() types.ValidatorStats {
	return types.ValidatorStats{
		AttestedCount: v.AttestedCount,
		SignedCurrent: v.SignedCurrent,
	}
}

// FromTypesStats creates a ValidatorStats from types.ValidatorStats
func FromTypesStats(v types.ValidatorStats) ValidatorStats {
	return ValidatorStats{
		AttestedCount: v.AttestedCount,
		SignedCurrent: v.SignedCurrent,
	}
}

// StatsManager manages the statistics for all validators.
type StatsManager struct {
	validatorStats          map[string]ValidatorStats // Keyed by validator SuiAddress
	totalCheckpointsWithSig uint64
}

// NewStatsManager creates a new StatsManager.
func NewStatsManager() *StatsManager {
	return &StatsManager{
		validatorStats: make(map[string]ValidatorStats),
	}
}

// InitializeCommitteeStats sets up initial stats for a new committee.
// It preserves stats for validators already known.
func (sm *StatsManager) InitializeCommitteeStats(committee []valmodel.ValidatorInfo) {
	for _, valInfo := range committee {
		if _, exists := sm.validatorStats[valInfo.SuiAddress]; !exists {
			sm.validatorStats[valInfo.SuiAddress] = ValidatorStats{AttestedCount: 0, SignedCurrent: false}
		}
	}
}

// ResetSignedCurrent resets the SignedCurrent flag for all validators in the provided committee.
func (sm *StatsManager) ResetSignedCurrent(committee []valmodel.ValidatorInfo) {
	for _, valInfo := range committee {
		if stats, ok := sm.validatorStats[valInfo.SuiAddress]; ok {
			stats.SignedCurrent = false
			sm.validatorStats[valInfo.SuiAddress] = stats
		}
	}
}

// UpdateValidatorSigned updates stats for a validator who signed the current checkpoint.
func (sm *StatsManager) UpdateValidatorSigned(suiAddress string) {
	if stats, ok := sm.validatorStats[suiAddress]; ok {
		stats.SignedCurrent = true
		stats.AttestedCount++
		sm.validatorStats[suiAddress] = stats
	}
}

// IncrementTotalCheckpointsWithSig increments the total count of checkpoints processed that had signatures.
func (sm *StatsManager) IncrementTotalCheckpointsWithSig() {
	sm.totalCheckpointsWithSig++
}

// GetStats returns the stats for a specific validator and the total processed checkpoints with signatures.
func (sm *StatsManager) GetStats(suiAddress string) (ValidatorStats, uint64, bool) {
	stats, ok := sm.validatorStats[suiAddress]
	return stats, sm.totalCheckpointsWithSig, ok
}

// GetAllStats returns the entire map of validator stats.
// Useful for reporting, but be mindful of concurrent access if the map is modified elsewhere.
func (sm *StatsManager) GetAllStats() map[string]ValidatorStats {
	// Consider returning a copy if concurrent modification is a concern
	return sm.validatorStats
}

// GetTotalCheckpointsWithSig returns the total number of checkpoints processed with signatures.
func (sm *StatsManager) GetTotalCheckpointsWithSig() uint64 {
	return sm.totalCheckpointsWithSig
}
