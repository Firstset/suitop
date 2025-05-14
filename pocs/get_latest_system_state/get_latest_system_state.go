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

// JSONRPCResponse defines the structure for a JSON-RPC response.
type JSONRPCResponse struct {
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

	fmt.Printf("Using JSON-RPC endpoint: %s\n", rpcURL)

	requestPayload := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "suix_getLatestSuiSystemState",
		Params:  []interface{}{},
	}

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		log.Fatalf("Error marshalling JSON request: %v", err)
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error creating HTTP request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		log.Fatalf("Error performing HTTP request: %v", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Fatalf("Error reading HTTP response body: %v", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		log.Fatalf("HTTP request failed with status %s: %s", httpResp.Status, string(body))
	}

	var rpcResponse JSONRPCResponse
	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		log.Fatalf("Error unmarshalling JSON response: %v\nRaw body: %s", err, string(body))
	}

	if rpcResponse.Error != nil {
		log.Fatalf("JSON-RPC Error - Code: %d, Message: %s", rpcResponse.Error.Code, rpcResponse.Error.Message)
	}

	if rpcResponse.Result == nil {
		log.Fatalf("JSON-RPC response does not contain a result. Raw body: %s", string(body))
	}

	prettyResult, err := json.MarshalIndent(rpcResponse.Result, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling result to pretty JSON: %v", err)
	}

	fmt.Println(string(prettyResult))
}
