package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/watcheth/watcheth/internal/beacon"
	"github.com/watcheth/watcheth/internal/config"
	"github.com/watcheth/watcheth/internal/monitor"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Start monitoring beacon nodes",
	Long:  `Start the real-time monitoring display for configured beacon nodes.`,
	Run:   runMonitor,
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}

func runMonitor(cmd *cobra.Command, args []string) {
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

	go mon.Start(ctx)

	display := monitor.NewDisplay(mon)
	if err := display.Run(); err != nil {
		fmt.Printf("Error running display: %v\n", err)
		os.Exit(1)
	}
}