package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug [endpoint]",
	Short: "Debug beacon node endpoint",
	Long:  `Test various API endpoints on a beacon node to see what's available.`,
	Args:  cobra.ExactArgs(1),
	Run:   runDebug,
}

func init() {
	rootCmd.AddCommand(debugCmd)
}

func runDebug(cmd *cobra.Command, args []string) {
	endpoint := args[0]
	fmt.Printf("Testing beacon node at: %s\n\n", endpoint)

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
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint+path, nil)
		if err != nil {
			fmt.Printf(" ❌ Error creating request: %v\n", err)
			cancel()
			continue
		}

		resp, err := client.Do(req)
		cancel()
		
		if err != nil {
			fmt.Printf(" ❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf(" ✅ OK (200)\n")
		} else {
			fmt.Printf(" ❌ Status: %d\n", resp.StatusCode)
		}
	}
}