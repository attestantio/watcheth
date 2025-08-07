package validator

import (
	"time"
)

type ValidatorNodeInfo struct {
	Name        string
	Endpoint    string
	IsConnected bool
	LastError   error
	LastUpdate  time.Time

	// Essential metrics
	Ready                    bool    // Service ready status
	AttestationMarkSeconds   float64 // Time into slot when attestations are broadcast
	AttestationSuccessRate   float64 // Percentage of successful attestations
	BlockProposalMarkSeconds float64 // Time into slot when block is broadcast
	BlockProposalSuccessRate float64 // Percentage of successful proposals
	BeaconNodeResponseTime   float64 // Average response time in milliseconds
	BestBidRelayCount        uint64  // Number of relays providing best bid
	BlocksFromRelay          uint64  // Blocks built via relay
}
