package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/watcheth/watcheth/internal/config"
	"github.com/watcheth/watcheth/internal/consensus"
	"github.com/watcheth/watcheth/internal/execution"
	"github.com/watcheth/watcheth/internal/logger"
	"github.com/watcheth/watcheth/internal/monitor"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Start monitoring clients",
	Long:  `Start the real-time monitoring display for configured consensus and execution clients.`,
	Run:   runMonitor,
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}

func runMonitor(cmd *cobra.Command, args []string) {
	// Initialize logger based on debug flag
	logger.SetDebugMode(IsDebugMode())

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

	mon := monitor.NewMonitorV2(cfg.GetRefreshInterval())

	// Add clients based on their type
	for _, clientCfg := range cfg.Clients {
		if clientCfg.IsConsensus() {
			client := consensus.NewConsensusClient(clientCfg.Name, clientCfg.Endpoint)
			mon.AddConsensusClient(client)
		} else if clientCfg.IsExecution() {
			client := execution.NewClient(clientCfg.Name, clientCfg.Endpoint)
			mon.AddExecutionClient(client)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	go mon.Start(ctx)

	display := monitor.NewDisplayV2(mon)
	display.SetupLogPaths(cfg.Clients)
	if err := display.Run(); err != nil {
		fmt.Printf("Error running display: %v\n", err)
		os.Exit(1)
	}
}
