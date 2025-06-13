package checkpoint

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"suitop/internal/config"
	"suitop/internal/types"
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
	plainMode    bool // When true, output to stdout instead of TUI
	dataset      *DatasetManager
	reportCount  int
}

// NewProcessor creates a new checkpoint processor.
func NewProcessor(valLoader *val.Loader, statsManager *StatsManager, cfg config.ProcessorConfig, plainMode bool, dataset *DatasetManager) *Processor {
	return &Processor{
		valLoader:    valLoader,
		statsManager: statsManager,
		cfg:          cfg,
		plainMode:    plainMode,
		dataset:      dataset,
	}
}

// Run starts the checkpoint processing loop.
// It takes the initial epoch and committee as arguments.
// The optional uiChan parameter sends state snapshots to the UI if provided.
func (p *Processor) Run(ctx context.Context, initialEpoch uint64, initialCommittee []val.ValidatorInfo,
	checkpointStream <-chan *rpcPb.Checkpoint, uiChan chan<- types.SnapshotMsg) {
	p.currentEpoch = initialEpoch
	p.committee = initialCommittee

	for {
		select {
		case receivedCheckpoint, ok := <-checkpointStream:
			if !ok {
				log.Println("Checkpoint channel closed, exiting processor loop.")
				if p.dataset != nil {
					p.dataset.Close()
				}
				return
			}

			// Check if the checkpoint has a signature
			if receivedCheckpoint.GetSignature() == nil {
				// log.Printf("Checkpoint %d received without signature data.", receivedCheckpoint.GetSequenceNumber())
				continue // Skip checkpoints without signatures for uptime calculation
			}

			p.statsManager.IncrementTotalCheckpointsWithSig()

			// Epoch value is stored inside the validator aggregated signature
			// which is guaranteed to be present in the subscription
			// because we request the full signature message.
			checkpointEpochVal := receivedCheckpoint.GetSignature().GetEpoch()

			// Epoch change detection and committee reload
			if checkpointEpochVal > p.currentEpoch {
				if p.plainMode {
					fmt.Printf("\nEpoch changed from %d to %d. Reloading committee...\n", p.currentEpoch, checkpointEpochVal)
				} else {
					log.Printf("Epoch changed from %d to %d. Reloading committee...", p.currentEpoch, checkpointEpochVal)
				}

				p.currentEpoch = checkpointEpochVal
				newCommittee, newLoadedEpoch, err := p.valLoader.LoadEpochValidatorData(ctx, checkpointEpochVal)
				if err != nil {
					log.Printf("Failed to load committee for new epoch %d: %v. Continuing with old committee.", checkpointEpochVal, err)
				} else {
					p.committee = newCommittee
					p.currentEpoch = newLoadedEpoch // Ensure currentEpoch matches what was loaded
					p.statsManager.InitializeCommitteeStats(newCommittee)

					if p.plainMode {
						fmt.Printf("Successfully reloaded committee for epoch %d with %d validators.\n", p.currentEpoch, len(p.committee))
					} else {
						log.Printf("Successfully reloaded committee for epoch %d with %d validators.", p.currentEpoch, len(p.committee))
					}
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

			if p.dataset != nil {
				p.dataset.RecordCheckpoint(p.currentEpoch, receivedCheckpoint.GetSequenceNumber(), bitmap, p.committee)
				p.reportCount++
			}

			// If uiChan is provided, send a snapshot to the UI
			if uiChan != nil {
				// Convert the internal validator info to the types package format
				committeeForUI := make([]types.ValidatorInfo, len(p.committee))
				for i, v := range p.committee {
					committeeForUI[i] = v.ToTypesInfo()
				}

				// Convert the stats map to the types package format
				statsForUI := make(map[string]types.ValidatorStats)
				allStats := p.statsManager.GetAllStats()
				for k, v := range allStats {
					statsForUI[k] = v.ToTypesStats()
				}

				// Calculate voting power metrics
				totalPower := 0
				signedPower := 0
				for _, v := range p.committee {
					totalPower += v.VotingPower
					if p.statsManager.IsSigned(v.SuiAddress) {
						signedPower += v.VotingPower
					}
				}

				uiChan <- types.SnapshotMsg{
					Epoch:         p.currentEpoch,
					CheckpointSeq: receivedCheckpoint.GetSequenceNumber(),
					TotalWithSig:  p.statsManager.GetTotalCheckpointsWithSig(),
					SignedPower:   signedPower,
					TotalPower:    totalPower,
					Committee:     committeeForUI,
					Stats:         statsForUI,
				}
			} else {
				if p.dataset != nil {
					if p.reportCount%10 == 0 {
						p.printReport(receivedCheckpoint.GetSequenceNumber(), os.Stdout)
						fmt.Println("[dataset mode] Press 'q' then Enter to stop and save dataset.")
					}
				} else {
					p.printReport(receivedCheckpoint.GetSequenceNumber(), os.Stdout)
				}
			}

		case <-ctx.Done():
			log.Println("Context done, exiting processor loop.")
			if p.dataset != nil {
				p.dataset.Close()
			}
			return
		}
	}
}

// printReport outputs a formatted report of the current validator status to the provided writer
func (p *Processor) printReport(checkpointSeqNum uint64, w io.Writer) {
	totalCheckpointsWithSig := p.statsManager.GetTotalCheckpointsWithSig()
	fmt.Fprintf(w, "\n--- Checkpoint #%d (Epoch: %d, Total w/Sig: %d) ---\n",
		checkpointSeqNum, p.currentEpoch, totalCheckpointsWithSig)

	// Calculate voting power metrics
	totalPower := 0
	signedPower := 0
	for _, v := range p.committee {
		totalPower += v.VotingPower
		if p.statsManager.IsSigned(v.SuiAddress) {
			signedPower += v.VotingPower
		}
	}

	// Print voting power stats if available
	if totalPower > 0 {
		pct := float64(signedPower) / float64(totalPower) * 100
		fmt.Fprintf(w, "Voting power signed: %.2f%% (%d/%d)\n", pct, signedPower, totalPower)
	}

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
		fmt.Fprint(w, linesToPrint[j])
		if j+1 < len(linesToPrint) {
			fmt.Fprintf(w, "   |   %s\n", linesToPrint[j+1])
		} else {
			fmt.Fprintln(w)
		}
	}
}
