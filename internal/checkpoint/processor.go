package checkpoint

import (
	"context"
	"fmt"
	"log"
	"sort"

	"suitop/internal/config"
	val "suitop/internal/validator" // Alias for validator package

	// Assuming pb types will be accessible via this path after go.mod setup
	// For CheckpointData type from subscription
	rpcPb "suitop/pb/sui/rpc/v2beta" // For Checkpoint type for epoch field reference if different. Removed as unused for now.
	// For Checkpoint type for epoch field reference if different
)

// Processor handles the main checkpoint processing loop.
type Processor struct {
	valLoader    *val.Loader
	statsManager *StatsManager
	cfg          config.ProcessorConfig // Placeholder for future config
	currentEpoch uint64
	committee    []val.ValidatorInfo
}

// NewProcessor creates a new checkpoint processor.
func NewProcessor(valLoader *val.Loader, statsManager *StatsManager, cfg config.ProcessorConfig) *Processor {
	return &Processor{
		valLoader:    valLoader,
		statsManager: statsManager,
		cfg:          cfg,
	}
}

// Run starts the checkpoint processing loop.
// It takes the initial epoch and committee as arguments.
func (p *Processor) Run(ctx context.Context, initialEpoch uint64, initialCommittee []val.ValidatorInfo, checkpointStream <-chan *rpcPb.Checkpoint) {
	p.currentEpoch = initialEpoch
	p.committee = initialCommittee

	for {
		select {
		case receivedCheckpoint, ok := <-checkpointStream:
			if !ok {
				log.Println("Checkpoint channel closed, exiting processor loop.")
				return
			}

			// Check if the checkpoint has a signature
			if receivedCheckpoint.GetSignature() == nil {
				// log.Printf("Checkpoint %d received without signature data.", receivedCheckpoint.GetSequenceNumber())
				continue // Skip checkpoints without signatures for uptime calculation
			}

			p.statsManager.IncrementTotalCheckpointsWithSig()

			// Assuming Summary has an Epoch field
			checkpointEpochVal := receivedCheckpoint.GetSummary().GetEpoch()

			// Epoch change detection and committee reload
			if checkpointEpochVal > p.currentEpoch {
				fmt.Printf("\nEpoch changed from %d to %d. Reloading committee...\n", p.currentEpoch, checkpointEpochVal)
				p.currentEpoch = checkpointEpochVal
				newCommittee, newLoadedEpoch, err := p.valLoader.LoadEpochValidatorData(ctx, checkpointEpochVal)
				if err != nil {
					log.Printf("Failed to load committee for new epoch %d: %v. Continuing with old committee.", checkpointEpochVal, err)
				} else {
					p.committee = newCommittee
					p.currentEpoch = newLoadedEpoch // Ensure currentEpoch matches what was loaded
					p.statsManager.InitializeCommitteeStats(newCommittee)
					fmt.Printf("Successfully reloaded committee for epoch %d with %d validators.\n", p.currentEpoch, len(p.committee))
				}
			}

			p.statsManager.ResetSignedCurrent(p.committee)

			// Assuming ValidatorAggregatedSignature has a bitmap of validator indices
			bitmap := receivedCheckpoint.GetSignature().GetBitmap()
			committeeSize := len(p.committee)

			for _, idxInBitmap := range bitmap {
				if idxInBitmap >= uint32(committeeSize) {
					log.Panicf(
						"CRITICAL: Bitmap index %d is out of bounds for committee size %d (epoch %d, checkpoint %d). Committee indices should be 0 to %d.",
						idxInBitmap,
						committeeSize,
						p.currentEpoch,
						receivedCheckpoint.GetSequenceNumber(),
						committeeSize-1,
					)
				}
			}

			for _, valInfo := range p.committee {
				if IsValidatorSigned(bitmap, valInfo.BitmapIndex) { // IsValidatorSigned is in this package
					p.statsManager.UpdateValidatorSigned(valInfo.SuiAddress)
				} else {
					// If validator was not in stats map (e.g. committee changed mid-checkpoint processing before stats init for new members)
					// This is less likely with current flow where stats are init/updated after committee load.
					// The original code had a warning here, but UpdateValidatorSigned handles the non-existence silently by not updating.
				}
			}

			p.printReport(receivedCheckpoint.GetSequenceNumber())

		case <-ctx.Done():
			log.Println("Context done, exiting processor loop.")
			return
		}
	}
}

func (p *Processor) printReport(checkpointSeqNum uint64) {
	totalCheckpointsWithSig := p.statsManager.GetTotalCheckpointsWithSig()
	fmt.Printf("\n--- Checkpoint #%d (Epoch: %d, Total w/Sig: %d) ---\n",
		checkpointSeqNum, p.currentEpoch, totalCheckpointsWithSig)

	displayCommittee := make([]val.ValidatorInfo, len(p.committee))
	copy(displayCommittee, p.committee)
	sort.Slice(displayCommittee, func(i, j int) bool {
		return displayCommittee[i].Name < displayCommittee[j].Name
	})

	var linesToPrint []string
	for _, valInfo := range displayCommittee {
		stats, _, ok := p.statsManager.GetStats(valInfo.SuiAddress)
		if !ok {
			log.Printf("Warning: Validator %s (SuiAddress: %s) in committee but missing from stats for reporting.", valInfo.Name, valInfo.SuiAddress)
			continue
		}

		statusIcon := "❌"
		if stats.SignedCurrent {
			statusIcon = "✅"
		}

		uptimePercent := 0.0
		if totalCheckpointsWithSig > 0 {
			uptimePercent = (float64(stats.AttestedCount) / float64(totalCheckpointsWithSig)) * 100
		}
		linesToPrint = append(linesToPrint, fmt.Sprintf("%s %-40s - Attested: %6.2f%% (%4d/%4d)",
			statusIcon, valInfo.Name, uptimePercent, stats.AttestedCount, totalCheckpointsWithSig))
	}

	for j := 0; j < len(linesToPrint); j += 2 {
		fmt.Print(linesToPrint[j])
		if j+1 < len(linesToPrint) {
			fmt.Printf("   |   %s\n", linesToPrint[j+1])
		} else {
			fmt.Println()
		}
	}
}
