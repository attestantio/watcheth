package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/watcheth/watcheth/internal/consensus"
	"github.com/watcheth/watcheth/internal/execution"
)

type NodeUpdate struct {
	ConsensusInfos []*consensus.ConsensusNodeInfo
	ExecutionInfos []*execution.ExecutionNodeInfo
}

type Monitor struct {
	consensusClients []consensus.Client
	executionClients []execution.Client
	refreshInterval  time.Duration

	consensusInfos []*consensus.ConsensusNodeInfo
	executionInfos []*execution.ExecutionNodeInfo

	mu         sync.RWMutex
	updateChan chan NodeUpdate
}

func NewMonitor(refreshInterval time.Duration) *Monitor {
	return &Monitor{
		consensusClients: make([]consensus.Client, 0),
		executionClients: make([]execution.Client, 0),
		refreshInterval:  refreshInterval,
		consensusInfos:   make([]*consensus.ConsensusNodeInfo, 0),
		executionInfos:   make([]*execution.ExecutionNodeInfo, 0),
		updateChan:       make(chan NodeUpdate, 1),
	}
}

func (m *Monitor) AddConsensusClient(client consensus.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.consensusClients = append(m.consensusClients, client)
	m.consensusInfos = append(m.consensusInfos, &consensus.ConsensusNodeInfo{})
}

func (m *Monitor) AddExecutionClient(client execution.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executionClients = append(m.executionClients, client)
	m.executionInfos = append(m.executionInfos, &execution.ExecutionNodeInfo{})
}

func (m *Monitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.refreshInterval)
	defer ticker.Stop()

	m.updateAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.updateAll(ctx)
		}
	}
}

func (m *Monitor) updateAll(ctx context.Context) {
	var wg sync.WaitGroup

	// Update consensus clients
	consensusResults := make([]*consensus.ConsensusNodeInfo, len(m.consensusClients))
	for i, client := range m.consensusClients {
		wg.Add(1)
		go func(idx int, c consensus.Client) {
			defer wg.Done()

			updateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			info, err := c.GetNodeInfo(updateCtx)
			if err != nil {
				// GetNodeInfo already returns a properly populated info even on error
				consensusResults[idx] = info
			} else {
				consensusResults[idx] = info
			}
		}(i, client)
	}

	// Update execution clients
	executionResults := make([]*execution.ExecutionNodeInfo, len(m.executionClients))
	for i, client := range m.executionClients {
		wg.Add(1)
		go func(idx int, c execution.Client) {
			defer wg.Done()

			updateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			info, err := c.GetNodeInfo(updateCtx)
			if err != nil {
				executionResults[idx] = info
			} else {
				executionResults[idx] = info
			}
		}(i, client)
	}

	wg.Wait()

	m.mu.Lock()
	m.consensusInfos = consensusResults
	m.executionInfos = executionResults
	m.mu.Unlock()

	update := NodeUpdate{
		ConsensusInfos: consensusResults,
		ExecutionInfos: executionResults,
	}

	select {
	case m.updateChan <- update:
	default:
	}
}

func (m *Monitor) GetNodeInfos() NodeUpdate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	consensusInfos := make([]*consensus.ConsensusNodeInfo, len(m.consensusInfos))
	copy(consensusInfos, m.consensusInfos)

	executionInfos := make([]*execution.ExecutionNodeInfo, len(m.executionInfos))
	copy(executionInfos, m.executionInfos)

	return NodeUpdate{
		ConsensusInfos: consensusInfos,
		ExecutionInfos: executionInfos,
	}
}

func (m *Monitor) Updates() <-chan NodeUpdate {
	return m.updateChan
}

func (m *Monitor) GetRefreshInterval() time.Duration {
	return m.refreshInterval
}

// Backward compatibility methods
func (m *Monitor) GetConsensusInfos() []*consensus.ConsensusNodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]*consensus.ConsensusNodeInfo, len(m.consensusInfos))
	copy(infos, m.consensusInfos)
	return infos
}

func (m *Monitor) GetExecutionInfos() []*execution.ExecutionNodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]*execution.ExecutionNodeInfo, len(m.executionInfos))
	copy(infos, m.executionInfos)
	return infos
}
