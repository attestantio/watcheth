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

package testutil

import (
	"time"
)

// Consensus API response fixtures
var (
	ValidNodeIdentityResponse = `{
		"data": {
			"peer_id": "16Uiu2HAm5JH3KzSJPg8qhVcqRn8XHZ3ZkGdkuVDYmZ6HhNqZGVYr",
			"enr": "enr:-Ku4QHWkxWiznJ5L4A8r-YbXK4w-TZH2gYmwVPOg4FHNp8WK7Ew7j4NqPGE6EwGX3QwJB6VqKL7pKPTYT2z0N0iJL4EBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD9pf1WAAAAAP__________gmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQL1k-1cVb",
			"p2p_addresses": ["/ip4/127.0.0.1/tcp/9001/p2p/16Uiu2HAm5JH3KzSJPg8qhVcqRn8XHZ3ZkGdkuVDYmZ6HhNqZGVYr"],
			"discovery_addresses": ["/ip4/127.0.0.1/udp/9000/p2p/16Uiu2HAm5JH3KzSJPg8qhVcqRn8XHZ3ZkGdkuVDYmZ6HhNqZGVYr"],
			"metadata": {
				"seq_number": "1",
				"attnets": "0x0000000000000000",
				"syncnets": "0x00"
			}
		}
	}`

	ValidNodeVersionResponse = `{
		"data": {
			"version": "Lighthouse/v4.5.0-1234567/x86_64-linux"
		}
	}`

	ValidChainConfigResponse = `{
		"data": {
			"GENESIS_TIME": "1606824023",
			"SECONDS_PER_SLOT": "12",
			"SLOTS_PER_EPOCH": "32"
		}
	}`

	ValidSyncingResponse = `{
		"data": {
			"head_slot": "100",
			"sync_distance": "50",
			"is_syncing": true,
			"is_optimistic": false,
			"el_offline": false
		}
	}`

	NotSyncingResponse = `{
		"data": {
			"head_slot": "1000",
			"sync_distance": "0",
			"is_syncing": false,
			"is_optimistic": false,
			"el_offline": false
		}
	}`

	ValidPeerCountResponse = `{
		"data": {
			"disconnected": "0",
			"connecting": "1",
			"connected": "50",
			"disconnecting": "0"
		}
	}`
)

// Execution API response fixtures
var (
	ValidSyncingRPCResponse = `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"startingBlock": "0x0",
			"currentBlock": "0x3e8",
			"highestBlock": "0x7d0"
		}
	}`

	NotSyncingRPCResponse = `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": false
	}`

	ValidClientVersionResponse = `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": "Geth/v1.13.0-stable-1234567/linux-amd64/go1.21.0"
	}`

	ValidChainIDResponse = `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": "0x1"
	}`

	ValidGasPriceResponse = `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": "0x3b9aca00"
	}`

	ValidPeerCountRPCResponse = `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": "0x19"
	}`

	RPCErrorResponse = `{
		"jsonrpc": "2.0",
		"id": 1,
		"error": {
			"code": -32000,
			"message": "Method not found"
		}
	}`
)

// Time utilities
func TestTime() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}

func TestGenesisTime() time.Time {
	return time.Unix(1606824023, 0)
}
