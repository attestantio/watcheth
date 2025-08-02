package beacon

import (
	"time"
)

type BeaconNodeInfo struct {
	Name            string
	Endpoint        string
	CurrentSlot     uint64
	HeadSlot        uint64
	JustifiedSlot   uint64
	FinalizedSlot   uint64
	CurrentEpoch    uint64
	JustifiedEpoch  uint64
	FinalizedEpoch  uint64
	SyncDistance    uint64
	IsSyncing       bool
	IsOptimistic    bool
	TimeToNextSlot  time.Duration
	TimeToNextEpoch time.Duration
	IsConnected     bool
	LastError       error
	LastUpdate      time.Time
}

type GenesisResponse struct {
	Data struct {
		GenesisTime           string `json:"genesis_time"`
		GenesisValidatorsRoot string `json:"genesis_validators_root"`
		GenesisForkVersion    string `json:"genesis_fork_version"`
	} `json:"data"`
}

type HeadersResponse struct {
	ExecutionOptimistic bool `json:"execution_optimistic"`
	Finalized          bool `json:"finalized"`
	Data               []struct {
		Root      string `json:"root"`
		Canonical bool   `json:"canonical"`
		Header    struct {
			Message struct {
				Slot          string `json:"slot"`
				ProposerIndex string `json:"proposer_index"`
				ParentRoot    string `json:"parent_root"`
				StateRoot     string `json:"state_root"`
				BodyRoot      string `json:"body_root"`
			} `json:"message"`
			Signature string `json:"signature"`
		} `json:"header"`
	} `json:"data"`
}

type FinalityCheckpointsResponse struct {
	ExecutionOptimistic bool `json:"execution_optimistic"`
	Finalized          bool `json:"finalized"`
	Data               struct {
		PreviousJustified struct {
			Epoch string `json:"epoch"`
			Root  string `json:"root"`
		} `json:"previous_justified"`
		CurrentJustified struct {
			Epoch string `json:"epoch"`
			Root  string `json:"root"`
		} `json:"current_justified"`
		Finalized struct {
			Epoch string `json:"epoch"`
			Root  string `json:"root"`
		} `json:"finalized"`
	} `json:"data"`
}

type SpecResponse struct {
	Data map[string]string `json:"data"`
}

type SyncingResponse struct {
	Data struct {
		HeadSlot     string `json:"head_slot"`
		SyncDistance string `json:"sync_distance"`
		IsSyncing    bool   `json:"is_syncing"`
		IsOptimistic bool   `json:"is_optimistic"`
		ElOffline    bool   `json:"el_offline"`
	} `json:"data"`
}

type NodeVersionResponse struct {
	Data struct {
		Version string `json:"version"`
	} `json:"data"`
}

type ChainConfig struct {
	SecondsPerSlot uint64
	SlotsPerEpoch  uint64
	GenesisTime    time.Time
}