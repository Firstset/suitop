package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time" // Added for http client timeout

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes" // Added for codes.Internal
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status" // For checking context canceled error code
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	subPb "suitop/pb/sui/rpc/v2alpha" // For SubscriptionService
	rpcPb "suitop/pb/sui/rpc/v2beta"  // For RpcService (GetLatestSuiSystemState) and CheckpointData
)

// JSON-RPC Structs
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ActiveValidatorJSON struct {
	SuiAddress          string `json:"suiAddress"`
	Name                string `json:"name"`
	ProtocolPubkeyBytes string `json:"protocolPubkeyBytes"` // Added to join with committee info
	// Other fields from the JSON can be added here if needed later
}

type SuiSystemStateResult struct {
	Epoch            string                `json:"epoch"`
	ActiveValidators []ActiveValidatorJSON `json:"activeValidators"`
	// Other fields from the JSON like protocolVersion, etc., can be added here
}

type JSONRPCResponseSuiSystemState struct { // Renamed for clarity
	Jsonrpc string               `json:"jsonrpc"`
	ID      int                  `json:"id"`
	Result  SuiSystemStateResult `json:"result,omitempty"`
	Error   *JSONRPCError        `json:"error,omitempty"`
}

// Structs for suix_getCommitteeInfo response
type CommitteeInfoResultJSON struct {
	Epoch      string     `json:"epoch"`
	Validators [][]string `json:"validators"` // Each inner array is [protocol_pubkey_bytes, voting_power_as_string]
}

type JSONRPCResponseCommitteeInfo struct { // New struct for CommitteeInfo
	Jsonrpc string                  `json:"jsonrpc"`
	ID      int                     `json:"id"`
	Result  CommitteeInfoResultJSON `json:"result,omitempty"`
	Error   *JSONRPCError           `json:"error,omitempty"`
}

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

// isValidatorSigned checks if the validator at a given index signed the checkpoint
// by checking if the index is present in the bitmap (list of signing validator indices).
func isValidatorSigned(bitmap []uint32, validatorIndex int) bool {
	if validatorIndex < 0 {
		// Validator indices are typically non-negative.
		return false
	}
	targetIndex := uint32(validatorIndex)
	for _, indexInBitmap := range bitmap {
		if indexInBitmap == targetIndex {
			return true
		}
	}
	return false
}

// loadEpochValidatorData fetches committee and validator metadata for a given epoch.
// It first calls suix_getCommitteeInfo to get the canonical order of validators (by protocolPubkeyBytes),
// then calls suix_getLatestSuiSystemState to get their metadata (name, suiAddress).
// The two are joined using protocolPubkeyBytes.
// If targetEpoch is 0, it first fetches the latest epoch from suix_getLatestSuiSystemState.
func loadEpochValidatorData(ctx context.Context, targetEpoch uint64) ([]ValidatorInfo, uint64, error) {
	log.Println("Loading epoch validator data...")

	rpcURL := os.Getenv("SUI_JSON_RPC_URL")
	if rpcURL == "" {
		rpcURL = "https://fullnode.mainnet.sui.io" // Default JSON-RPC endpoint
	}
	httpClient := &http.Client{Timeout: 15 * time.Second}

	var epochToQueryStr string
	var actualEpoch uint64

	// Step 0: Determine current epoch if targetEpoch is not specified
	if targetEpoch == 0 {
		log.Println("Target epoch not specified, fetching latest Sui system state to determine current epoch...")
		systemStateRequestPayload := JSONRPCRequest{
			JSONRPC: "2.0", ID: 1, Method: "suix_getLatestSuiSystemState", Params: []interface{}{},
		}
		jsonData, err := json.Marshal(systemStateRequestPayload)
		if err != nil {
			return nil, 0, fmt.Errorf("error marshalling JSON request for system state (pre-fetch): %w", err)
		}
		httpReq, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, 0, fmt.Errorf("error creating HTTP request for system state (pre-fetch): %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			return nil, 0, fmt.Errorf("error performing HTTP request for system state (pre-fetch): %w", err)
		}
		defer httpResp.Body.Close()
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, 0, fmt.Errorf("error reading HTTP response body for system state (pre-fetch): %w", err)
		}
		if httpResp.StatusCode != http.StatusOK {
			return nil, 0, fmt.Errorf("HTTP request for system state (pre-fetch) failed with status %s: %s", httpResp.Status, string(body))
		}
		var rpcSystemStateResp JSONRPCResponseSuiSystemState
		if err := json.Unmarshal(body, &rpcSystemStateResp); err != nil {
			return nil, 0, fmt.Errorf("error unmarshalling JSON for system state (pre-fetch): %w\nRaw: %s", err, string(body))
		}
		if rpcSystemStateResp.Error != nil {
			return nil, 0, fmt.Errorf("JSON-RPC error for system state (pre-fetch) - Code: %d, Message: %s", rpcSystemStateResp.Error.Code, rpcSystemStateResp.Error.Message)
		}
		if rpcSystemStateResp.Result.Epoch == "" {
			return nil, 0, fmt.Errorf("epoch not found in pre-fetch system state response")
		}
		actualEpoch, err = strconv.ParseUint(rpcSystemStateResp.Result.Epoch, 10, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("error parsing epoch from pre-fetch system state response: %w", err)
		}
		epochToQueryStr = rpcSystemStateResp.Result.Epoch
		log.Printf("Determined latest epoch to be %d for querying committee info.", actualEpoch)
	} else {
		actualEpoch = targetEpoch
		epochToQueryStr = strconv.FormatUint(targetEpoch, 10)
		log.Printf("Using specified target epoch %d for querying committee info.", actualEpoch)
	}

	// Step 1: Call suix_getCommitteeInfo for the determined/specified epoch
	log.Printf("Fetching committee info for epoch %s using suix_getCommitteeInfo...", epochToQueryStr)
	committeeInfoRequestPayload := JSONRPCRequest{
		JSONRPC: "2.0", ID: 1, Method: "suix_getCommitteeInfo", Params: []interface{}{epochToQueryStr},
	}
	jsonDataCommittee, err := json.Marshal(committeeInfoRequestPayload)
	if err != nil {
		return nil, 0, fmt.Errorf("error marshalling JSON request for committee info: %w", err)
	}
	httpReqCommittee, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewBuffer(jsonDataCommittee))
	if err != nil {
		return nil, 0, fmt.Errorf("error creating HTTP request for committee info: %w", err)
	}
	httpReqCommittee.Header.Set("Content-Type", "application/json")
	httpRespCommittee, err := httpClient.Do(httpReqCommittee)
	if err != nil {
		return nil, 0, fmt.Errorf("error performing HTTP request for committee info: %w", err)
	}
	defer httpRespCommittee.Body.Close()
	bodyCommittee, err := io.ReadAll(httpRespCommittee.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("error reading HTTP response body for committee info: %w", err)
	}
	if httpRespCommittee.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("HTTP request for committee info failed with status %s: %s", httpRespCommittee.Status, string(bodyCommittee))
	}
	var rpcCommitteeInfoResp JSONRPCResponseCommitteeInfo
	if err := json.Unmarshal(bodyCommittee, &rpcCommitteeInfoResp); err != nil {
		return nil, 0, fmt.Errorf("error unmarshalling JSON response for committee info: %w\nRaw body: %s", err, string(bodyCommittee))
	}
	if rpcCommitteeInfoResp.Error != nil {
		return nil, 0, fmt.Errorf("JSON-RPC error for committee info - Code: %d, Message: %s", rpcCommitteeInfoResp.Error.Code, rpcCommitteeInfoResp.Error.Message)
	}
	if rpcCommitteeInfoResp.Result.Epoch == "" { // Also check if Validators field is nil or empty if necessary
		return nil, 0, fmt.Errorf("epoch or validators not found in committee info response for epoch %s", epochToQueryStr)
	}

	// Store committee pubkeys by their bitmap index
	committeePubKeysOrdered := make([]string, len(rpcCommitteeInfoResp.Result.Validators))
	for i, valData := range rpcCommitteeInfoResp.Result.Validators {
		if len(valData) > 0 {
			committeePubKeysOrdered[i] = valData[0] // valData[0] is protocolPubkeyBytes
		} else {
			return nil, 0, fmt.Errorf("malformed validator data in committee info for epoch %s, index %d", epochToQueryStr, i)
		}
	}
	log.Printf("Successfully fetched %d validator public keys (committee order) for epoch %s.", len(committeePubKeysOrdered), epochToQueryStr)

	// Step 2: Call suix_getLatestSuiSystemState to get validator metadata
	// (This might seem redundant if we already called it for epoch determination, but ensures we have data for the *exact* committee epoch)
	log.Printf("Fetching full Sui system state for epoch %s to get validator metadata...", epochToQueryStr) // epochToQueryStr comes from committeeInfo.Result.Epoch or targetEpoch
	systemStateRequestPayload := JSONRPCRequest{
		JSONRPC: "2.0", ID: 2, // Use a different ID if within the same function scope of other calls
		Method: "suix_getLatestSuiSystemState", Params: []interface{}{}, // suix_getLatestSuiSystemState does not take an epoch parameter
	}
	jsonDataSystemState, err := json.Marshal(systemStateRequestPayload)
	if err != nil {
		return nil, 0, fmt.Errorf("error marshalling JSON request for system state metadata: %w", err)
	}
	httpReqSystemState, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewBuffer(jsonDataSystemState))
	if err != nil {
		return nil, 0, fmt.Errorf("error creating HTTP request for system state metadata: %w", err)
	}
	httpReqSystemState.Header.Set("Content-Type", "application/json")
	httpRespSystemState, err := httpClient.Do(httpReqSystemState)
	if err != nil {
		return nil, 0, fmt.Errorf("error performing HTTP request for system state metadata: %w", err)
	}
	defer httpRespSystemState.Body.Close()
	bodySystemState, err := io.ReadAll(httpRespSystemState.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("error reading HTTP response body for system state metadata: %w", err)
	}
	if httpRespSystemState.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("HTTP request for system state metadata failed with status %s: %s", httpRespSystemState.Status, string(bodySystemState))
	}
	var rpcSystemStateResp JSONRPCResponseSuiSystemState
	if err := json.Unmarshal(bodySystemState, &rpcSystemStateResp); err != nil {
		return nil, 0, fmt.Errorf("error unmarshalling JSON response for system state metadata: %w\nRaw body: %s", err, string(bodySystemState))
	}
	if rpcSystemStateResp.Error != nil {
		return nil, 0, fmt.Errorf("JSON-RPC error for system state metadata - Code: %d, Message: %s", rpcSystemStateResp.Error.Code, rpcSystemStateResp.Error.Message)
	}

	// Verify that the epoch from system state matches the one we queried committee for
	// This is crucial because suix_getLatestSuiSystemState always returns the *latest*
	// and we need metadata for the *specific* epoch of the committee.
	// If they don't match, it means we're in an epoch transition or using stale data, which is problematic.
	// For now, we'll log a warning. A more robust solution might require suix_getSuiSystemState covering a specific epoch if available,
	// or acknowledging that metadata might be slightly off during epoch changes if we only use latest.
	// The user's current `validator_uptime.go` uses `suix_getLatestSuiSystemState` for committee.
	// The problem description says: "activeValidators vector you get from suix_getLatestSuiSystemState is just whatever order"
	// and "suix_getCommitteeInfo(epoch) ... is the canonical ... list for that epoch".
	// This implies we should trust committee info for the epoch, and system state is for metadata lookup.
	// If suix_getLatestSuiSystemState().epoch != suix_getCommitteeInfo(epoch).epoch, we have a potential mismatch.
	// For this implementation, we proceed assuming the system state is "recent enough" for metadata.
	// The critical part is the *order* from suix_getCommitteeInfo.

	latestSystemStateEpoch, err := strconv.ParseUint(rpcSystemStateResp.Result.Epoch, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing epoch from system state metadata response: %w", err)
	}
	if latestSystemStateEpoch != actualEpoch {
		log.Printf("Warning: Epoch from suix_getCommitteeInfo (%d) differs from suix_getLatestSuiSystemState epoch (%d). Metadata might be from a slightly different epoch.", actualEpoch, latestSystemStateEpoch)
		// We will proceed using `actualEpoch` (from committeeInfo or targetEpoch) as the source of truth for the committee structure.
	}

	// Step 3: Join committee order with metadata
	// Create a map of active validators by their protocol public key for quick lookup
	activeValidatorsMap := make(map[string]ActiveValidatorJSON)
	for _, valMeta := range rpcSystemStateResp.Result.ActiveValidators {
		if valMeta.ProtocolPubkeyBytes == "" {
			log.Printf("Warning: Active validator %s (SuiAddress: %s) is missing ProtocolPubkeyBytes in system state response. It will be skipped.", valMeta.Name, valMeta.SuiAddress)
			continue
		}
		activeValidatorsMap[valMeta.ProtocolPubkeyBytes] = valMeta
	}

	var committee []ValidatorInfo
	for bitmapIdx, pubKey := range committeePubKeysOrdered {
		meta, found := activeValidatorsMap[pubKey]
		if !found {
			// This validator was in committeeInfo but not in activeValidators of latest system state.
			// This could happen if a validator is in the committee for an epoch but is not "active"
			// or if there's a mismatch between committee epoch data and latest system state.
			log.Printf("Warning: Validator with ProtocolPubkeyBytes %s (BitmapIndex %d) from committee info (epoch %d) not found in activeValidators from latest system state (epoch %d). Using placeholder info.", pubKey, bitmapIdx, actualEpoch, latestSystemStateEpoch)
			committee = append(committee, ValidatorInfo{
				Name:                fmt.Sprintf("Unknown Validator (Pubkey: %s...)", shortPubKey(pubKey)),
				SuiAddress:          fmt.Sprintf("unknown-sui-address-for-%s", pubKey), // Placeholder, stats might not work well for this one
				ProtocolPubkeyBytes: pubKey,
				BitmapIndex:         bitmapIdx,
			})
			continue
		}
		committee = append(committee, ValidatorInfo{
			Name:                strings.TrimSpace(meta.Name),
			SuiAddress:          meta.SuiAddress,
			ProtocolPubkeyBytes: pubKey, // Use pubKey from committeeInfo for consistency
			BitmapIndex:         bitmapIdx,
		})
	}

	log.Printf("Successfully loaded and merged data for %d validators for epoch %d.", len(committee), actualEpoch)
	return committee, actualEpoch, nil
}

// Helper function to shorten pubkey for logging, if needed
func shortPubKey(pubKey string) string {
	if len(pubKey) > 10 {
		return pubKey[:10]
	}
	return pubKey
}

// subscribeToCheckpoints subscribes to the checkpoint stream and sends data to a channel.
// It attempts to automatically resubscribe if the stream is terminated.
func subscribeToCheckpoints(
	ctx context.Context,
	subClient subPb.SubscriptionServiceClient,
	checkpointChan chan<- *rpcPb.Checkpoint, // CheckpointData is from rpcPb
) {
	defer close(checkpointChan) // Close channel when subscription goroutine exits

	retryDelay := 1 * time.Second // Delay before retrying subscription

	for { // Outer loop for attempting to subscribe and resubscribe
		// Check if the context is already cancelled before trying to subscribe
		select {
		case <-ctx.Done():
			log.Printf("Context done, exiting subscription goroutine permanently (epoch: %v).", ctx.Err())
			return
		default:
			// Proceed to subscribe
		}

		log.Println("Attempting to subscribe to checkpoints...")
		stream, err := subClient.SubscribeCheckpoints(ctx, &subPb.SubscribeCheckpointsRequest{
			ReadMask: &fieldmaskpb.FieldMask{
				Paths: []string{"signature", "sequence_number", "epoch"}, // Request fields needed
			},
		})

		if err != nil {
			// If subscription attempt itself fails (e.g., network issue before stream established, or context cancelled during dial)
			log.Printf("Failed to subscribe to checkpoints: %v", err)
			if ctx.Err() != nil {
				log.Println("Context cancelled during subscription attempt. Exiting.")
				return
			}
			log.Printf("Will retry subscription in %v...", retryDelay)
			select {
			case <-time.After(retryDelay):
				// Potentially increase retryDelay here for backoff
				continue // Continue to the next iteration of the outer loop to resubscribe
			case <-ctx.Done():
				log.Printf("Context done while waiting to retry subscription: %v. Exiting.", ctx.Err())
				return
			}
		}

		log.Println("Successfully subscribed. Waiting for checkpoints...")

	recvLoop: // Label for the inner loop that receives messages from the current stream
		for {
			resp, err := stream.Recv()
			if err != nil {
				// Always check for parent context cancellation first.
				if ctx.Err() != nil {
					log.Printf("Context cancelled during Recv(): %v. Exiting subscription.", ctx.Err())
					return // Exit the entire function
				}

				if err == io.EOF {
					log.Println("Checkpoint stream ended (EOF). Attempting to resubscribe...")
					break recvLoop // Break from recvLoop to allow outer loop to resubscribe
				}

				s, ok := status.FromError(err)
				if ok {
					// Handle gRPC status codes
					if s.Code() == codes.Canceled {
						log.Printf("Subscription explicitly cancelled via gRPC context status (code: Canceled). Exiting: %v", err)
						return // Permanent exit, context associated with gRPC call was cancelled
					}
					// Specific RST_STREAM error or other Internal errors often indicate server-side stream termination
					if s.Code() == codes.Internal { // covers RST_STREAM
						log.Printf("Stream terminated with gRPC Internal error: %v. Attempting to resubscribe...", err)
						break recvLoop // Break from recvLoop to allow outer loop to resubscribe
					}
					// Other gRPC errors that might be transient
					log.Printf("Unhandled gRPC error receiving checkpoint: %v (Code: %s). Attempting to resubscribe...", err, s.Code())
					break recvLoop // Break from recvLoop to allow outer loop to resubscribe
				} else {
					// Non-gRPC error (e.g., raw network issues not wrapped by gRPC status)
					log.Printf("Non-gRPC error receiving checkpoint: %v. Attempting to resubscribe...", err)
					break recvLoop // Break from recvLoop to allow outer loop to resubscribe
				}
			} // end if err != nil

			// Successfully received a response
			if resp.Checkpoint != nil {
				select {
				case checkpointChan <- resp.Checkpoint:
					// Successfully sent to channel
				case <-ctx.Done():
					log.Printf("Context done while trying to send checkpoint to channel: %v. Exiting.", ctx.Err())
					return // Exit the entire function
				}
			}
		} // End of recvLoop

		// If we broke out of recvLoop, it means we need to resubscribe.
		// Wait before retrying the outer loop to avoid tight loops on persistent errors.
		log.Printf("Disconnected from stream. Waiting %v before attempting to resubscribe...", retryDelay)
		select {
		case <-time.After(retryDelay):
			// Potentially increase retryDelay for exponential backoff if desired
		case <-ctx.Done():
			log.Printf("Context done while waiting to resubscribe after disconnection: %v. Exiting.", ctx.Err())
			return // Exit the entire function
		}
	} // End of outer subscription loop
}

func main() {
	node := os.Getenv("SUI_NODE")
	if node == "" {
		node = "fullnode.mainnet.sui.io:443" // Default gRPC node for subscriptions
	}

	fmt.Printf("Connecting to Sui node for subscriptions: %s\n", node)

	// Shared context for managing shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("\nReceived signal: %s, shutting down gracefully...\n", sig)
		cancel()
	}()

	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	conn, err := grpc.DialContext(ctx, node, // This connection is for gRPC subscriptions
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC node %s: %v", node, err)
	}
	defer conn.Close()

	fmt.Println("Successfully connected to gRPC node for subscriptions.")

	subClient := subPb.NewSubscriptionServiceClient(conn) // Still needed for checkpoint subscriptions

	// Initial state
	var currentEpoch uint64
	var committee []ValidatorInfo
	validatorStats := make(map[string]ValidatorStats)
	var totalCheckpointsWithSig uint64

	// Initial committee load
	var loadedEpoch uint64
	// Load for epoch 0, meaning loadEpochValidatorData will determine the latest epoch itself.
	committee, loadedEpoch, err = loadEpochValidatorData(ctx, 0)
	if err != nil {
		log.Fatalf("Failed to load initial committee data: %v", err)
	}
	currentEpoch = loadedEpoch
	// Initialize stats for the initial committee
	for _, valInfo := range committee {
		validatorStats[valInfo.SuiAddress] = ValidatorStats{AttestedCount: 0, SignedCurrent: false}
	}

	checkpointChan := make(chan *rpcPb.Checkpoint)
	go subscribeToCheckpoints(ctx, subClient, checkpointChan)

	fmt.Println("Starting checkpoint processing loop...")

	for {
		select {
		case checkpointData, ok := <-checkpointChan:
			if !ok {
				log.Println("Checkpoint channel closed, exiting main loop.")
				return
			}

			if checkpointData.Signature == nil {
				// log.Printf("Checkpoint %d received without signature data.", checkpointData.SequenceNumber)
				continue // Skip checkpoints without signatures for uptime calculation
			}

			totalCheckpointsWithSig++
			checkpointEpoch := checkpointData.Signature.Epoch // Epoch from the signature block

			// Epoch change detection and committee reload
			if *checkpointEpoch > currentEpoch {
				fmt.Printf("\nEpoch changed from %d to %d. Reloading committee...\n", currentEpoch, checkpointEpoch)
				currentEpoch = *checkpointEpoch
				newCommittee, newLoadedEpoch, err := loadEpochValidatorData(ctx, *checkpointEpoch) // Pass specific epoch
				if err != nil {
					log.Printf("Failed to load committee for new epoch %d: %v. Continuing with old committee.", *checkpointEpoch, err)
					// Decide on error handling: potentially skip this checkpoint or retry loading?
					// For now, we'll log and might process with a stale committee if load fails.
				} else {
					committee = newCommittee
					currentEpoch = newLoadedEpoch // Ensure currentEpoch matches what was loaded
					// Update validatorStats: add new validators, existing attested counts are preserved
					for _, valInfo := range newCommittee {
						if _, exists := validatorStats[valInfo.SuiAddress]; !exists {
							validatorStats[valInfo.SuiAddress] = ValidatorStats{AttestedCount: 0, SignedCurrent: false}
						}
					}
					fmt.Printf("Successfully reloaded committee for epoch %d with %d validators.\n", currentEpoch, len(committee))
				}
			}

			// Reset SignedCurrent status for all validators in the current committee
			for _, valInfo := range committee {
				if stats, ok := validatorStats[valInfo.SuiAddress]; ok {
					stats.SignedCurrent = false
					validatorStats[valInfo.SuiAddress] = stats
				}
			}

			// Process bitmap for the current checkpoint
			bitmap := checkpointData.Signature.Bitmap
			committeeSize := len(committee)

			// Sanity check: Ensure all indices in the bitmap are within the bounds of the current committee size.
			// The committee is 0-indexed, so valid indices are 0 to len(committee)-1.
			for _, idxInBitmap := range bitmap {
				if idxInBitmap >= uint32(committeeSize) {
					log.Panicf(
						"CRITICAL: Bitmap index %d is out of bounds for committee size %d (epoch %d, checkpoint %d). Committee indices should be 0 to %d.",
						idxInBitmap,
						committeeSize,
						currentEpoch,
						checkpointData.SequenceNumber,
						committeeSize-1, // Corrected: max valid index is size-1
					)
				}
			}

			for _, valInfo := range committee { // valInfo.BitmapIndex is crucial here
				if isValidatorSigned(bitmap, valInfo.BitmapIndex) {
					if stats, ok := validatorStats[valInfo.SuiAddress]; ok {
						stats.SignedCurrent = true
						stats.AttestedCount++
						validatorStats[valInfo.SuiAddress] = stats
					} else {
						// This might happen if committee changed and stats weren't updated, though unlikely with current logic
						log.Printf("Warning: Validator %s (SuiAddress: %s) found in committee but not in stats map.", valInfo.Name, valInfo.SuiAddress)
					}
				}
			}

			// Print report
			fmt.Printf("\n--- Checkpoint #%d (Epoch: %d, Total w/Sig: %d) ---\n",
				checkpointData.SequenceNumber, currentEpoch, totalCheckpointsWithSig)

			// Sort committee by name for consistent display order
			// sort.Slice(committee, func(i, j int) bool {
			// 	return committee[i].Name < committee[j].Name
			// })

			// Sort committee by name for consistent display order before printing
			// Create a temporary slice to sort, as the main 'committee' slice is ordered by BitmapIndex
			displayCommittee := make([]ValidatorInfo, len(committee))
			copy(displayCommittee, committee)
			sort.Slice(displayCommittee, func(i, j int) bool {
				return displayCommittee[i].Name < displayCommittee[j].Name
			})

			var linesToPrint []string
			for _, valInfo := range displayCommittee { // Iterate sorted committee for display
				stats, ok := validatorStats[valInfo.SuiAddress]
				if !ok {
					log.Printf("Warning: Validator %s (SuiAddress: %s) in committee but missing from stats for reporting.", valInfo.Name, valInfo.SuiAddress)
					continue // Skip validators missing from stats
				}

				statusIcon := "❌"
				if stats.SignedCurrent {
					statusIcon = "✅"
				}

				uptimePercent := 0.0
				if totalCheckpointsWithSig > 0 {
					uptimePercent = (float64(stats.AttestedCount) / float64(totalCheckpointsWithSig)) * 100
				}
				// Prepare the string for this validator without a newline
				linesToPrint = append(linesToPrint, fmt.Sprintf("%s %-40s - Attested: %6.2f%% (%4d/%4d)",
					statusIcon, valInfo.Name, uptimePercent, stats.AttestedCount, totalCheckpointsWithSig))
			}

			// Now print the collected lines in two columns
			for j := 0; j < len(linesToPrint); j += 2 {
				// Print the first column item
				fmt.Print(linesToPrint[j])

				if j+1 < len(linesToPrint) {
					// If there's a second item for the row, print separator and the second item
					fmt.Printf("   |   %s\n", linesToPrint[j+1])
				} else {
					// If it's the last item (odd number), just add a newline
					fmt.Println()
				}
			}

		case <-ctx.Done():
			log.Println("Context done, exiting main processing loop.")
			return
		}
	}
}
