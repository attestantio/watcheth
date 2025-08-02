package config

import (
	"time"
)

type Config struct {
	Clients         []ClientConfig `mapstructure:"clients"`
	RefreshInterval string         `mapstructure:"refresh_interval"`
}

type ClientConfig struct {
	Name     string `mapstructure:"name"`
	Endpoint string `mapstructure:"endpoint"`
}

func (c *Config) GetRefreshInterval() time.Duration {
	duration, err := time.ParseDuration(c.RefreshInterval)
	if err != nil {
		return 2 * time.Second
	}
	return duration
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
