package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/watcheth/watcheth/internal/config"
	"github.com/watcheth/watcheth/internal/consensus"
	"github.com/watcheth/watcheth/internal/logger"
	"github.com/watcheth/watcheth/internal/monitor"
)

var monitorDebugCmd = &cobra.Command{
	Use:   "monitor-debug",
	Short: "Start monitoring consensus clients with debug output",
	Long:  `Start monitoring with debug output to see what's happening.`,
	Run:   runMonitorDebug,
}

func init() {
	rootCmd.AddCommand(monitorDebugCmd)
}

func runMonitorDebug(cmd *cobra.Command, args []string) {
	// Always enable debug logging for monitor-debug command
	logger.SetDebugMode(true)

	var cfg config.Config

	if err := viper.Unmarshal(&cfg); err != nil {
		if err := viper.ReadInConfig(); err == nil {
			if err := viper.Unmarshal(&cfg); err != nil {
				fmt.Printf("Error parsing config: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Config file not found. Please create a watcheth.yaml file or specify one with --config\n")
			os.Exit(1)
		}
	}

	if len(cfg.Clients) == 0 {
		fmt.Printf("No clients configured in config file. Please add at least one client to your watcheth.yaml\n")
		os.Exit(1)
	}

	fmt.Printf("Starting monitor with %d clients\n", len(cfg.Clients))
	for _, c := range cfg.Clients {
		fmt.Printf("  - %s at %s\n", c.Name, c.Endpoint)
	}

	mon := monitor.NewMonitor(cfg.GetRefreshInterval())

	for _, clientCfg := range cfg.Clients {
		client := consensus.NewConsensusClient(clientCfg.Name, clientCfg.Endpoint)
		mon.AddClient(client)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Debug: print updates
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case infos := <-mon.Updates():
				fmt.Printf("\n=== Update at %s ===\n", time.Now().Format("15:04:05"))
				for i, info := range infos {
					fmt.Printf("Node %d: Name=%s, Connected=%v, Slot=%d\n",
						i, info.Name, info.IsConnected, info.CurrentSlot)
				}
			}
		}
	}()

	go mon.Start(ctx)

	// Let it run for a bit
	time.Sleep(10 * time.Second)
	fmt.Println("\nStopping...")
}
