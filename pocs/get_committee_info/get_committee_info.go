package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

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

// LatestSuiSystemStateResult defines the structure for the result of suix_getLatestSuiSystemState.
// We only need the epoch for this POC.
type LatestSuiSystemStateResult struct {
	Epoch string `json:"epoch"`
}

// JSONRPCResponseForSystemState defines the structure for the JSON-RPC response of suix_getLatestSuiSystemState.
type JSONRPCResponseForSystemState struct {
	Jsonrpc string                     `json:"jsonrpc"`
	ID      int                        `json:"id"`
	Result  LatestSuiSystemStateResult `json:"result,omitempty"`
	Error   *JSONRPCError              `json:"error,omitempty"`
}

// JSONRPCResponseForCommitteeInfo defines the structure for the JSON-RPC response of suix_getCommitteeInfo.
// The result can be complex, so we'll use map[string]interface{} and pretty print it.
type JSONRPCResponseForCommitteeInfo struct {
	Jsonrpc string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   *JSONRPCError          `json:"error,omitempty"`
}

func main() {
	rpcURL := os.Getenv("SUI_JSON_RPC_URL")
	if rpcURL == "" {
		rpcURL = "https://fullnode.mainnet.sui.io"
	}
	fmt.Printf("Using JSON-RPC endpoint: %s\n\n", rpcURL)

	httpClient := &http.Client{Timeout: 15 * time.Second}

	// 1. Call suix_getLatestSuiSystemState to get the current epoch
	fmt.Println("Fetching latest Sui system state to get current epoch...")
	systemStateRequestPayload := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "suix_getLatestSuiSystemState",
		Params:  []interface{}{},
	}

	systemStateJsonData, err := json.Marshal(systemStateRequestPayload)
	if err != nil {
		log.Fatalf("Error marshalling JSON request for system state: %v", err)
	}

	ctxSystemState, cancelSystemState := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelSystemState()

	httpSystemStateReq, err := http.NewRequestWithContext(ctxSystemState, "POST", rpcURL, bytes.NewBuffer(systemStateJsonData))
	if err != nil {
		log.Fatalf("Error creating HTTP request for system state: %v", err)
	}
	httpSystemStateReq.Header.Set("Content-Type", "application/json")

	httpSystemStateResp, err := httpClient.Do(httpSystemStateReq)
	if err != nil {
		log.Fatalf("Error performing HTTP request for system state: %v", err)
	}
	defer httpSystemStateResp.Body.Close()

	systemStateBody, err := io.ReadAll(httpSystemStateResp.Body)
	if err != nil {
		log.Fatalf("Error reading HTTP response body for system state: %v", err)
	}

	if httpSystemStateResp.StatusCode != http.StatusOK {
		log.Fatalf("HTTP request for system state failed with status %s: %s", httpSystemStateResp.Status, string(systemStateBody))
	}

	var systemStateRpcResponse JSONRPCResponseForSystemState
	if err := json.Unmarshal(systemStateBody, &systemStateRpcResponse); err != nil {
		log.Fatalf("Error unmarshalling JSON response for system state: %v\nRaw body: %s", err, string(systemStateBody))
	}

	if systemStateRpcResponse.Error != nil {
		log.Fatalf("JSON-RPC Error for system state - Code: %d, Message: %s", systemStateRpcResponse.Error.Code, systemStateRpcResponse.Error.Message)
	}

	if systemStateRpcResponse.Result.Epoch == "" {
		log.Fatalf("Epoch not found in system state response. Raw body: %s", string(systemStateBody))
	}
	currentEpoch := systemStateRpcResponse.Result.Epoch
	fmt.Printf("Successfully fetched current epoch: %s\n\n", currentEpoch)

	// 2. Call suix_getCommitteeInfo with the obtained epoch
	fmt.Printf("Fetching committee info for epoch %s...\n", currentEpoch)
	committeeInfoRequestPayload := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2, // Using a different ID for the second request
		Method:  "suix_getCommitteeInfo",
		Params:  []interface{}{currentEpoch}, // Pass epoch as a string parameter
	}

	committeeInfoJsonData, err := json.Marshal(committeeInfoRequestPayload)
	if err != nil {
		log.Fatalf("Error marshalling JSON request for committee info: %v", err)
	}

	ctxCommitteeInfo, cancelCommitteeInfo := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelCommitteeInfo()

	httpCommitteeInfoReq, err := http.NewRequestWithContext(ctxCommitteeInfo, "POST", rpcURL, bytes.NewBuffer(committeeInfoJsonData))
	if err != nil {
		log.Fatalf("Error creating HTTP request for committee info: %v", err)
	}
	httpCommitteeInfoReq.Header.Set("Content-Type", "application/json")

	httpCommitteeInfoResp, err := httpClient.Do(httpCommitteeInfoReq)
	if err != nil {
		log.Fatalf("Error performing HTTP request for committee info: %v", err)
	}
	defer httpCommitteeInfoResp.Body.Close()

	committeeInfoBody, err := io.ReadAll(httpCommitteeInfoResp.Body)
	if err != nil {
		log.Fatalf("Error reading HTTP response body for committee info: %v", err)
	}

	if httpCommitteeInfoResp.StatusCode != http.StatusOK {
		log.Fatalf("HTTP request for committee info failed with status %s: %s", httpCommitteeInfoResp.Status, string(committeeInfoBody))
	}

	var committeeInfoRpcResponse JSONRPCResponseForCommitteeInfo
	if err := json.Unmarshal(committeeInfoBody, &committeeInfoRpcResponse); err != nil {
		log.Fatalf("Error unmarshalling JSON response for committee info: %v\nRaw body: %s", err, string(committeeInfoBody))
	}

	if committeeInfoRpcResponse.Error != nil {
		log.Fatalf("JSON-RPC Error for committee info - Code: %d, Message: %s", committeeInfoRpcResponse.Error.Code, committeeInfoRpcResponse.Error.Message)
	}

	if committeeInfoRpcResponse.Result == nil {
		log.Fatalf("JSON-RPC response for committee info does not contain a result. Raw body: %s", string(committeeInfoBody))
	}

	fmt.Println("\nCommittee Info Result:")
	prettyResult, err := json.MarshalIndent(committeeInfoRpcResponse.Result, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling committee info result to pretty JSON: %v", err)
	}

	fmt.Println(string(prettyResult))
}
