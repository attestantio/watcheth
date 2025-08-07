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
	Ready                      bool    // Service ready status
	AttestationMarkSeconds     float64 // Time into slot when attestations are broadcast
	AttestationSuccessRate     float64 // Percentage of successful attestations
	AttestationSucceeded       uint64  // Number of successful attestations
	AttestationFailed          uint64  // Number of failed attestations
	BlockProposalMarkSeconds   float64 // Time into slot when block is broadcast
	BlockProposalSuccessRate   float64 // Percentage of successful proposals
	BlockProposalSucceeded     uint64  // Number of successful block proposals
	BlockProposalFailed        uint64  // Number of failed block proposals
	BeaconNodeResponseTime     float64 // Average response time in milliseconds
	BestBidRelayCount          uint64  // Number of relays providing best bid
	BlocksFromRelay            uint64  // Blocks built via relay
	RelayAuctionDuration       float64 // Time to get best bid from relays (seconds)
	RelayAuctionCount          uint64  // Number of relay auctions (indicates relay usage)
	RelayRegistrationSucceeded uint64  // Number of successful relay validator registrations
	RelayRegistrationFailed    uint64  // Number of failed relay validator registrations

	// Additional Vouch metrics
	RelayBuilderBidSucceeded      uint64 // Successful relay builder bid requests
	RelayBuilderBidFailed         uint64 // Failed relay builder bid requests
	RelayExecutionConfigSucceeded uint64 // Successful relay execution config requests
	RelayExecutionConfigFailed    uint64 // Failed relay execution config requests
}
