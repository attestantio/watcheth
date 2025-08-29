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

package execution

import (
	"math/big"
	"time"
)

type ExecutionNodeInfo struct {
	Name            string
	Endpoint        string
	CurrentBlock    uint64
	HighestBlock    uint64
	StartingBlock   uint64
	IsSyncing       bool
	SyncProgress    float64 // Percentage 0-100
	PeerCount       uint64
	IsConnected     bool
	LastError       error
	LastUpdate      time.Time
	NodeVersion     string
	ChainID         *big.Int
	GasPrice        *big.Int
	NetworkID       string
	ProtocolVersion string
	BlockTime       time.Duration // Time since last block
	LastBlockTime   time.Time
}

type SyncingResponse struct {
	Result interface{} `json:"result"`
}

type SyncingData struct {
	StartingBlock string `json:"startingBlock"`
	CurrentBlock  string `json:"currentBlock"`
	HighestBlock  string `json:"highestBlock"`
}

type BlockNumberResponse struct {
	Result string `json:"result"`
}

type PeerCountResponse struct {
	Result string `json:"result"`
}

type ChainIDResponse struct {
	Result string `json:"result"`
}

type GasPriceResponse struct {
	Result string `json:"result"`
}

type ClientVersionResponse struct {
	Result string `json:"result"`
}

type NetVersionResponse struct {
	Result string `json:"result"`
}

type ProtocolVersionResponse struct {
	Result string `json:"result"`
}

type BlockResponse struct {
	Result *Block `json:"result"`
}

type Block struct {
	Number     string `json:"number"`
	Timestamp  string `json:"timestamp"`
	Hash       string `json:"hash"`
	ParentHash string `json:"parentHash"`
}
