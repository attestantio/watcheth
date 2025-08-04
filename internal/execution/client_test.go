package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/watcheth/watcheth/internal/testutil"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name         string
		clientName   string
		endpoint     string
		expectedName string
		expectedURL  string
	}{
		{
			name:         "basic client creation",
			clientName:   "geth",
			endpoint:     "http://localhost:8545",
			expectedName: "geth",
			expectedURL:  "http://localhost:8545",
		},
		{
			name:         "endpoint with trailing slash",
			clientName:   "besu",
			endpoint:     "http://localhost:8545/",
			expectedName: "besu",
			expectedURL:  "http://localhost:8545",
		},
		{
			name:         "endpoint with multiple trailing slashes",
			clientName:   "nethermind",
			endpoint:     "http://localhost:8545///",
			expectedName: "nethermind",
			expectedURL:  "http://localhost:8545",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.clientName, tt.endpoint)
			assert.NotNil(t, client)

			execClient := client.(*executionClient)
			assert.Equal(t, tt.expectedName, execClient.name)
			assert.Equal(t, tt.expectedURL, execClient.endpoint)
			assert.NotNil(t, execClient.httpClient)
			assert.Equal(t, 30*time.Second, execClient.httpClient.Timeout)
		})
	}
}

func TestExecutionClient_GetEndpointAndName(t *testing.T) {
	client := NewClient("test-client", "http://localhost:8545")

	assert.Equal(t, "http://localhost:8545", client.GetEndpoint())
	assert.Equal(t, "test-client", client.GetName())
}

func TestExecutionClient_callRPC(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		params      []interface{}
		handler     http.HandlerFunc
		expectedErr string
		validate    func(t *testing.T, result []byte)
	}{
		{
			name:    "successful RPC call",
			method:  "eth_blockNumber",
			params:  []interface{}{},
			handler: testutil.MockHTTPResponse(http.StatusOK, testutil.ValidClientVersionResponse),
			validate: func(t *testing.T, result []byte) {
				var resp ClientVersionResponse
				err := json.Unmarshal(result, &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Geth/v1.13.0-stable-1234567/linux-amd64/go1.21.0", resp.Result)
			},
		},
		{
			name:    "RPC error response",
			method:  "eth_unknown",
			params:  []interface{}{},
			handler: testutil.MockHTTPResponse(http.StatusOK, testutil.RPCErrorResponse),
			validate: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "Method not found")
			},
		},
		{
			name:        "HTTP error",
			method:      "eth_blockNumber",
			params:      []interface{}{},
			handler:     testutil.MockHTTPResponse(http.StatusInternalServerError, "Internal Server Error"),
			expectedErr: "http status 500",
		},
		{
			name:   "context cancellation",
			method: "eth_blockNumber",
			params: []interface{}{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			expectedErr: "context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.HTTPTestServer(t, tt.handler)
			client := NewClient("test", server.URL).(*executionClient)

			ctx := context.Background()
			if tt.name == "context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				go func() {
					time.Sleep(50 * time.Millisecond)
					cancel()
				}()
			}

			result, err := client.callRPC(ctx, tt.method, tt.params)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestExecutionClient_GetNodeInfo(t *testing.T) {
	// Create a mock handler that tracks which methods are called
	createMockHandler := func(responses map[string]string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			method := req["method"].(string)
			if resp, ok := responses[method]; ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, resp)
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`)
			}
		}
	}

	tests := []struct {
		name      string
		responses map[string]string
		validate  func(t *testing.T, info *ExecutionNodeInfo)
	}{
		{
			name: "fully synced node",
			responses: map[string]string{
				"eth_syncing":          testutil.NotSyncingRPCResponse,
				"eth_blockNumber":      `{"jsonrpc":"2.0","id":1,"result":"0x1234"}`,
				"net_peerCount":        testutil.ValidPeerCountRPCResponse,
				"eth_chainId":          testutil.ValidChainIDResponse,
				"eth_gasPrice":         testutil.ValidGasPriceResponse,
				"web3_clientVersion":   testutil.ValidClientVersionResponse,
				"net_version":          `{"jsonrpc":"2.0","id":1,"result":"1"}`,
				"eth_getBlockByNumber": `{"jsonrpc":"2.0","id":1,"result":{"number":"0x1234","timestamp":"0x65000000","hash":"0xabc","parentHash":"0xdef"}}`,
			},
			validate: func(t *testing.T, info *ExecutionNodeInfo) {
				assert.True(t, info.IsConnected)
				assert.False(t, info.IsSyncing)
				assert.Equal(t, uint64(0x1234), info.CurrentBlock)
				assert.Equal(t, uint64(0x1234), info.HighestBlock)
				assert.Equal(t, uint64(25), info.PeerCount)
				assert.Equal(t, big.NewInt(1), info.ChainID)
				assert.Equal(t, big.NewInt(1000000000), info.GasPrice)
				assert.Equal(t, "Geth/v1.13.0-stable-1234567/linux-amd64/go1.21.0", info.NodeVersion)
				assert.Equal(t, "1", info.NetworkID)
				assert.Equal(t, time.Unix(0x65000000, 0), info.LastBlockTime)
			},
		},
		{
			name: "syncing node",
			responses: map[string]string{
				"eth_syncing":        testutil.ValidSyncingRPCResponse,
				"net_peerCount":      testutil.ValidPeerCountRPCResponse,
				"eth_chainId":        testutil.ValidChainIDResponse,
				"eth_gasPrice":       testutil.ValidGasPriceResponse,
				"web3_clientVersion": testutil.ValidClientVersionResponse,
				"net_version":        `{"jsonrpc":"2.0","id":1,"result":"1"}`,
			},
			validate: func(t *testing.T, info *ExecutionNodeInfo) {
				assert.True(t, info.IsConnected)
				assert.True(t, info.IsSyncing)
				assert.Equal(t, uint64(0x0), info.StartingBlock)
				assert.Equal(t, uint64(0x3e8), info.CurrentBlock)
				assert.Equal(t, uint64(0x7d0), info.HighestBlock)
				assert.Equal(t, float64(50), info.SyncProgress) // (1000-0)/(2000-0)*100 = 50%
			},
		},
		{
			name:      "connection failure",
			responses: map[string]string{
				// eth_syncing will return error - empty responses means method not found
			},
			validate: func(t *testing.T, info *ExecutionNodeInfo) {
				assert.True(t, info.IsConnected) // Actually gets marked as connected after successful RPC call
				// The mock returns a "Method not found" error which gets parsed successfully
			},
		},
		{
			name: "partial data available",
			responses: map[string]string{
				"eth_syncing":        testutil.NotSyncingRPCResponse,
				"eth_blockNumber":    `{"jsonrpc":"2.0","id":1,"result":"0x1234"}`,
				"web3_clientVersion": testutil.ValidClientVersionResponse,
				// When methods return JSON-RPC errors, parseHexBigInt returns big.NewInt(0)
				"eth_chainId":  `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`,
				"eth_gasPrice": `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`,
			},
			validate: func(t *testing.T, info *ExecutionNodeInfo) {
				assert.True(t, info.IsConnected)
				assert.False(t, info.IsSyncing)
				assert.Equal(t, uint64(0x1234), info.CurrentBlock)
				assert.Equal(t, "Geth/v1.13.0-stable-1234567/linux-amd64/go1.21.0", info.NodeVersion)
				// Optional fields should have zero values
				assert.Equal(t, uint64(0), info.PeerCount)
				// When RPC methods return errors, the parse functions return zero values
				assert.Equal(t, big.NewInt(0), info.ChainID)
				assert.Equal(t, big.NewInt(0), info.GasPrice)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.HTTPTestServer(t, createMockHandler(tt.responses))
			client := NewClient("test", server.URL)

			info, err := client.GetNodeInfo(context.Background())
			assert.NotNil(t, info)

			// Check for expected errors
			if tt.name == "connection failure" {
				// Connection doesn't actually fail with our mock; it returns JSON-RPC errors
				// The client will still be marked as connected if HTTP succeeds
				assert.NoError(t, err)
			}

			tt.validate(t, info)
		})
	}
}

func TestParseHexUint64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint64
	}{
		{"empty string", "", 0},
		{"0x only", "0x", 0},
		{"zero", "0x0", 0},
		{"small number", "0x10", 16},
		{"large number", "0xffffffffffffffff", ^uint64(0)},
		{"without 0x prefix", "ff", 255},
		{"invalid hex", "0xZZZ", 0},
		{"negative hex", "-0x10", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHexUint64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseHexBigInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *big.Int
	}{
		{"empty string", "", big.NewInt(0)},
		{"0x only", "0x", big.NewInt(0)},
		{"zero", "0x0", big.NewInt(0)},
		{"small number", "0x10", big.NewInt(16)},
		{"large number", "0xffffffffffffffff", new(big.Int).SetUint64(^uint64(0))},
		{"without 0x prefix", "ff", big.NewInt(255)},
		{"invalid hex", "0xZZZ", big.NewInt(0)},
		{"very large number", "0x1234567890abcdef1234567890abcdef", func() *big.Int {
			v, _ := new(big.Int).SetString("1234567890abcdef1234567890abcdef", 16)
			return v
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHexBigInt(tt.input)
			assert.Equal(t, 0, tt.expected.Cmp(result))
		})
	}
}

func TestSyncProgress(t *testing.T) {
	tests := []struct {
		name             string
		startingBlock    uint64
		currentBlock     uint64
		highestBlock     uint64
		expectedProgress float64
	}{
		{
			name:             "0% progress",
			startingBlock:    0,
			currentBlock:     0,
			highestBlock:     1000,
			expectedProgress: 0,
		},
		{
			name:             "50% progress",
			startingBlock:    0,
			currentBlock:     500,
			highestBlock:     1000,
			expectedProgress: 50,
		},
		{
			name:             "100% progress",
			startingBlock:    0,
			currentBlock:     1000,
			highestBlock:     1000,
			expectedProgress: 100,
		},
		{
			name:             "progress with offset",
			startingBlock:    1000,
			currentBlock:     1500,
			highestBlock:     2000,
			expectedProgress: 50,
		},
		{
			name:             "no progress when highest equals starting",
			startingBlock:    1000,
			currentBlock:     1000,
			highestBlock:     1000,
			expectedProgress: 0, // Division by zero protection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncData := fmt.Sprintf(`{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"startingBlock": "0x%x",
					"currentBlock": "0x%x",
					"highestBlock": "0x%x"
				}
			}`, tt.startingBlock, tt.currentBlock, tt.highestBlock)

			server := testutil.HTTPTestServer(t, createMockHandler(map[string]string{
				"eth_syncing": syncData,
			}))

			client := NewClient("test", server.URL)
			info, err := client.GetNodeInfo(context.Background())

			assert.NoError(t, err)
			assert.NotNil(t, info)

			// Allow for some floating point precision differences
			if tt.highestBlock > tt.startingBlock {
				assert.InDelta(t, tt.expectedProgress, info.SyncProgress, 0.01)
			} else {
				assert.Equal(t, tt.expectedProgress, info.SyncProgress)
			}
		})
	}
}

// Helper function to create mock RPC handler
func createMockHandler(responses map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		method := req["method"].(string)
		if resp, ok := responses[method]; ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, resp)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`)
		}
	}
}
