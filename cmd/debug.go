// Copyright © 2025 Attestant Limited.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/watcheth/watcheth/internal/logger"
)

var (
	clientType string
	outputFile string
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
	debugCmd.Flags().StringVarP(&clientType, "type", "t", "consensus", "Client type (consensus, execution, or vouch)")
	debugCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path to save debug results")
}

func runDebug(cmd *cobra.Command, args []string) {
	// Initialize logger based on debug flag
	logger.SetDebugMode(IsDebugMode())

	endpoint := args[0]

	// Create output writer
	var output io.Writer = os.Stdout
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		// Write to both stdout and file
		output = io.MultiWriter(os.Stdout, file)
	}

	if clientType == "execution" {
		debugExecutionClient(endpoint, output)
	} else if clientType == "vouch" {
		debugVouchClient(endpoint, output)
	} else {
		debugConsensusClient(endpoint, output)
	}
}

func debugConsensusClient(endpoint string, w io.Writer) {
	fmt.Fprintf(w, "Testing consensus client at: %s\n\n", endpoint)

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
		fmt.Fprintf(w, "Testing %s...", path)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint+path, nil)
		if err != nil {
			fmt.Fprintf(w, " ❌ Error creating request: %v\n", err)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(w, " ❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(w, " ❌ Error reading body: %v\n", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			fmt.Fprintf(w, " ✅ OK (200)\n")

			// Try to parse as JSON to show formatted response
			var rawJSON any
			if err := json.Unmarshal(body, &rawJSON); err != nil {
				fmt.Fprintf(w, "   Failed to parse JSON: %v\n", err)
				fmt.Fprintf(w, "   Raw response: %s\n", string(body))
			} else {
				formatted, _ := json.MarshalIndent(rawJSON, "   ", "  ")
				fmt.Fprintf(w, "   Response:\n%s\n", string(formatted))
			}
		} else {
			fmt.Fprintf(w, " ❌ Status: %d\n", resp.StatusCode)
			fmt.Fprintf(w, "   Response: %s\n", string(body))
		}
	}
}

func debugExecutionClient(endpoint string, w io.Writer) {
	fmt.Fprintf(w, "Testing execution client at: %s\n\n", endpoint)

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
		fmt.Fprintf(w, "Testing %s...", method)

		// Create JSON-RPC request
		jsonReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  method,
			"params":  []interface{}{},
			"id":      1,
		}

		reqBody, err := json.Marshal(jsonReq)
		if err != nil {
			fmt.Fprintf(w, " ❌ Error creating request: %v\n", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(reqBody)))
		if err != nil {
			fmt.Fprintf(w, " ❌ Error creating request: %v\n", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(w, " ❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(w, " ❌ Error reading body: %v\n", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			fmt.Fprintf(w, " ✅ OK (200)\n")

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Fprintf(w, "   Failed to parse JSON: %v\n", err)
			} else {
				if res, ok := result["result"]; ok {
					fmt.Fprintf(w, "   Result: %v\n", res)
				} else if errMsg, ok := result["error"]; ok {
					fmt.Fprintf(w, "   Error: %v\n", errMsg)
				}
			}
		} else {
			fmt.Fprintf(w, " ❌ Status: %d\n", resp.StatusCode)
			fmt.Fprintf(w, "   Response: %s\n", string(body))
		}
	}
}

func debugVouchClient(endpoint string, w io.Writer) {
	fmt.Fprintf(w, "Testing Vouch validator client at: %s\n\n", endpoint)

	// Determine the metrics URL - don't append /metrics if it's already in the endpoint
	metricsURL := endpoint
	if !strings.HasSuffix(endpoint, "/metrics") {
		metricsURL = endpoint + "/metrics"
	}

	// Test Prometheus metrics endpoint
	fmt.Fprintf(w, "Testing %s...", metricsURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", metricsURL, nil)
	if err != nil {
		fmt.Fprintf(w, " ❌ Error creating request: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(w, " ❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(w, " ❌ Error reading body: %v\n", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Fprintf(w, " ✅ OK (200)\n\n")

		// Just print the full raw response
		fmt.Fprintf(w, "=== Full Response ===\n")
		fmt.Fprintf(w, "%s\n", string(body))
	} else {
		fmt.Fprintf(w, " ❌ Status: %d\n", resp.StatusCode)
		fmt.Fprintf(w, "   Response: %s\n", string(body))
	}
}
