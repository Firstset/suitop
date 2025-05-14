package rpc

import (
	"context"
	"fmt"
	"log"
	// valmodel "suitop/internal/validator" // No longer needed for SuiSystemStateResult
)

// GetLatestSuiSystemState fetches the latest Sui system state.
// It uses the generic Call method from the Client.
func (c *Client) GetLatestSuiSystemState(ctx context.Context) (*SuiSystemStateResult, error) {
	log.Println("Fetching latest Sui system state using suix_getLatestSuiSystemState...")

	var response JSONRPCResponseSuiSystemState // Uses the struct defined in client.go
	// suix_getLatestSuiSystemState does not take an epoch parameter, so params is empty or nil
	params := []interface{}{}

	err := c.Call(ctx, "suix_getLatestSuiSystemState", params, &response)
	if err != nil {
		return nil, fmt.Errorf("suix_getLatestSuiSystemState call failed: %w", err)
	}

	if response.Result.Epoch == "" { // Basic validation
		return nil, fmt.Errorf("epoch not found in latest system state response")
	}

	log.Printf("Successfully fetched latest system state for epoch %s, found %d active validators.", response.Result.Epoch, len(response.Result.ActiveValidators))
	return &response.Result, nil
}
