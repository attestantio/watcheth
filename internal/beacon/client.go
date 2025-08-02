package beacon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Client interface {
	GetNodeInfo(ctx context.Context) (*BeaconNodeInfo, error)
	GetChainConfig(ctx context.Context) (*ChainConfig, error)
}

type BeaconClient struct {
	endpoint   string
	httpClient *http.Client
	name       string
}

func NewBeaconClient(name, endpoint string) *BeaconClient {
	return &BeaconClient{
		name:     name,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *BeaconClient) GetNodeInfo(ctx context.Context) (*BeaconNodeInfo, error) {
	info := &BeaconNodeInfo{
		Name:       c.name,
		Endpoint:   c.endpoint,
		LastUpdate: time.Now(),
	}

	chainConfig, err := c.GetChainConfig(ctx)
	if err != nil {
		info.IsConnected = false
		info.LastError = err
		return info, nil
	}

	syncing, err := c.getSyncing(ctx)
	if err != nil {
		info.IsConnected = false
		info.LastError = err
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
		return info, nil
	}

	justifiedEpoch, _ := strconv.ParseUint(finality.Data.CurrentJustified.Epoch, 10, 64)
	finalizedEpoch, _ := strconv.ParseUint(finality.Data.Finalized.Epoch, 10, 64)
	info.JustifiedEpoch = justifiedEpoch
	info.FinalizedEpoch = finalizedEpoch
	info.JustifiedSlot = justifiedEpoch * chainConfig.SlotsPerEpoch
	info.FinalizedSlot = finalizedEpoch * chainConfig.SlotsPerEpoch

	currentTime := time.Now()
	timeSinceGenesis := currentTime.Sub(chainConfig.GenesisTime)
	currentSlot := uint64(timeSinceGenesis.Seconds()) / chainConfig.SecondsPerSlot
	info.CurrentSlot = currentSlot
	info.CurrentEpoch = currentSlot / chainConfig.SlotsPerEpoch

	slotDuration := time.Duration(chainConfig.SecondsPerSlot) * time.Second
	timeInCurrentSlot := time.Duration(uint64(timeSinceGenesis.Seconds())%chainConfig.SecondsPerSlot) * time.Second
	info.TimeToNextSlot = slotDuration - timeInCurrentSlot

	slotsInCurrentEpoch := currentSlot % chainConfig.SlotsPerEpoch
	slotsUntilNextEpoch := chainConfig.SlotsPerEpoch - slotsInCurrentEpoch
	info.TimeToNextEpoch = info.TimeToNextSlot + time.Duration((slotsUntilNextEpoch-1)*chainConfig.SecondsPerSlot)*time.Second

	info.IsConnected = true
	return info, nil
}

func (c *BeaconClient) GetChainConfig(ctx context.Context) (*ChainConfig, error) {
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

	secondsPerSlot, _ := strconv.ParseUint(secondsPerSlotStr, 10, 64)
	slotsPerEpoch, _ := strconv.ParseUint(slotsPerEpochStr, 10, 64)

	return &ChainConfig{
		SecondsPerSlot: secondsPerSlot,
		SlotsPerEpoch:  slotsPerEpoch,
		GenesisTime:    time.Unix(genesisTime, 0),
	}, nil
}

func (c *BeaconClient) get(ctx context.Context, path string, v any) error {
	url := fmt.Sprintf("%s%s", c.endpoint, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, path)
	}

	// Read the body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Decode the response
	if err := json.Unmarshal(body, v); err != nil {
		// Only log errors when verbose logging is enabled
		log.Printf("ERROR: Failed to decode response from %s: %v", url, err)
		log.Printf("ERROR: Response body: %s", string(body))
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *BeaconClient) getGenesis(ctx context.Context) (*GenesisResponse, error) {
	var resp GenesisResponse
	err := c.get(ctx, "/eth/v1/beacon/genesis", &resp)
	return &resp, err
}

func (c *BeaconClient) getHeaders(ctx context.Context) (*HeadersResponse, error) {
	var resp HeadersResponse
	err := c.get(ctx, "/eth/v1/beacon/headers", &resp)
	return &resp, err
}

func (c *BeaconClient) getFinalityCheckpoints(ctx context.Context) (*FinalityCheckpointsResponse, error) {
	var resp FinalityCheckpointsResponse
	err := c.get(ctx, "/eth/v1/beacon/states/head/finality_checkpoints", &resp)
	return &resp, err
}

func (c *BeaconClient) getSpec(ctx context.Context) (*SpecResponse, error) {
	var resp SpecResponse
	err := c.get(ctx, "/eth/v1/config/spec", &resp)
	return &resp, err
}

func (c *BeaconClient) getSyncing(ctx context.Context) (*SyncingResponse, error) {
	var resp SyncingResponse
	err := c.get(ctx, "/eth/v1/node/syncing", &resp)
	return &resp, err
}
