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

package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/watcheth/watcheth/internal/consensus"
	"github.com/watcheth/watcheth/internal/execution"
)

// Mock implementations for testing
type mockConsensusClient struct {
	name     string
	nodeInfo *consensus.ConsensusNodeInfo
	err      error
	delay    time.Duration
}

func (m *mockConsensusClient) GetNodeInfo(ctx context.Context) (*consensus.ConsensusNodeInfo, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return &consensus.ConsensusNodeInfo{Name: m.name, LastError: ctx.Err()}, ctx.Err()
		}
	}
	if m.err != nil {
		return &consensus.ConsensusNodeInfo{Name: m.name, LastError: m.err}, m.err
	}
	return m.nodeInfo, nil
}

func (m *mockConsensusClient) GetChainConfig(ctx context.Context) (*consensus.ChainConfig, error) {
	return &consensus.ChainConfig{}, nil
}

type mockExecutionClient struct {
	name     string
	endpoint string
	nodeInfo *execution.ExecutionNodeInfo
	err      error
	delay    time.Duration
}

func (m *mockExecutionClient) GetNodeInfo(ctx context.Context) (*execution.ExecutionNodeInfo, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return &execution.ExecutionNodeInfo{Name: m.name, LastError: ctx.Err()}, ctx.Err()
		}
	}
	if m.err != nil {
		return &execution.ExecutionNodeInfo{Name: m.name, LastError: m.err}, m.err
	}
	return m.nodeInfo, nil
}

func (m *mockExecutionClient) GetEndpoint() string {
	return m.endpoint
}

func (m *mockExecutionClient) GetName() string {
	return m.name
}

func TestNewMonitor(t *testing.T) {
	refreshInterval := 5 * time.Second
	monitor := NewMonitor(refreshInterval)

	assert.NotNil(t, monitor)
	assert.Equal(t, refreshInterval, monitor.refreshInterval)
	assert.Empty(t, monitor.consensusClients)
	assert.Empty(t, monitor.executionClients)
	assert.Empty(t, monitor.consensusInfos)
	assert.Empty(t, monitor.executionInfos)
	assert.NotNil(t, monitor.updateChan)
}

func TestMonitor_AddClients(t *testing.T) {
	monitor := NewMonitor(time.Second)

	// Add consensus client
	consensusClient := &mockConsensusClient{name: "lighthouse"}
	monitor.AddConsensusClient(consensusClient)

	assert.Len(t, monitor.consensusClients, 1)
	assert.Len(t, monitor.consensusInfos, 1)

	// Add execution client
	executionClient := &mockExecutionClient{name: "geth", endpoint: "http://localhost:8545"}
	monitor.AddExecutionClient(executionClient)

	assert.Len(t, monitor.executionClients, 1)
	assert.Len(t, monitor.executionInfos, 1)

	// Add more clients
	monitor.AddConsensusClient(&mockConsensusClient{name: "prysm"})
	monitor.AddExecutionClient(&mockExecutionClient{name: "besu", endpoint: "http://localhost:8546"})

	assert.Len(t, monitor.consensusClients, 2)
	assert.Len(t, monitor.executionClients, 2)
}

func TestMonitor_GetNodeInfos(t *testing.T) {
	monitor := NewMonitor(time.Second)

	// Initially empty
	update := monitor.GetNodeInfos()
	assert.Empty(t, update.ConsensusInfos)
	assert.Empty(t, update.ExecutionInfos)

	// Add clients
	monitor.AddConsensusClient(&mockConsensusClient{
		name: "lighthouse",
		nodeInfo: &consensus.ConsensusNodeInfo{
			Name:        "lighthouse",
			IsConnected: true,
			CurrentSlot: 100,
			PeerCount:   50,
		},
	})

	monitor.AddExecutionClient(&mockExecutionClient{
		name:     "geth",
		endpoint: "http://localhost:8545",
		nodeInfo: &execution.ExecutionNodeInfo{
			Name:         "geth",
			IsConnected:  true,
			CurrentBlock: 1000,
			PeerCount:    25,
		},
	})

	// Update the monitor
	ctx := context.Background()
	monitor.updateAll(ctx)

	// Get updated infos
	update = monitor.GetNodeInfos()
	assert.Len(t, update.ConsensusInfos, 1)
	assert.Len(t, update.ExecutionInfos, 1)
	assert.Equal(t, "lighthouse", update.ConsensusInfos[0].Name)
	assert.Equal(t, "geth", update.ExecutionInfos[0].Name)
}

func TestMonitor_UpdateAll(t *testing.T) {
	monitor := NewMonitor(time.Second)

	// Add multiple clients with different behaviors
	monitor.AddConsensusClient(&mockConsensusClient{
		name: "lighthouse",
		nodeInfo: &consensus.ConsensusNodeInfo{
			Name:        "lighthouse",
			IsConnected: true,
			CurrentSlot: 100,
		},
	})

	monitor.AddConsensusClient(&mockConsensusClient{
		name:  "prysm",
		delay: 100 * time.Millisecond,
		nodeInfo: &consensus.ConsensusNodeInfo{
			Name:        "prysm",
			IsConnected: true,
			CurrentSlot: 101,
		},
	})

	monitor.AddExecutionClient(&mockExecutionClient{
		name:     "geth",
		endpoint: "http://localhost:8545",
		nodeInfo: &execution.ExecutionNodeInfo{
			Name:         "geth",
			IsConnected:  true,
			CurrentBlock: 1000,
		},
	})

	// Test update
	ctx := context.Background()
	monitor.updateAll(ctx)

	// Verify results
	update := monitor.GetNodeInfos()
	assert.Len(t, update.ConsensusInfos, 2)
	assert.Len(t, update.ExecutionInfos, 1)

	// Check that all clients were updated
	assert.Equal(t, "lighthouse", update.ConsensusInfos[0].Name)
	assert.Equal(t, uint64(100), update.ConsensusInfos[0].CurrentSlot)
	assert.Equal(t, "prysm", update.ConsensusInfos[1].Name)
	assert.Equal(t, uint64(101), update.ConsensusInfos[1].CurrentSlot)
	assert.Equal(t, "geth", update.ExecutionInfos[0].Name)
	assert.Equal(t, uint64(1000), update.ExecutionInfos[0].CurrentBlock)
}

func TestMonitor_UpdatesChannel(t *testing.T) {
	monitor := NewMonitor(100 * time.Millisecond)

	// Add a client
	monitor.AddConsensusClient(&mockConsensusClient{
		name: "lighthouse",
		nodeInfo: &consensus.ConsensusNodeInfo{
			Name:        "lighthouse",
			IsConnected: true,
		},
	})

	// Listen for updates
	updateChan := monitor.Updates()
	ctx := context.Background()

	// Trigger an update
	go monitor.updateAll(ctx)

	// Wait for update
	select {
	case update := <-updateChan:
		assert.Len(t, update.ConsensusInfos, 1)
		assert.Equal(t, "lighthouse", update.ConsensusInfos[0].Name)
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for update")
	}
}

func TestMonitor_Start(t *testing.T) {
	monitor := NewMonitor(50 * time.Millisecond)

	updateCount := 0
	var mu sync.Mutex

	// Add a client that counts updates
	monitor.AddConsensusClient(&mockConsensusClient{
		name: "lighthouse",
		nodeInfo: &consensus.ConsensusNodeInfo{
			Name:        "lighthouse",
			IsConnected: true,
			CurrentSlot: uint64(updateCount),
		},
	})

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start monitoring in a goroutine
	go monitor.Start(ctx)

	// Count updates
	go func() {
		for range monitor.Updates() {
			mu.Lock()
			updateCount++
			mu.Unlock()
		}
	}()

	// Let it run for a bit
	time.Sleep(200 * time.Millisecond)

	// Stop monitoring
	cancel()

	// Give it time to stop
	time.Sleep(50 * time.Millisecond)

	// Check that we got multiple updates
	mu.Lock()
	count := updateCount
	mu.Unlock()

	assert.GreaterOrEqual(t, count, 2, "Expected at least 2 updates")
}

func TestMonitor_ConcurrentAccess(t *testing.T) {
	monitor := NewMonitor(time.Second)

	// Add initial clients
	for i := 0; i < 5; i++ {
		monitor.AddConsensusClient(&mockConsensusClient{
			name: "consensus-" + string(rune('0'+i)),
			nodeInfo: &consensus.ConsensusNodeInfo{
				Name:        "consensus-" + string(rune('0'+i)),
				IsConnected: true,
			},
		})
		monitor.AddExecutionClient(&mockExecutionClient{
			name:     "execution-" + string(rune('0'+i)),
			endpoint: "http://localhost:854" + string(rune('0'+i)),
			nodeInfo: &execution.ExecutionNodeInfo{
				Name:        "execution-" + string(rune('0'+i)),
				IsConnected: true,
			},
		})
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent updates
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitor.updateAll(ctx)
		}()
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = monitor.GetNodeInfos()
			_ = monitor.GetConsensusInfos()
			_ = monitor.GetExecutionInfos()
		}()
	}

	// Concurrent adds
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			monitor.AddConsensusClient(&mockConsensusClient{
				name: "new-consensus-" + string(rune('0'+idx)),
			})
			monitor.AddExecutionClient(&mockExecutionClient{
				name: "new-execution-" + string(rune('0'+idx)),
			})
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify final state
	update := monitor.GetNodeInfos()
	assert.GreaterOrEqual(t, len(update.ConsensusInfos), 5)
	assert.GreaterOrEqual(t, len(update.ExecutionInfos), 5)
}

func TestMonitor_TimeoutHandling(t *testing.T) {
	monitor := NewMonitor(time.Second)

	// Add a client that times out
	monitor.AddConsensusClient(&mockConsensusClient{
		name:  "slow-client",
		delay: 10 * time.Second, // Longer than the 5s timeout
		nodeInfo: &consensus.ConsensusNodeInfo{
			Name: "slow-client",
		},
	})

	// Add a normal client
	monitor.AddConsensusClient(&mockConsensusClient{
		name: "fast-client",
		nodeInfo: &consensus.ConsensusNodeInfo{
			Name:        "fast-client",
			IsConnected: true,
		},
	})

	ctx := context.Background()

	// Measure update time
	start := time.Now()
	monitor.updateAll(ctx)
	duration := time.Since(start)

	// Should complete within timeout (5s) + some buffer
	assert.Less(t, duration, 6*time.Second)

	// Check results
	update := monitor.GetNodeInfos()
	assert.Len(t, update.ConsensusInfos, 2)

	// Slow client should have timeout error
	var slowClient *consensus.ConsensusNodeInfo
	for _, info := range update.ConsensusInfos {
		if info.Name == "slow-client" {
			slowClient = info
			break
		}
	}
	assert.NotNil(t, slowClient)
	assert.NotNil(t, slowClient.LastError)
}

func TestMonitor_GetRefreshInterval(t *testing.T) {
	interval := 3 * time.Second
	monitor := NewMonitor(interval)

	assert.Equal(t, interval, monitor.GetRefreshInterval())
}

func TestMonitor_BackwardCompatibility(t *testing.T) {
	monitor := NewMonitor(time.Second)

	// Add clients
	monitor.AddConsensusClient(&mockConsensusClient{
		name: "lighthouse",
		nodeInfo: &consensus.ConsensusNodeInfo{
			Name:        "lighthouse",
			IsConnected: true,
		},
	})

	monitor.AddExecutionClient(&mockExecutionClient{
		name:     "geth",
		endpoint: "http://localhost:8545",
		nodeInfo: &execution.ExecutionNodeInfo{
			Name:        "geth",
			IsConnected: true,
		},
	})

	// Update
	ctx := context.Background()
	monitor.updateAll(ctx)

	// Test backward compatibility methods
	consensusInfos := monitor.GetConsensusInfos()
	executionInfos := monitor.GetExecutionInfos()

	assert.Len(t, consensusInfos, 1)
	assert.Len(t, executionInfos, 1)
	assert.Equal(t, "lighthouse", consensusInfos[0].Name)
	assert.Equal(t, "geth", executionInfos[0].Name)
}

func TestMonitor_UpdateChannelNonBlocking(t *testing.T) {
	monitor := NewMonitor(time.Second)

	// Add a client
	monitor.AddConsensusClient(&mockConsensusClient{
		name: "lighthouse",
		nodeInfo: &consensus.ConsensusNodeInfo{
			Name: "lighthouse",
		},
	})

	ctx := context.Background()

	// Fill the channel buffer (capacity is 1)
	monitor.updateAll(ctx)

	// This should not block even though channel is full
	done := make(chan bool)
	go func() {
		monitor.updateAll(ctx)
		done <- true
	}()

	select {
	case <-done:
		// Success - updateAll didn't block
	case <-time.After(100 * time.Millisecond):
		t.Fatal("updateAll blocked when channel was full")
	}
}
