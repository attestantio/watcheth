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
	Type     string `mapstructure:"type"` // "consensus", "execution", or "validator"
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

// IsValidator returns true if this is a validator client
func (cc *ClientConfig) IsValidator() bool {
	t := cc.GetType()
	return t == "validator" || t == "vouch"
}
