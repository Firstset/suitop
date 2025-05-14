package rpc

import (
	"context"
	"fmt"
	"log"
	"strconv"
	// valmodel "suitop/internal/validator" // No longer needed for CommitteeInfoResultJSON
)

// GetCommitteeInfo fetches committee information for a specific epoch.
// It uses the generic Call method from the Client.
func (c *Client) GetCommitteeInfo(ctx context.Context, epochStr string) (*CommitteeInfoResultJSON, error) {
	log.Printf("Fetching committee info for epoch %s using suix_getCommitteeInfo...", epochStr)

	var response JSONRPCResponseCommitteeInfo // Uses the struct defined in client.go (which now includes CommitteeInfoResultJSON)
	params := []interface{}{epochStr}

	err := c.Call(ctx, "suix_getCommitteeInfo", params, &response)
	if err != nil {
		return nil, fmt.Errorf("suix_getCommitteeInfo call failed: %w", err)
	}

	if response.Result.Epoch == "" { // Basic validation
		return nil, fmt.Errorf("epoch or validators not found in committee info response for epoch %s", epochStr)
	}

	// Validate committee epoch matches requested epoch
	responseEpoch, err := strconv.ParseUint(response.Result.Epoch, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing epoch from committee info response: %w", err)
	}
	queryEpoch, _ := strconv.ParseUint(epochStr, 10, 64) // Assume epochStr is valid if we reach here
	if responseEpoch != queryEpoch {
		log.Printf("Warning: Queried committee info for epoch %s but received data for epoch %d", epochStr, responseEpoch)
		// Depending on strictness, this could be an error. For now, a log warning.
	}

	log.Printf("Successfully fetched committee info for epoch %s, found %d validator entries.", response.Result.Epoch, len(response.Result.Validators))
	return &response.Result, nil
}
