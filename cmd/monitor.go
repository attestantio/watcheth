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
	"github.com/watcheth/watcheth/internal/validator/vouch"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Start unified client monitoring dashboard",
	Long: `Start the real-time monitoring dashboard for all configured Ethereum clients.
Provides a unified view of consensus, execution, and validator client metrics.`,
	Run: runMonitor,
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
			fmt.Printf("Config file not found. Please create a watcheth.yml file or specify one with --config\n")
			os.Exit(1)
		}
	}

	if len(cfg.Clients) == 0 {
		fmt.Printf("No clients configured in config file. Please add at least one client to your watcheth.yml\n")
		os.Exit(1)
	}

	mon := monitor.NewMonitor(cfg.GetRefreshInterval())

	// Add clients based on their type
	for _, clientCfg := range cfg.Clients {
		if clientCfg.IsConsensus() {
			client := consensus.NewConsensusClient(clientCfg.Name, clientCfg.Endpoint)
			mon.AddConsensusClient(client)
		} else if clientCfg.IsExecution() {
			client := execution.NewClient(clientCfg.Name, clientCfg.Endpoint)
			mon.AddExecutionClient(client)
		} else if clientCfg.IsValidator() {
			// Special handling for different validator types
			if clientCfg.Type == "vouch" {
				client := vouch.NewVouchClient(clientCfg.Name, clientCfg.Endpoint)
				mon.AddValidatorClient(client)
			}
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

	display := monitor.NewDisplay(mon)
	display.SetupLogPaths(cfg.Clients)
	if err := display.Run(); err != nil {
		fmt.Printf("Error running display: %v\n", err)
		os.Exit(1)
	}
}
