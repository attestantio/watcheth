package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/watcheth/watcheth/internal/logger"
)

type Client interface {
	GetNodeInfo(ctx context.Context) (*ConsensusNodeInfo, error)
	GetChainConfig(ctx context.Context) (*ChainConfig, error)
}

type ConsensusClient struct {
	endpoint   string
	httpClient *http.Client
	name       string
}

func NewConsensusClient(name, endpoint string) *ConsensusClient {
	return &ConsensusClient{
		name:     name,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Increased from 5s to 10s for better reliability
		},
	}
}

func (c *ConsensusClient) GetNodeInfo(ctx context.Context) (*ConsensusNodeInfo, error) {
	info := &ConsensusNodeInfo{
		Name:       c.name,
		Endpoint:   c.endpoint,
		LastUpdate: time.Now(),
	}

	chainConfig, err := c.GetChainConfig(ctx)
	if err != nil {
		info.IsConnected = false
		info.LastError = err
		logger.Error("[%s]: Failed to get chain config: %v", c.name, err)
		return info, nil
	}

	syncing, err := c.getSyncing(ctx)
	if err != nil {
		info.IsConnected = false
		info.LastError = err
		logger.Error("[%s]: Failed to get syncing status: %v", c.name, err)
		return info, nil
	}

	info.IsSyncing = syncing.Data.IsSyncing
	info.IsOptimistic = syncing.Data.IsOptimistic

	headSlot, _ := strconv.ParseUint(syncing.Data.HeadSlot, 10, 64)
	syncDistance, _ := strconv.ParseUint(syncing.Data.SyncDistance, 10, 64)
	info.HeadSlot = headSlot
	info.SyncDistance = syncDistance

	// Try to get headers, but don't fail if not available
	headers, err := c.getHeaders(ctx)
	if err == nil && len(headers.Data) > 0 {
		slot, _ := strconv.ParseUint(headers.Data[0].Header.Message.Slot, 10, 64)
		info.HeadSlot = slot
	}
	// If headers endpoint fails, head slot was already set from syncing response

	finality, err := c.getFinalityCheckpoints(ctx)
	if err != nil {
		info.IsConnected = false
		info.LastError = err
		logger.Error("[%s]: Failed to get finality checkpoints: %v", c.name, err)
		return info, nil
	}

	justifiedEpoch, _ := strconv.ParseUint(finality.Data.CurrentJustified.Epoch, 10, 64)
	finalizedEpoch, _ := strconv.ParseUint(finality.Data.Finalized.Epoch, 10, 64)
	info.JustifiedEpoch = justifiedEpoch
	info.FinalizedEpoch = finalizedEpoch

	// Safely calculate slot numbers with overflow protection
	if justifiedEpoch > 0 && justifiedEpoch <= (^uint64(0))/chainConfig.SlotsPerEpoch {
		info.JustifiedSlot = justifiedEpoch * chainConfig.SlotsPerEpoch
	}
	if finalizedEpoch > 0 && finalizedEpoch <= (^uint64(0))/chainConfig.SlotsPerEpoch {
		info.FinalizedSlot = finalizedEpoch * chainConfig.SlotsPerEpoch
	}

	currentTime := time.Now()
	timeSinceGenesis := currentTime.Sub(chainConfig.GenesisTime)

	// Only calculate current slot if time since genesis is positive
	if timeSinceGenesis > 0 {
		currentSlot := uint64(timeSinceGenesis.Seconds()) / chainConfig.SecondsPerSlot
		info.CurrentSlot = currentSlot
		info.CurrentEpoch = currentSlot / chainConfig.SlotsPerEpoch
	}

	// Only calculate timing information if we have valid slot data
	if timeSinceGenesis > 0 && info.CurrentSlot > 0 {
		slotDuration := time.Duration(chainConfig.SecondsPerSlot) * time.Second
		timeInCurrentSlot := time.Duration(uint64(timeSinceGenesis.Seconds())%chainConfig.SecondsPerSlot) * time.Second
		info.TimeToNextSlot = slotDuration - timeInCurrentSlot

		slotsInCurrentEpoch := info.CurrentSlot % chainConfig.SlotsPerEpoch
		slotsUntilNextEpoch := chainConfig.SlotsPerEpoch - slotsInCurrentEpoch
		info.TimeToNextEpoch = info.TimeToNextSlot + time.Duration((slotsUntilNextEpoch-1)*chainConfig.SecondsPerSlot)*time.Second
	}

	// Get peer count
	peerCount, err := c.getPeerCount(ctx)
	if err == nil {
		connected, _ := strconv.ParseUint(peerCount.Data.Connected, 10, 64)
		info.PeerCount = connected
	}

	// Get node version
	nodeVersion, err := c.getNodeVersion(ctx)
	if err == nil {
		info.NodeVersion = nodeVersion.Data.Version
	}

	// Get fork info
	fork, err := c.getFork(ctx)
	if err == nil {
		info.CurrentFork = fork.Data.CurrentVersion
	}

	info.IsConnected = true
	logger.Info("[%s]: Successfully connected and retrieved node info", c.name)
	return info, nil
}

func (c *ConsensusClient) GetChainConfig(ctx context.Context) (*ChainConfig, error) {
	genesis, err := c.getGenesis(ctx)
	if err != nil {
		return nil, err
	}

	spec, err := c.getSpec(ctx)
	if err != nil {
		return nil, err
	}

	genesisTime, err := strconv.ParseInt(genesis.Data.GenesisTime, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse genesis time: %w", err)
	}

	// Extract string values from interface{}
	secondsPerSlotStr, ok := spec.Data["SECONDS_PER_SLOT"].(string)
	if !ok {
		return nil, fmt.Errorf("SECONDS_PER_SLOT is not a string")
	}
	slotsPerEpochStr, ok := spec.Data["SLOTS_PER_EPOCH"].(string)
	if !ok {
		return nil, fmt.Errorf("SLOTS_PER_EPOCH is not a string")
	}

	secondsPerSlot, err := strconv.ParseUint(secondsPerSlotStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SECONDS_PER_SLOT: %w", err)
	}
	if secondsPerSlot == 0 {
		return nil, fmt.Errorf("SECONDS_PER_SLOT cannot be zero")
	}

	slotsPerEpoch, err := strconv.ParseUint(slotsPerEpochStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SLOTS_PER_EPOCH: %w", err)
	}
	if slotsPerEpoch == 0 {
		return nil, fmt.Errorf("SLOTS_PER_EPOCH cannot be zero")
	}

	return &ChainConfig{
		SecondsPerSlot: secondsPerSlot,
		SlotsPerEpoch:  slotsPerEpoch,
		GenesisTime:    time.Unix(genesisTime, 0),
	}, nil
}

func (c *ConsensusClient) get(ctx context.Context, path string, v any) error {
	url := fmt.Sprintf("%s%s", c.endpoint, path)
	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Add delay for retries (exponential backoff)
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1)) // 100ms, 200ms, 400ms
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Check if this is the last attempt
			if attempt == maxRetries-1 {
				return fmt.Errorf("failed to execute request after %d attempts: %w", maxRetries, err)
			}
			// Log and retry for network errors
			logger.Debug("Request failed (attempt %d/%d) for %s: %v", attempt+1, maxRetries, url, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Don't retry for client errors (4xx), but retry for server errors (5xx)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return fmt.Errorf("HTTP %d for %s", resp.StatusCode, path)
			}
			if attempt == maxRetries-1 {
				return fmt.Errorf("HTTP %d for %s after %d attempts", resp.StatusCode, path, maxRetries)
			}
			logger.Debug("Server error %d (attempt %d/%d) for %s", resp.StatusCode, attempt+1, maxRetries, url)
			continue
		}

		// Read the body for debugging
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			if attempt == maxRetries-1 {
				return fmt.Errorf("failed to read response body after %d attempts: %w", maxRetries, err)
			}
			continue
		}

		// Decode the response
		if err := json.Unmarshal(body, v); err != nil {
			// JSON parsing errors are not retryable
			logger.Error("Failed to decode response from %s: %v", url, err)
			logger.Error("Response body: %s", string(body))
			return fmt.Errorf("failed to decode response: %w", err)
		}

		return nil // Success
	}

	return fmt.Errorf("exhausted all retry attempts for %s", url)
}

func (c *ConsensusClient) getGenesis(ctx context.Context) (*GenesisResponse, error) {
	var resp GenesisResponse
	err := c.get(ctx, "/eth/v1/beacon/genesis", &resp)
	return &resp, err
}

func (c *ConsensusClient) getHeaders(ctx context.Context) (*HeadersResponse, error) {
	var resp HeadersResponse
	err := c.get(ctx, "/eth/v1/beacon/headers", &resp)
	return &resp, err
}

func (c *ConsensusClient) getFinalityCheckpoints(ctx context.Context) (*FinalityCheckpointsResponse, error) {
	var resp FinalityCheckpointsResponse
	err := c.get(ctx, "/eth/v1/beacon/states/head/finality_checkpoints", &resp)
	return &resp, err
}

func (c *ConsensusClient) getSpec(ctx context.Context) (*SpecResponse, error) {
	var resp SpecResponse
	err := c.get(ctx, "/eth/v1/config/spec", &resp)
	return &resp, err
}

func (c *ConsensusClient) getSyncing(ctx context.Context) (*SyncingResponse, error) {
	var resp SyncingResponse
	err := c.get(ctx, "/eth/v1/node/syncing", &resp)
	return &resp, err
}

func (c *ConsensusClient) getPeerCount(ctx context.Context) (*PeerCountResponse, error) {
	var resp PeerCountResponse
	err := c.get(ctx, "/eth/v1/node/peer_count", &resp)
	return &resp, err
}

func (c *ConsensusClient) getNodeVersion(ctx context.Context) (*NodeVersionResponse, error) {
	var resp NodeVersionResponse
	err := c.get(ctx, "/eth/v1/node/version", &resp)
	return &resp, err
}

func (c *ConsensusClient) getFork(ctx context.Context) (*ForkResponse, error) {
	var resp ForkResponse
	err := c.get(ctx, "/eth/v1/beacon/states/head/fork", &resp)
	return &resp, err
}
