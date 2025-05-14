package validator

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"suitop/internal/config"
	"suitop/internal/rpc"
)

// Loader handles loading validator information.
type Loader struct {
	rpcClient *rpc.Client
}

// NewLoader creates a new validator loader.
func NewLoader(rpcCfg config.RPCClientConfig) *Loader {
	return &Loader{
		rpcClient: rpc.NewClient(rpcCfg),
	}
}

// LoadEpochValidatorData fetches committee and validator metadata for a given epoch.
// If targetEpoch is 0, it first fetches the latest epoch.
func (l *Loader) LoadEpochValidatorData(ctx context.Context, targetEpoch uint64) ([]ValidatorInfo, uint64, error) {
	log.Println("Loading epoch validator data...")

	var epochToQueryStr string
	var actualEpoch uint64

	// Step 0: Determine current epoch if targetEpoch is not specified
	if targetEpoch == 0 {
		log.Println("Target epoch not specified, fetching latest Sui system state to determine current epoch...")
		systemState, err := l.rpcClient.GetLatestSuiSystemState(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("error fetching system state (pre-fetch): %w", err)
		}
		if systemState.Epoch == "" {
			return nil, 0, fmt.Errorf("epoch not found in pre-fetch system state response")
		}
		actualEpoch, err = strconv.ParseUint(systemState.Epoch, 10, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("error parsing epoch from pre-fetch system state response: %w", err)
		}
		epochToQueryStr = systemState.Epoch
		log.Printf("Determined latest epoch to be %d for querying committee info.", actualEpoch)
	} else {
		actualEpoch = targetEpoch
		epochToQueryStr = strconv.FormatUint(targetEpoch, 10)
		log.Printf("Using specified target epoch %d for querying committee info.", actualEpoch)
	}

	// Step 1: Call suix_getCommitteeInfo for the determined/specified epoch
	committeeInfo, err := l.rpcClient.GetCommitteeInfo(ctx, epochToQueryStr)
	if err != nil {
		return nil, 0, fmt.Errorf("error fetching committee info for epoch %s: %w", epochToQueryStr, err)
	}

	committeePubKeysOrdered := make([]string, len(committeeInfo.Validators))
	for i, valData := range committeeInfo.Validators {
		if len(valData) > 0 && valData[0] != "" {
			committeePubKeysOrdered[i] = valData[0] // valData[0] is protocolPubkeyBytes
		} else {
			return nil, 0, fmt.Errorf("malformed validator data or empty pubkey in committee info for epoch %s, index %d", epochToQueryStr, i)
		}
	}
	log.Printf("Successfully fetched %d validator public keys (committee order) for epoch %s.", len(committeePubKeysOrdered), epochToQueryStr)

	// Step 2: Call suix_getLatestSuiSystemState to get validator metadata
	log.Printf("Fetching full Sui system state to get validator metadata (potentially for epoch %s)...", epochToQueryStr)
	systemStateMetadata, err := l.rpcClient.GetLatestSuiSystemState(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("error fetching system state for metadata: %w", err)
	}

	latestSystemStateEpoch, err := strconv.ParseUint(systemStateMetadata.Epoch, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing epoch from system state metadata response: %w", err)
	}

	if latestSystemStateEpoch != actualEpoch {
		log.Printf("Warning: Epoch from suix_getCommitteeInfo (%d) differs from suix_getLatestSuiSystemState epoch (%d). Metadata might be from a slightly different epoch. Proceeding with committee epoch %d.", actualEpoch, latestSystemStateEpoch, actualEpoch)
	}

	// Step 3: Join committee order with metadata
	activeValidatorsMap := make(map[string]rpc.ActiveValidatorJSON)
	for _, valMeta := range systemStateMetadata.ActiveValidators {
		if valMeta.ProtocolPubkeyBytes == "" {
			log.Printf("Warning: Active validator %s (SuiAddress: %s) is missing ProtocolPubkeyBytes in system state response (epoch %d). It will be skipped for metadata lookup.", valMeta.Name, valMeta.SuiAddress, latestSystemStateEpoch)
			continue
		}
		activeValidatorsMap[valMeta.ProtocolPubkeyBytes] = valMeta
	}

	var committee []ValidatorInfo
	for bitmapIdx, pubKey := range committeePubKeysOrdered {
		meta, found := activeValidatorsMap[pubKey]
		if !found {
			log.Printf("Warning: Validator with ProtocolPubkeyBytes %s (BitmapIndex %d) from committee info (epoch %d) not found in activeValidators from latest system state (epoch %d). Using placeholder info.", ShortPubKey(pubKey), bitmapIdx, actualEpoch, latestSystemStateEpoch)
			committee = append(committee, ValidatorInfo{
				Name:                fmt.Sprintf("Unknown Validator (Pubkey: %s...)", ShortPubKey(pubKey)),
				SuiAddress:          fmt.Sprintf("unknown-sui-address-for-%s", ShortPubKey(pubKey)),
				ProtocolPubkeyBytes: pubKey,
				BitmapIndex:         bitmapIdx,
			})
			continue
		}
		committee = append(committee, NewValidatorInfo(meta.Name, meta.SuiAddress, pubKey, bitmapIdx))
	}

	log.Printf("Successfully loaded and merged data for %d validators for epoch %d.", len(committee), actualEpoch)
	return committee, actualEpoch, nil
}
