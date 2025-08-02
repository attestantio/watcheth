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
	"github.com/watcheth/watcheth/internal/beacon"
	"github.com/watcheth/watcheth/internal/config"
	"github.com/watcheth/watcheth/internal/monitor"
)

var monitorDebugCmd = &cobra.Command{
	Use:   "monitor-debug",
	Short: "Start monitoring beacon nodes with debug output",
	Long:  `Start monitoring with debug output to see what's happening.`,
	Run:   runMonitorDebug,
}

func init() {
	rootCmd.AddCommand(monitorDebugCmd)
}

func runMonitorDebug(cmd *cobra.Command, args []string) {
	var cfg config.Config
	
	if err := viper.Unmarshal(&cfg); err != nil {
		if err := viper.ReadInConfig(); err == nil {
			if err := viper.Unmarshal(&cfg); err != nil {
				fmt.Printf("Error parsing config: %v\n", err)
				os.Exit(1)
			}
		} else {
			cfg = config.DefaultConfig
		}
	}

	if len(cfg.Clients) == 0 {
		fmt.Println("No clients configured. Using default configuration.")
		cfg = config.DefaultConfig
	}

	fmt.Printf("Starting monitor with %d clients\n", len(cfg.Clients))
	for _, c := range cfg.Clients {
		fmt.Printf("  - %s at %s\n", c.Name, c.Endpoint)
	}

	mon := monitor.NewMonitor(cfg.GetRefreshInterval())

	for _, clientCfg := range cfg.Clients {
		client := beacon.NewBeaconClient(clientCfg.Name, clientCfg.Endpoint)
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