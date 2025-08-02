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
	Type     string `mapstructure:"type"` // "consensus" or "execution"
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

// GetType returns the client type, defaulting to "consensus" for backward compatibility
func (cc *ClientConfig) GetType() string {
	if cc.Type == "" {
		return "consensus"
	}
	return strings.ToLower(cc.Type)
}

// IsConsensus returns true if this is a consensus client
func (cc *ClientConfig) IsConsensus() bool {
	return cc.GetType() == "consensus"
}

// IsExecution returns true if this is an execution client
func (cc *ClientConfig) IsExecution() bool {
	return cc.GetType() == "execution"
}
