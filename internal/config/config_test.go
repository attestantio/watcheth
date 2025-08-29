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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_GetRefreshInterval(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected time.Duration
	}{
		{
			name: "valid duration string",
			config: Config{
				RefreshInterval: "5s",
			},
			expected: 5 * time.Second,
		},
		{
			name: "valid duration with minutes",
			config: Config{
				RefreshInterval: "2m30s",
			},
			expected: 2*time.Minute + 30*time.Second,
		},
		{
			name: "invalid duration string returns default",
			config: Config{
				RefreshInterval: "invalid",
			},
			expected: 2 * time.Second,
		},
		{
			name: "empty duration string returns default",
			config: Config{
				RefreshInterval: "",
			},
			expected: 2 * time.Second,
		},
		{
			name: "milliseconds",
			config: Config{
				RefreshInterval: "500ms",
			},
			expected: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetRefreshInterval()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClientConfig_GetLogPath(t *testing.T) {
	tests := []struct {
		name     string
		client   ClientConfig
		expected string
	}{
		{
			name: "empty log path returns default",
			client: ClientConfig{
				Name:    "Lighthouse",
				LogPath: "",
			},
			expected: "/var/log/lighthouse/lighthouse.log",
		},
		{
			name: "custom log path with {name} placeholder",
			client: ClientConfig{
				Name:    "Prysm",
				LogPath: "/custom/logs/{name}/{name}-beacon.log",
			},
			expected: "/custom/logs/prysm/prysm-beacon.log",
		},
		{
			name: "custom log path without placeholder",
			client: ClientConfig{
				Name:    "Geth",
				LogPath: "/var/log/ethereum/geth.log",
			},
			expected: "/var/log/ethereum/geth.log",
		},
		{
			name: "name with mixed case gets lowercased",
			client: ClientConfig{
				Name:    "LightHouse",
				LogPath: "",
			},
			expected: "/var/log/lighthouse/lighthouse.log",
		},
		{
			name: "multiple {name} placeholders",
			client: ClientConfig{
				Name:    "Teku",
				LogPath: "/logs/{name}/{name}-{name}.log",
			},
			expected: "/logs/teku/teku-teku.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.GetLogPath()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClientConfig_GetType(t *testing.T) {
	tests := []struct {
		name     string
		client   ClientConfig
		expected string
	}{
		{
			name: "empty type returns consensus default",
			client: ClientConfig{
				Type: "",
			},
			expected: "consensus",
		},
		{
			name: "consensus type lowercase",
			client: ClientConfig{
				Type: "consensus",
			},
			expected: "consensus",
		},
		{
			name: "execution type lowercase",
			client: ClientConfig{
				Type: "execution",
			},
			expected: "execution",
		},
		{
			name: "mixed case gets lowercased",
			client: ClientConfig{
				Type: "Consensus",
			},
			expected: "consensus",
		},
		{
			name: "uppercase gets lowercased",
			client: ClientConfig{
				Type: "EXECUTION",
			},
			expected: "execution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.GetType()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClientConfig_IsConsensus(t *testing.T) {
	tests := []struct {
		name     string
		client   ClientConfig
		expected bool
	}{
		{
			name: "consensus type returns true",
			client: ClientConfig{
				Type: "consensus",
			},
			expected: true,
		},
		{
			name: "execution type returns false",
			client: ClientConfig{
				Type: "execution",
			},
			expected: false,
		},
		{
			name: "empty type defaults to consensus",
			client: ClientConfig{
				Type: "",
			},
			expected: true,
		},
		{
			name: "mixed case consensus",
			client: ClientConfig{
				Type: "Consensus",
			},
			expected: true,
		},
		{
			name: "unknown type returns false",
			client: ClientConfig{
				Type: "unknown",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsConsensus()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClientConfig_IsExecution(t *testing.T) {
	tests := []struct {
		name     string
		client   ClientConfig
		expected bool
	}{
		{
			name: "execution type returns true",
			client: ClientConfig{
				Type: "execution",
			},
			expected: true,
		},
		{
			name: "consensus type returns false",
			client: ClientConfig{
				Type: "consensus",
			},
			expected: false,
		},
		{
			name: "empty type defaults to consensus",
			client: ClientConfig{
				Type: "",
			},
			expected: false,
		},
		{
			name: "mixed case execution",
			client: ClientConfig{
				Type: "Execution",
			},
			expected: true,
		},
		{
			name: "unknown type returns false",
			client: ClientConfig{
				Type: "unknown",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsExecution()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigStruct(t *testing.T) {
	// Test full config struct
	config := Config{
		RefreshInterval: "10s",
		Clients: []ClientConfig{
			{
				Name:     "lighthouse",
				Type:     "consensus",
				Endpoint: "http://localhost:5052",
				LogPath:  "/var/log/{name}/{name}.log",
			},
			{
				Name:     "geth",
				Type:     "execution",
				Endpoint: "http://localhost:8545",
				LogPath:  "",
			},
		},
	}

	assert.Equal(t, 10*time.Second, config.GetRefreshInterval())
	assert.Len(t, config.Clients, 2)

	// Test first client (consensus)
	assert.Equal(t, "lighthouse", config.Clients[0].Name)
	assert.True(t, config.Clients[0].IsConsensus())
	assert.False(t, config.Clients[0].IsExecution())
	assert.Equal(t, "/var/log/lighthouse/lighthouse.log", config.Clients[0].GetLogPath())

	// Test second client (execution)
	assert.Equal(t, "geth", config.Clients[1].Name)
	assert.False(t, config.Clients[1].IsConsensus())
	assert.True(t, config.Clients[1].IsExecution())
	assert.Equal(t, "/var/log/geth/geth.log", config.Clients[1].GetLogPath())
}
