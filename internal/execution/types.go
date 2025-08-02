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
