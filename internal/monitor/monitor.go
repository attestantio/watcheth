package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/watcheth/watcheth/internal/beacon"
)

type Monitor struct {
	clients        []beacon.Client
	refreshInterval time.Duration
	nodeInfos      []*beacon.BeaconNodeInfo
	mu             sync.RWMutex
	updateChan     chan []*beacon.BeaconNodeInfo
}

func NewMonitor(refreshInterval time.Duration) *Monitor {
	return &Monitor{
		clients:         make([]beacon.Client, 0),
		refreshInterval: refreshInterval,
		nodeInfos:       make([]*beacon.BeaconNodeInfo, 0),
		updateChan:      make(chan []*beacon.BeaconNodeInfo, 1),
	}
}

func (m *Monitor) AddClient(client beacon.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients = append(m.clients, client)
	m.nodeInfos = append(m.nodeInfos, &beacon.BeaconNodeInfo{})
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
	results := make([]*beacon.BeaconNodeInfo, len(m.clients))

	for i, client := range m.clients {
		wg.Add(1)
		go func(idx int, c beacon.Client) {
			defer wg.Done()
			
			updateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			
			info, err := c.GetNodeInfo(updateCtx)
			if err != nil {
				// GetNodeInfo already returns a properly populated info even on error
				results[idx] = info
			} else {
				results[idx] = info
			}
		}(i, client)
	}

	wg.Wait()

	m.mu.Lock()
	m.nodeInfos = results
	m.mu.Unlock()

	select {
	case m.updateChan <- results:
	default:
	}
}

func (m *Monitor) GetNodeInfos() []*beacon.BeaconNodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	infos := make([]*beacon.BeaconNodeInfo, len(m.nodeInfos))
	copy(infos, m.nodeInfos)
	return infos
}

func (m *Monitor) Updates() <-chan []*beacon.BeaconNodeInfo {
	return m.updateChan
}