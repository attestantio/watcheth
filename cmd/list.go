package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/watcheth/watcheth/internal/beacon"
	"github.com/watcheth/watcheth/internal/config"
)

var (
	verbose bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List beacon node status once (non-interactive)",
	Long:  `List the current status of all configured beacon nodes without the interactive UI.`,
	Run:   runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose debug output")
}

func runList(cmd *cobra.Command, args []string) {
	var cfg config.Config
	
	if err := viper.Unmarshal(&cfg); err != nil {
		if err := viper.ReadInConfig(); err == nil {
			if err := viper.Unmarshal(&cfg); err != nil {
				fmt.Printf("Error parsing config: %v\n", err)
				return
			}
		} else {
			cfg = config.DefaultConfig
		}
	}

	if len(cfg.Clients) == 0 {
		fmt.Println("No clients configured. Using default configuration.")
		cfg = config.DefaultConfig
	}

	// Enable logging if verbose flag is set
	if !verbose {
		log.SetOutput(io.Discard)
	}

	fmt.Printf("Checking %d beacon nodes...\n\n", len(cfg.Clients))

	for _, clientCfg := range cfg.Clients {
		fmt.Printf("Checking %s at %s...\n", clientCfg.Name, clientCfg.Endpoint)
		client := beacon.NewBeaconClient(clientCfg.Name, clientCfg.Endpoint)
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		info, err := client.GetNodeInfo(ctx)
		cancel()

		if err != nil {
			fmt.Printf("  ❌ Error: %v\n\n", err)
			continue
		}

		if !info.IsConnected {
			fmt.Printf("  ❌ Not connected: %v\n\n", info.LastError)
			continue
		}

		status := "Synced"
		if info.IsSyncing {
			status = "Syncing"
		} else if info.IsOptimistic {
			status = "Optimistic"
		}

		fmt.Printf("  ✅ Status: %s\n", status)
		fmt.Printf("  Current Slot: %d\n", info.CurrentSlot)
		fmt.Printf("  Head Slot: %d\n", info.HeadSlot)
		fmt.Printf("  Sync Distance: %d\n", info.SyncDistance)
		fmt.Printf("  Current Epoch: %d\n", info.CurrentEpoch)
		fmt.Printf("  Finalized Epoch: %d\n", info.FinalizedEpoch)
		fmt.Printf("  Next Slot In: %s\n", formatDuration(info.TimeToNextSlot))
		fmt.Printf("  Next Epoch In: %s\n\n", formatDuration(info.TimeToNextEpoch))
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