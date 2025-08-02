package config

import (
	"strings"
	"time"
)

type Config struct {
	Clients         []ClientConfig `mapstructure:"clients"`
	RefreshInterval string         `mapstructure:"refresh_interval"`
}

type ClientConfig struct {
	Name     string `mapstructure:"name"`
	Endpoint string `mapstructure:"endpoint"`
	LogPath  string `mapstructure:"log_path"`
}

func (c *Config) GetRefreshInterval() time.Duration {
	duration, err := time.ParseDuration(c.RefreshInterval)
	if err != nil {
		return 2 * time.Second
	}
	return duration
}

// GetLogPath returns the log path for the client, substituting {name} with the client name
func (cc *ClientConfig) GetLogPath() string {
	if cc.LogPath == "" {
		// Default log path pattern
		return "/var/log/" + strings.ToLower(cc.Name) + "/" + strings.ToLower(cc.Name) + ".log"
	}
	// Replace {name} placeholder with actual client name
	return strings.ReplaceAll(cc.LogPath, "{name}", strings.ToLower(cc.Name))
}

var DefaultConfig = Config{
	RefreshInterval: "2s",
	Clients: []ClientConfig{
		{
			Name:     "EthStaker",
			Endpoint: "https://beaconstate.ethstaker.cc",
		},
		{
			Name:     "Attestant",
			Endpoint: "https://mainnet-checkpoint-sync.attestant.io",
		},
		{
			Name:     "ChainSafe",
			Endpoint: "https://beaconstate-mainnet.chainsafe.io",
		},
		{
			Name:     "Sigma Prime",
			Endpoint: "https://mainnet.checkpoint.sigp.io",
		},
	},
}
