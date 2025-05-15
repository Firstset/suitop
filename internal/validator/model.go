package validator

import (
	"strings"
	"suitop/internal/types"
)

// ValidatorInfo holds static information about a validator in a specific epoch's committee.
// This is a copy of the types.ValidatorInfo struct for internal usage.
type ValidatorInfo struct {
	Name                string
	SuiAddress          string // Used as a persistent key for stats
	ProtocolPubkeyBytes string // BLS key from committee info / system state
	BitmapIndex         int    // Index from suix_getCommitteeInfo (0 to N-1), for bitmap lookup
	VotingPower         int    // Voting power from committee info
}

// ToTypesInfo converts this ValidatorInfo to a types.ValidatorInfo
func (v ValidatorInfo) ToTypesInfo() types.ValidatorInfo {
	return types.ValidatorInfo{
		Name:                v.Name,
		SuiAddress:          v.SuiAddress,
		ProtocolPubkeyBytes: v.ProtocolPubkeyBytes,
		BitmapIndex:         v.BitmapIndex,
		VotingPower:         v.VotingPower,
	}
}

// FromTypesInfo creates a ValidatorInfo from a types.ValidatorInfo
func FromTypesInfo(v types.ValidatorInfo) ValidatorInfo {
	return ValidatorInfo{
		Name:                v.Name,
		SuiAddress:          v.SuiAddress,
		ProtocolPubkeyBytes: v.ProtocolPubkeyBytes,
		BitmapIndex:         v.BitmapIndex,
		VotingPower:         v.VotingPower,
	}
}

// shortPubKey is a helper function to shorten pubkey for logging.
// It's placed here as it operates on validator-related data (pubkeys).
func ShortPubKey(pubKey string) string {
	if len(pubKey) > 10 {
		return pubKey[:10]
	}
	return pubKey
}

// NewValidatorInfo creates a new ValidatorInfo struct.
// This can be expanded if more complex initialization is needed.
func NewValidatorInfo(name, suiAddress, protocolPubkeyBytes string, bitmapIndex int, votingPower int) ValidatorInfo {
	return ValidatorInfo{
		Name:                strings.TrimSpace(name),
		SuiAddress:          suiAddress,
		ProtocolPubkeyBytes: protocolPubkeyBytes,
		BitmapIndex:         bitmapIndex,
		VotingPower:         votingPower,
	}
}
