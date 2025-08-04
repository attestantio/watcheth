package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/watcheth/watcheth/internal/logger"
)

var (
	clientType string
)

var debugCmd = &cobra.Command{
	Use:   "debug [endpoint]",
	Short: "Debug client endpoint",
	Long:  `Test various API endpoints on a consensus or execution client to see what's available.`,
	Args:  cobra.ExactArgs(1),
	Run:   runDebug,
}

func init() {
	rootCmd.AddCommand(debugCmd)
	debugCmd.Flags().StringVarP(&clientType, "type", "t", "consensus", "Client type (consensus or execution)")
}

func runDebug(cmd *cobra.Command, args []string) {
	// Initialize logger based on debug flag
	logger.SetDebugMode(IsDebugMode())

	endpoint := args[0]

	if clientType == "execution" {
		debugExecutionClient(endpoint)
	} else {
		debugConsensusClient(endpoint)
	}
}

func debugConsensusClient(endpoint string) {
	fmt.Printf("Testing consensus client at: %s\n\n", endpoint)

	endpoints := []string{
		"/eth/v1/beacon/genesis",
		"/eth/v1/beacon/headers",
		"/eth/v1/beacon/states/head/finality_checkpoints",
		"/eth/v1/config/spec",
		"/eth/v1/node/syncing",
		"/eth/v1/node/version",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, path := range endpoints {
		fmt.Printf("Testing %s...", path)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint+path, nil)
		if err != nil {
			fmt.Printf(" ❌ Error creating request: %v\n", err)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf(" ❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf(" ✅ OK (200)\n")

			// Read and display response body for spec endpoint
			if path == "/eth/v1/config/spec" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("   Error reading body: %v\n", err)
				} else {
					fmt.Printf("   Response preview (first 500 chars):\n   %s\n", truncateString(string(body), 500))

					// Try to parse as JSON to show structure
					var rawJSON any
					if err := json.Unmarshal(body, &rawJSON); err != nil {
						fmt.Printf("   Failed to parse JSON: %v\n", err)
					} else {
						formatted, _ := json.MarshalIndent(rawJSON, "   ", "  ")
						fmt.Printf("   JSON structure:\n   %s\n", truncateString(string(formatted), 1000))
					}
				}
			}
		} else {
			fmt.Printf(" ❌ Status: %d\n", resp.StatusCode)
		}
	}
}

func debugExecutionClient(endpoint string) {
	fmt.Printf("Testing execution client at: %s\n\n", endpoint)

	// Test JSON-RPC methods
	methods := []string{
		"eth_syncing",
		"eth_blockNumber",
		"net_peerCount",
		"eth_chainId",
		"eth_gasPrice",
		"web3_clientVersion",
		"net_version",
		"eth_protocolVersion",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, method := range methods {
		fmt.Printf("Testing %s...", method)

		// Create JSON-RPC request
		jsonReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  method,
			"params":  []interface{}{},
			"id":      1,
		}

		jsonData, err := json.Marshal(jsonReq)
		if err != nil {
			fmt.Printf(" ❌ Error marshaling request: %v\n", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
		if err != nil {
			fmt.Printf(" ❌ Error creating request: %v\n", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf(" ❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf(" ❌ Error reading body: %v\n", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			fmt.Printf(" ✅ OK (200)\n")

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Printf("   Failed to parse JSON: %v\n", err)
			} else {
				if res, ok := result["result"]; ok {
					fmt.Printf("   Result: %v\n", res)
				} else if errMsg, ok := result["error"]; ok {
					fmt.Printf("   Error: %v\n", errMsg)
				}
			}
		} else {
			fmt.Printf(" ❌ Status: %d\n", resp.StatusCode)
			fmt.Printf("   Response: %s\n", string(body))
		}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}
