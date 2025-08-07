package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/watcheth/watcheth/internal/config"
	"github.com/watcheth/watcheth/internal/consensus"
	"github.com/watcheth/watcheth/internal/execution"
	"github.com/watcheth/watcheth/internal/logger"
	"github.com/watcheth/watcheth/internal/validator/vouch"
)

var (
	verbose bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List client status once (non-interactive)",
	Long:  `List the current status of all configured consensus and execution clients without the interactive UI.`,
	Run:   runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose debug output")
}

func runList(cmd *cobra.Command, args []string) {
	// Initialize logger based on debug flag or verbose flag
	logger.SetDebugMode(IsDebugMode() || verbose)

	var cfg config.Config

	if err := viper.Unmarshal(&cfg); err != nil {
		if err := viper.ReadInConfig(); err == nil {
			if err := viper.Unmarshal(&cfg); err != nil {
				fmt.Printf("Error parsing config: %v\n", err)
				return
			}
		} else {
			fmt.Printf("Config file not found. Please create a watcheth.yaml file or specify one with --config\n")
			return
		}
	}

	if len(cfg.Clients) == 0 {
		fmt.Printf("No clients configured in config file. Please add at least one client to your watcheth.yaml\n")
		return
	}

	// Logger is already initialized based on flags

	// Separate clients by type
	var consensusClients []config.ClientConfig
	var executionClients []config.ClientConfig
	var validatorClients []config.ClientConfig

	for _, clientCfg := range cfg.Clients {
		if clientCfg.IsConsensus() {
			consensusClients = append(consensusClients, clientCfg)
		} else if clientCfg.IsExecution() {
			executionClients = append(executionClients, clientCfg)
		} else if clientCfg.IsValidator() {
			validatorClients = append(validatorClients, clientCfg)
		}
	}

	// Check consensus clients
	if len(consensusClients) > 0 {
		fmt.Printf("=== Consensus Clients (%d) ===\n\n", len(consensusClients))
		for _, clientCfg := range consensusClients {
			checkConsensusClient(clientCfg)
		}
	}

	// Check execution clients
	if len(executionClients) > 0 {
		fmt.Printf("=== Execution Clients (%d) ===\n\n", len(executionClients))
		for _, clientCfg := range executionClients {
			checkExecutionClient(clientCfg)
		}
	}

	// Check validator clients
	if len(validatorClients) > 0 {
		fmt.Printf("=== Validator Clients (%d) ===\n\n", len(validatorClients))
		for _, clientCfg := range validatorClients {
			checkValidatorClient(clientCfg)
		}
	}
}

func checkConsensusClient(clientCfg config.ClientConfig) {
	fmt.Printf("Checking %s at %s...\n", clientCfg.Name, clientCfg.Endpoint)
	client := consensus.NewConsensusClient(clientCfg.Name, clientCfg.Endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	info, err := client.GetNodeInfo(ctx)
	cancel()

	if err != nil {
		fmt.Printf("  ❌ Error: %v\n\n", err)
		return
	}

	if !info.IsConnected {
		fmt.Printf("  ❌ Not connected: %v\n\n", info.LastError)
		return
	}

	fmt.Printf("  ✅ Connected\n")
	if info.PeerCount > 0 {
		fmt.Printf("  Peer Count: %d\n", info.PeerCount)
	}
	if info.NodeVersion != "" {
		fmt.Printf("  Node Version: %s\n", info.NodeVersion)
	}
	if info.CurrentFork != "" {
		fmt.Printf("  Current Fork: %s\n", info.CurrentFork)
	}
	fmt.Printf("  Is Syncing: %v\n", info.IsSyncing)
	fmt.Printf("  Is Optimistic: %v\n", info.IsOptimistic)
	fmt.Printf("  EL Offline: %v\n", info.ElOffline)
	fmt.Printf("  Current Slot: %d\n", info.CurrentSlot)
	fmt.Printf("  Head Slot: %d\n", info.HeadSlot)
	fmt.Printf("  Sync Distance: %d\n", info.SyncDistance)
	fmt.Printf("  Current Epoch: %d\n", info.CurrentEpoch)
	fmt.Printf("  Finalized Epoch: %d\n", info.FinalizedEpoch)
	fmt.Printf("  Next Slot In: %s\n", formatDuration(info.TimeToNextSlot))
	fmt.Printf("  Next Epoch In: %s\n\n", formatDuration(info.TimeToNextEpoch))
}

func checkExecutionClient(clientCfg config.ClientConfig) {
	fmt.Printf("Checking %s at %s...\n", clientCfg.Name, clientCfg.Endpoint)
	client := execution.NewClient(clientCfg.Name, clientCfg.Endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	info, err := client.GetNodeInfo(ctx)
	cancel()

	if err != nil {
		fmt.Printf("  ❌ Error: %v\n\n", err)
		return
	}

	if !info.IsConnected {
		fmt.Printf("  ❌ Not connected: %v\n\n", info.LastError)
		return
	}

	status := "Synced"
	if info.IsSyncing {
		status = fmt.Sprintf("Syncing (%.1f%%)", info.SyncProgress)
	}

	fmt.Printf("  ✅ Status: %s\n", status)
	if info.PeerCount > 0 {
		fmt.Printf("  Peer Count: %d\n", info.PeerCount)
	}
	if info.NodeVersion != "" {
		fmt.Printf("  Node Version: %s\n", info.NodeVersion)
	}
	fmt.Printf("  Current Block: %d\n", info.CurrentBlock)
	if info.IsSyncing {
		fmt.Printf("  Highest Block: %d\n", info.HighestBlock)
		fmt.Printf("  Starting Block: %d\n", info.StartingBlock)
	}
	if info.ChainID != nil {
		fmt.Printf("  Chain ID: %s\n", info.ChainID.String())
	}
	if info.NetworkID != "" {
		fmt.Printf("  Network ID: %s\n", info.NetworkID)
	}
	if info.GasPrice != nil {
		gasPriceGwei := info.GasPrice.Int64() / 1e9
		fmt.Printf("  Gas Price: %d gwei\n", gasPriceGwei)
	}
	if info.BlockTime > 0 {
		fmt.Printf("  Time Since Last Block: %s\n", formatDuration(info.BlockTime))
	}
	fmt.Println()
}

func checkValidatorClient(clientCfg config.ClientConfig) {
	fmt.Printf("Checking %s at %s...\n", clientCfg.Name, clientCfg.Endpoint)

	// Special handling for different validator types
	if clientCfg.Type == "vouch" {
		client := vouch.NewVouchClient(clientCfg.Name, clientCfg.Endpoint)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		info, err := client.GetNodeInfo(ctx)
		cancel()

		if err != nil {
			fmt.Printf("  ❌ Error: %v\n\n", err)
			return
		}

		if !info.IsConnected {
			fmt.Printf("  ❌ Not connected: %v\n\n", info.LastError)
			return
		}

		fmt.Printf("  ✅ Connected\n")
		fmt.Printf("  Service Ready: %v\n", info.Ready)

		// Attestation performance
		fmt.Printf("\n  Attestation Performance:\n")
		if info.AttestationMarkSeconds > 0 {
			fmt.Printf("    Mark Time: %.2fs into slot\n", info.AttestationMarkSeconds)
		}
		if info.AttestationSuccessRate > 0 {
			fmt.Printf("    Success Rate: %.1f%%\n", info.AttestationSuccessRate)
		}

		// Block proposal performance
		if info.BlockProposalMarkSeconds > 0 || info.BlockProposalSuccessRate > 0 {
			fmt.Printf("\n  Block Proposal Performance:\n")
			if info.BlockProposalMarkSeconds > 0 {
				fmt.Printf("    Mark Time: %.2fs into slot\n", info.BlockProposalMarkSeconds)
			}
			if info.BlockProposalSuccessRate > 0 {
				fmt.Printf("    Success Rate: %.1f%%\n", info.BlockProposalSuccessRate)
			}
		}

		// Network health
		fmt.Printf("\n  Network Health:\n")
		if info.BeaconNodeResponseTime > 0 {
			fmt.Printf("    Beacon Node Response: %.0fms\n", info.BeaconNodeResponseTime)
		}

		// MEV/Builder metrics
		if info.BestBidRelayCount > 0 || info.BlocksFromRelay > 0 {
			fmt.Printf("\n  MEV/Builder:\n")
			if info.BestBidRelayCount > 0 {
				fmt.Printf("    Best Bid Relay Count: %d\n", info.BestBidRelayCount)
			}
			if info.BlocksFromRelay > 0 {
				fmt.Printf("    Blocks from Relay: %d\n", info.BlocksFromRelay)
			}
		}

		fmt.Println()
	}
}

func formatDuration(duration time.Duration) string {
	if duration < 0 {
		return "0s"
	}

	seconds := int(duration.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
