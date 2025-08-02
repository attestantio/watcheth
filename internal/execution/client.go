package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Client interface {
	GetNodeInfo(ctx context.Context) (*ExecutionNodeInfo, error)
	GetEndpoint() string
	GetName() string
}

type executionClient struct {
	endpoint   string
	name       string
	httpClient *http.Client
}

func NewClient(name, endpoint string) Client {
	return &executionClient{
		name:     name,
		endpoint: strings.TrimRight(endpoint, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *executionClient) GetEndpoint() string {
	return c.endpoint
}

func (c *executionClient) GetName() string {
	return c.name
}

func (c *executionClient) GetNodeInfo(ctx context.Context) (*ExecutionNodeInfo, error) {
	info := &ExecutionNodeInfo{
		Name:        c.name,
		Endpoint:    c.endpoint,
		IsConnected: false,
		LastUpdate:  time.Now(),
	}

	// Get sync status
	syncResp, err := c.callRPC(ctx, "eth_syncing", []interface{}{})
	if err != nil {
		info.LastError = fmt.Errorf("eth_syncing: %w", err)
		return info, err
	}

	// Parse sync status
	var syncData SyncingResponse
	if err := json.Unmarshal(syncResp, &syncData); err != nil {
		info.LastError = fmt.Errorf("parse sync response: %w", err)
		return info, err
	}

	info.IsConnected = true

	// Check if syncing
	switch v := syncData.Result.(type) {
	case bool:
		info.IsSyncing = v
	case map[string]interface{}:
		info.IsSyncing = true
		if starting, ok := v["startingBlock"].(string); ok {
			info.StartingBlock = parseHexUint64(starting)
		}
		if current, ok := v["currentBlock"].(string); ok {
			info.CurrentBlock = parseHexUint64(current)
		}
		if highest, ok := v["highestBlock"].(string); ok {
			info.HighestBlock = parseHexUint64(highest)
		}

		// Calculate sync progress
		if info.HighestBlock > info.StartingBlock {
			progress := float64(info.CurrentBlock-info.StartingBlock) / float64(info.HighestBlock-info.StartingBlock) * 100
			info.SyncProgress = progress
		}
	}

	// Get current block number if not syncing
	if !info.IsSyncing {
		blockResp, err := c.callRPC(ctx, "eth_blockNumber", []interface{}{})
		if err == nil {
			var blockNum BlockNumberResponse
			if err := json.Unmarshal(blockResp, &blockNum); err == nil {
				info.CurrentBlock = parseHexUint64(blockNum.Result)
				info.HighestBlock = info.CurrentBlock
			}
		}
	}

	// Get peer count
	peerResp, err := c.callRPC(ctx, "net_peerCount", []interface{}{})
	if err == nil {
		var peerCount PeerCountResponse
		if err := json.Unmarshal(peerResp, &peerCount); err == nil {
			info.PeerCount = parseHexUint64(peerCount.Result)
		}
	}

	// Get chain ID
	chainResp, err := c.callRPC(ctx, "eth_chainId", []interface{}{})
	if err == nil {
		var chainID ChainIDResponse
		if err := json.Unmarshal(chainResp, &chainID); err == nil {
			info.ChainID = parseHexBigInt(chainID.Result)
		}
	}

	// Get gas price
	gasResp, err := c.callRPC(ctx, "eth_gasPrice", []interface{}{})
	if err == nil {
		var gasPrice GasPriceResponse
		if err := json.Unmarshal(gasResp, &gasPrice); err == nil {
			info.GasPrice = parseHexBigInt(gasPrice.Result)
		}
	}

	// Get client version
	versionResp, err := c.callRPC(ctx, "web3_clientVersion", []interface{}{})
	if err == nil {
		var version ClientVersionResponse
		if err := json.Unmarshal(versionResp, &version); err == nil {
			info.NodeVersion = version.Result
		}
	}

	// Get network ID
	netResp, err := c.callRPC(ctx, "net_version", []interface{}{})
	if err == nil {
		var netVersion NetVersionResponse
		if err := json.Unmarshal(netResp, &netVersion); err == nil {
			info.NetworkID = netVersion.Result
		}
	}

	// Get latest block to calculate block time
	if info.CurrentBlock > 0 {
		blockResp, err := c.callRPC(ctx, "eth_getBlockByNumber", []interface{}{"latest", false})
		if err == nil {
			var block BlockResponse
			if err := json.Unmarshal(blockResp, &block); err == nil && block.Result != nil {
				timestamp := parseHexUint64(block.Result.Timestamp)
				info.LastBlockTime = time.Unix(int64(timestamp), 0)
				info.BlockTime = time.Since(info.LastBlockTime)
			}
		}
	}

	return info, nil
}

func (c *executionClient) callRPC(ctx context.Context, method string, params []interface{}) ([]byte, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func parseHexUint64(hex string) uint64 {
	if hex == "" || hex == "0x" {
		return 0
	}
	hex = strings.TrimPrefix(hex, "0x")
	val, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return 0
	}
	return val
}

func parseHexBigInt(hex string) *big.Int {
	if hex == "" || hex == "0x" {
		return big.NewInt(0)
	}
	hex = strings.TrimPrefix(hex, "0x")
	val, ok := new(big.Int).SetString(hex, 16)
	if !ok {
		return big.NewInt(0)
	}
	return val
}
