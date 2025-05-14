package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"suitop/internal/config" // For RPCClientConfig
	// valmodel "suitop/internal/validator" // No longer needed for these specific structs
)

// --- JSON Data Structs Moved from validator/model.go ---

// ActiveValidatorJSON is part of the suix_getLatestSuiSystemState response.
type ActiveValidatorJSON struct {
	SuiAddress          string `json:"suiAddress"`
	Name                string `json:"name"`
	ProtocolPubkeyBytes string `json:"protocolPubkeyBytes"`
}

// SuiSystemStateResult is the 'result' field of suix_getLatestSuiSystemState response.
type SuiSystemStateResult struct {
	Epoch            string                `json:"epoch"`
	ActiveValidators []ActiveValidatorJSON `json:"activeValidators"`
}

// CommitteeValidatorEntryJSON represents a single validator entry in suix_getCommitteeInfo response.
type CommitteeValidatorEntryJSON []string

// CommitteeInfoResultJSON is the 'result' field of suix_getCommitteeInfo response.
type CommitteeInfoResultJSON struct {
	Epoch      string                        `json:"epoch"`
	Validators []CommitteeValidatorEntryJSON `json:"validators"`
}

// --- End of Moved Structs ---

// JSONRPCRequest defines the structure for a JSON-RPC request.
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// JSONRPCError defines the structure for a JSON-RPC error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// BaseJSONRPCResponse is a generic response structure to check for errors before unmarshalling into specific result types.
type BaseJSONRPCResponse struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCResponseSuiSystemState wraps the system state result.
type JSONRPCResponseSuiSystemState struct {
	BaseJSONRPCResponse
	Result SuiSystemStateResult `json:"result,omitempty"`
}

// JSONRPCResponseCommitteeInfo wraps the committee info result.
type JSONRPCResponseCommitteeInfo struct {
	BaseJSONRPCResponse
	Result CommitteeInfoResultJSON `json:"result,omitempty"`
}

// Client manages making JSON-RPC calls.
type Client struct {
	httpClient *http.Client
	url        string
}

// NewClient creates a new RPC client.
func NewClient(cfg config.RPCClientConfig) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		url:        cfg.URL,
	}
}

// Call performs a JSON-RPC request and unmarshals the response.
// The `result` parameter should be a pointer to the specific expected result structure (e.g., *valmodel.SuiSystemStateResult).
// This generic method can be used by specific methods like GetCommitteeInfo or GetLatestSuiSystemState.
func (c *Client) Call(ctx context.Context, method string, params []interface{}, result interface{}) error {
	requestPayload := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1, // Simple ID, could be made more robust if needed
		Method:  method,
		Params:  params,
	}

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return fmt.Errorf("error marshalling JSON request for %s: %w", method, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating HTTP request for %s: %w", method, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error performing HTTP request for %s: %w", method, err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("error reading HTTP response body for %s: %w", method, err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request for %s failed with status %s: %s", method, httpResp.Status, string(body))
	}

	// First, unmarshal into a base response to check for RPC errors
	var baseResp BaseJSONRPCResponse
	if err := json.Unmarshal(body, &baseResp); err != nil {
		return fmt.Errorf("error unmarshalling base JSON response for %s: %w\nRaw: %s", method, err, string(body))
	}

	if baseResp.Error != nil {
		return fmt.Errorf("JSON-RPC error for %s - Code: %d, Message: %s", method, baseResp.Error.Code, baseResp.Error.Message)
	}

	// If no error, unmarshal the full response including the result field
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("error unmarshalling JSON result for %s: %w\nRaw: %s", method, err, string(body))
	}

	return nil
}
