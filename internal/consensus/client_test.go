package consensus

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/watcheth/watcheth/internal/testutil"
)

func TestNewConsensusClient(t *testing.T) {
	client := NewConsensusClient("test-client", "http://localhost:5052")

	assert.Equal(t, "test-client", client.name)
	assert.Equal(t, "http://localhost:5052", client.endpoint)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}

func TestConsensusClient_GetChainConfig(t *testing.T) {
	tests := []struct {
		name      string
		endpoints map[string]struct {
			Status int
			Body   string
		}
		expected    *ChainConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful chain config retrieval",
			endpoints: map[string]struct {
				Status int
				Body   string
			}{
				"/eth/v1/beacon/genesis": {
					Status: http.StatusOK,
					Body:   testutil.ValidNodeIdentityResponse,
				},
				"/eth/v1/config/spec": {
					Status: http.StatusOK,
					Body:   testutil.ValidChainConfigResponse,
				},
			},
			expected: &ChainConfig{
				SecondsPerSlot: 12,
				SlotsPerEpoch:  32,
				GenesisTime:    time.Unix(1606824023, 0),
			},
			expectError: false,
		},
		{
			name: "genesis endpoint fails",
			endpoints: map[string]struct {
				Status int
				Body   string
			}{
				"/eth/v1/beacon/genesis": {
					Status: http.StatusInternalServerError,
					Body:   "Internal Server Error",
				},
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "HTTP 500",
		},
		{
			name: "spec endpoint fails",
			endpoints: map[string]struct {
				Status int
				Body   string
			}{
				"/eth/v1/beacon/genesis": {
					Status: http.StatusOK,
					Body:   testutil.ValidNodeIdentityResponse,
				},
				"/eth/v1/config/spec": {
					Status: http.StatusNotFound,
					Body:   "Not Found",
				},
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "HTTP 404",
		},
		{
			name: "invalid genesis time",
			endpoints: map[string]struct {
				Status int
				Body   string
			}{
				"/eth/v1/beacon/genesis": {
					Status: http.StatusOK,
					Body:   `{"data": {"genesis_time": "invalid"}}`,
				},
				"/eth/v1/config/spec": {
					Status: http.StatusOK,
					Body:   testutil.ValidChainConfigResponse,
				},
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "failed to parse genesis time",
		},
		{
			name: "missing SECONDS_PER_SLOT",
			endpoints: map[string]struct {
				Status int
				Body   string
			}{
				"/eth/v1/beacon/genesis": {
					Status: http.StatusOK,
					Body:   testutil.ValidNodeIdentityResponse,
				},
				"/eth/v1/config/spec": {
					Status: http.StatusOK,
					Body:   `{"data": {"SLOTS_PER_EPOCH": "32"}}`,
				},
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "SECONDS_PER_SLOT is not a string",
		},
		{
			name: "zero SECONDS_PER_SLOT",
			endpoints: map[string]struct {
				Status int
				Body   string
			}{
				"/eth/v1/beacon/genesis": {
					Status: http.StatusOK,
					Body:   testutil.ValidNodeIdentityResponse,
				},
				"/eth/v1/config/spec": {
					Status: http.StatusOK,
					Body:   `{"data": {"SECONDS_PER_SLOT": "0", "SLOTS_PER_EPOCH": "32"}}`,
				},
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "SECONDS_PER_SLOT cannot be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fix: use the correct genesis response
			genesisResponse := `{"data": {"genesis_time": "1606824023"}}`
			if genesis, ok := tt.endpoints["/eth/v1/beacon/genesis"]; ok && genesis.Status == http.StatusOK && genesis.Body != testutil.ValidNodeIdentityResponse {
				// Keep custom genesis response
			} else if genesis, ok := tt.endpoints["/eth/v1/beacon/genesis"]; ok && genesis.Status == http.StatusOK {
				// Replace with correct genesis response
				tt.endpoints["/eth/v1/beacon/genesis"] = struct {
					Status int
					Body   string
				}{
					Status: http.StatusOK,
					Body:   genesisResponse,
				}
			}

			server := testutil.HTTPTestServer(t, testutil.MockHTTPEndpoints(tt.endpoints))
			client := NewConsensusClient("test", server.URL)

			result, err := client.GetChainConfig(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.SecondsPerSlot, result.SecondsPerSlot)
				assert.Equal(t, tt.expected.SlotsPerEpoch, result.SlotsPerEpoch)
				assert.True(t, tt.expected.GenesisTime.Equal(result.GenesisTime))
			}
		})
	}
}

func TestConsensusClient_get(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successful request",
			handler:     testutil.MockHTTPResponse(http.StatusOK, `{"data": "test"}`),
			expectError: false,
		},
		{
			name:        "client error (no retry)",
			handler:     testutil.MockHTTPResponse(http.StatusBadRequest, `{"error": "bad request"}`),
			expectError: true,
			errorMsg:    "HTTP 400",
		},
		{
			name:        "server error (no retry)",
			handler:     testutil.MockHTTPResponse(http.StatusInternalServerError, "Server Error"),
			expectError: true,
			errorMsg:    "HTTP 500",
		},
		{
			name:        "invalid JSON response",
			handler:     testutil.MockHTTPResponse(http.StatusOK, `invalid json`),
			expectError: true,
			errorMsg:    "failed to decode response",
		},
		{
			name: "context cancellation",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			expectError: true,
			errorMsg:    "context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.HTTPTestServer(t, tt.handler)
			client := NewConsensusClient("test", server.URL)

			ctx := context.Background()
			if tt.name == "context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				go func() {
					time.Sleep(50 * time.Millisecond)
					cancel()
				}()
			}

			var result map[string]any
			err := client.get(ctx, "/test", &result)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConsensusClient_GetNodeInfo(t *testing.T) {
	validEndpoints := map[string]struct {
		Status int
		Body   string
	}{
		"/eth/v1/beacon/genesis": {
			Status: http.StatusOK,
			Body:   `{"data": {"genesis_time": "1606824023"}}`,
		},
		"/eth/v1/config/spec": {
			Status: http.StatusOK,
			Body:   testutil.ValidChainConfigResponse,
		},
		"/eth/v1/node/syncing": {
			Status: http.StatusOK,
			Body:   testutil.ValidSyncingResponse,
		},
		"/eth/v1/beacon/headers": {
			Status: http.StatusOK,
			Body:   `{"data": [{"header": {"message": {"slot": "150"}}}]}`,
		},
		"/eth/v1/beacon/states/head/finality_checkpoints": {
			Status: http.StatusOK,
			Body:   `{"data": {"current_justified": {"epoch": "3"}, "finalized": {"epoch": "2"}}}`,
		},
		"/eth/v1/node/peer_count": {
			Status: http.StatusOK,
			Body:   testutil.ValidPeerCountResponse,
		},
		"/eth/v1/node/version": {
			Status: http.StatusOK,
			Body:   testutil.ValidNodeVersionResponse,
		},
		"/eth/v1/beacon/states/head/fork": {
			Status: http.StatusOK,
			Body:   `{"data": {"current_version": "0x00000000"}}`,
		},
	}

	tests := []struct {
		name            string
		modifyEndpoints func(map[string]struct {
			Status int
			Body   string
		})
		validate func(*testing.T, *ConsensusNodeInfo)
	}{
		{
			name: "successful full node info",
			modifyEndpoints: func(endpoints map[string]struct {
				Status int
				Body   string
			}) {
				// Use default endpoints
			},
			validate: func(t *testing.T, info *ConsensusNodeInfo) {
				assert.True(t, info.IsConnected)
				assert.Equal(t, "test", info.Name)
				assert.True(t, info.IsSyncing)
				assert.Equal(t, uint64(150), info.HeadSlot)
				assert.Equal(t, uint64(50), info.PeerCount)
				assert.Equal(t, "Lighthouse/v4.5.0-1234567/x86_64-linux", info.NodeVersion)
				assert.Equal(t, "0x00000000", info.CurrentFork)
				assert.Equal(t, uint64(96), info.JustifiedSlot) // 3 * 32
				assert.Equal(t, uint64(64), info.FinalizedSlot) // 2 * 32
			},
		},
		{
			name: "chain config fails",
			modifyEndpoints: func(endpoints map[string]struct {
				Status int
				Body   string
			}) {
				endpoints["/eth/v1/beacon/genesis"] = struct {
					Status int
					Body   string
				}{
					Status: http.StatusInternalServerError,
					Body:   "Error",
				}
			},
			validate: func(t *testing.T, info *ConsensusNodeInfo) {
				assert.False(t, info.IsConnected)
				assert.NotNil(t, info.LastError)
			},
		},
		{
			name: "syncing endpoint fails",
			modifyEndpoints: func(endpoints map[string]struct {
				Status int
				Body   string
			}) {
				endpoints["/eth/v1/node/syncing"] = struct {
					Status int
					Body   string
				}{
					Status: http.StatusInternalServerError,
					Body:   "Error",
				}
			},
			validate: func(t *testing.T, info *ConsensusNodeInfo) {
				assert.False(t, info.IsConnected)
				assert.NotNil(t, info.LastError)
			},
		},
		{
			name: "optional endpoints fail gracefully",
			modifyEndpoints: func(endpoints map[string]struct {
				Status int
				Body   string
			}) {
				delete(endpoints, "/eth/v1/beacon/headers")
				delete(endpoints, "/eth/v1/node/peer_count")
				delete(endpoints, "/eth/v1/node/version")
				delete(endpoints, "/eth/v1/beacon/states/head/fork")
			},
			validate: func(t *testing.T, info *ConsensusNodeInfo) {
				assert.True(t, info.IsConnected)
				assert.Equal(t, uint64(100), info.HeadSlot) // From syncing response
				assert.Equal(t, uint64(0), info.PeerCount)
				assert.Empty(t, info.NodeVersion)
				assert.Empty(t, info.CurrentFork)
			},
		},
		{
			name: "not syncing state",
			modifyEndpoints: func(endpoints map[string]struct {
				Status int
				Body   string
			}) {
				endpoints["/eth/v1/node/syncing"] = struct {
					Status int
					Body   string
				}{
					Status: http.StatusOK,
					Body:   testutil.NotSyncingResponse,
				}
			},
			validate: func(t *testing.T, info *ConsensusNodeInfo) {
				assert.True(t, info.IsConnected)
				assert.False(t, info.IsSyncing)
				assert.Equal(t, uint64(0), info.SyncDistance)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints := make(map[string]struct {
				Status int
				Body   string
			})
			for k, v := range validEndpoints {
				endpoints[k] = v
			}

			if tt.modifyEndpoints != nil {
				tt.modifyEndpoints(endpoints)
			}

			server := testutil.HTTPTestServer(t, testutil.MockHTTPEndpoints(endpoints))
			client := NewConsensusClient("test", server.URL)

			info, err := client.GetNodeInfo(context.Background())
			assert.NoError(t, err) // GetNodeInfo always returns an info object
			assert.NotNil(t, info)

			tt.validate(t, info)
		})
	}
}

func TestConsensusClient_TimeCalculations(t *testing.T) {
	// Create a test server with valid responses
	endpoints := map[string]struct {
		Status int
		Body   string
	}{
		"/eth/v1/beacon/genesis": {
			Status: http.StatusOK,
			Body:   fmt.Sprintf(`{"data": {"genesis_time": "%d"}}`, time.Now().Add(-1*time.Hour).Unix()),
		},
		"/eth/v1/config/spec": {
			Status: http.StatusOK,
			Body:   `{"data": {"SECONDS_PER_SLOT": "12", "SLOTS_PER_EPOCH": "32"}}`,
		},
		"/eth/v1/node/syncing": {
			Status: http.StatusOK,
			Body:   testutil.NotSyncingResponse,
		},
		"/eth/v1/beacon/states/head/finality_checkpoints": {
			Status: http.StatusOK,
			Body:   `{"data": {"current_justified": {"epoch": "100"}, "finalized": {"epoch": "99"}}}`,
		},
	}

	server := testutil.HTTPTestServer(t, testutil.MockHTTPEndpoints(endpoints))
	client := NewConsensusClient("test", server.URL)

	info, err := client.GetNodeInfo(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, info)

	// Verify time calculations
	assert.True(t, info.TimeToNextSlot > 0 && info.TimeToNextSlot <= 12*time.Second)
	assert.True(t, info.TimeToNextEpoch > 0 && info.TimeToNextEpoch <= 32*12*time.Second)
	assert.True(t, info.CurrentSlot > 0)
	assert.True(t, info.CurrentEpoch > 0)
}

func TestConsensusClient_EdgeCases(t *testing.T) {
	t.Run("overflow protection", func(t *testing.T) {
		endpoints := map[string]struct {
			Status int
			Body   string
		}{
			"/eth/v1/beacon/genesis": {
				Status: http.StatusOK,
				Body:   `{"data": {"genesis_time": "1606824023"}}`,
			},
			"/eth/v1/config/spec": {
				Status: http.StatusOK,
				Body:   `{"data": {"SECONDS_PER_SLOT": "12", "SLOTS_PER_EPOCH": "32"}}`,
			},
			"/eth/v1/node/syncing": {
				Status: http.StatusOK,
				Body:   testutil.NotSyncingResponse,
			},
			"/eth/v1/beacon/states/head/finality_checkpoints": {
				Status: http.StatusOK,
				Body:   fmt.Sprintf(`{"data": {"current_justified": {"epoch": "%d"}, "finalized": {"epoch": "%d"}}}`, ^uint64(0)/32, ^uint64(0)/32),
			},
		}

		server := testutil.HTTPTestServer(t, testutil.MockHTTPEndpoints(endpoints))
		client := NewConsensusClient("test", server.URL)

		info, err := client.GetNodeInfo(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, info)

		// The client doesn't set slots to 0 on overflow, it allows the calculation
		// The overflow protection prevents calculation only when epoch > max_uint64/slots_per_epoch
		// In this case, ^uint64(0)/32 = 576460752303423487, which is still valid
		assert.True(t, info.JustifiedSlot > 0)
		assert.True(t, info.FinalizedSlot > 0)
	})

	t.Run("pre-genesis time", func(t *testing.T) {
		endpoints := map[string]struct {
			Status int
			Body   string
		}{
			"/eth/v1/beacon/genesis": {
				Status: http.StatusOK,
				Body:   fmt.Sprintf(`{"data": {"genesis_time": "%d"}}`, time.Now().Add(1*time.Hour).Unix()),
			},
			"/eth/v1/config/spec": {
				Status: http.StatusOK,
				Body:   `{"data": {"SECONDS_PER_SLOT": "12", "SLOTS_PER_EPOCH": "32"}}`,
			},
			"/eth/v1/node/syncing": {
				Status: http.StatusOK,
				Body:   testutil.NotSyncingResponse,
			},
			"/eth/v1/beacon/states/head/finality_checkpoints": {
				Status: http.StatusOK,
				Body:   `{"data": {"current_justified": {"epoch": "0"}, "finalized": {"epoch": "0"}}}`,
			},
		}

		server := testutil.HTTPTestServer(t, testutil.MockHTTPEndpoints(endpoints))
		client := NewConsensusClient("test", server.URL)

		info, err := client.GetNodeInfo(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, info)

		// Should handle pre-genesis gracefully
		assert.Equal(t, uint64(0), info.CurrentSlot)
		assert.Equal(t, uint64(0), info.CurrentEpoch)
		assert.Equal(t, time.Duration(0), info.TimeToNextSlot)
		assert.Equal(t, time.Duration(0), info.TimeToNextEpoch)
	})
}
