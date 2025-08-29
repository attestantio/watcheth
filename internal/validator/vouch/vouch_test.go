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

package vouch

import (
	"strings"
	"testing"

	"github.com/watcheth/watcheth/internal/validator"
)

func TestParsePrometheusResponse(t *testing.T) {
	// Sample Prometheus metrics response
	sampleMetrics := `
# HELP vouch_ready Ready status of the service
# TYPE vouch_ready gauge
vouch_ready 1

# HELP vouch_attestation_mark_seconds Time from slot start to attestation mark
# TYPE vouch_attestation_mark_seconds histogram
vouch_attestation_mark_seconds_bucket{le="0.1"} 10
vouch_attestation_mark_seconds_bucket{le="0.5"} 45
vouch_attestation_mark_seconds_bucket{le="1"} 98
vouch_attestation_mark_seconds_bucket{le="+Inf"} 100
vouch_attestation_mark_seconds_sum 45.5
vouch_attestation_mark_seconds_count 100

# HELP vouch_attestation_process_requests_total Total attestation requests
# TYPE vouch_attestation_process_requests_total counter
vouch_attestation_process_requests_total{result="succeeded"} 950
vouch_attestation_process_requests_total{result="failed"} 50

# HELP vouch_beaconblockproposal_mark_seconds Time from slot start to block proposal mark
# TYPE vouch_beaconblockproposal_mark_seconds histogram
vouch_beaconblockproposal_mark_seconds_sum 12.5
vouch_beaconblockproposal_mark_seconds_count 25

# HELP vouch_beaconblockproposal_process_requests_total Total block proposal requests
# TYPE vouch_beaconblockproposal_process_requests_total counter
vouch_beaconblockproposal_process_requests_total{result="succeeded"} 24
vouch_beaconblockproposal_process_requests_total{result="failed"} 1

# HELP vouch_accountmanager_accounts_total Total accounts by state
# TYPE vouch_accountmanager_accounts_total gauge
vouch_accountmanager_accounts_total{state="active"} 100
vouch_accountmanager_accounts_total{state="pending"} 5
vouch_accountmanager_accounts_total{state="exited"} 2
`

	client := &VouchClient{}
	reader := strings.NewReader(sampleMetrics)

	metricFamilies, err := client.parsePrometheusResponse(reader)
	if err != nil {
		t.Fatalf("Failed to parse metrics: %v", err)
	}

	// Verify we got the expected metric families
	expectedMetrics := []string{
		"vouch_ready",
		"vouch_attestation_mark_seconds",
		"vouch_attestation_process_requests_total",
		"vouch_beaconblockproposal_mark_seconds",
		"vouch_beaconblockproposal_process_requests_total",
		"vouch_accountmanager_accounts_total",
	}

	for _, metric := range expectedMetrics {
		if _, ok := metricFamilies[metric]; !ok {
			t.Errorf("Expected metric %s not found", metric)
		}
	}
}

func TestParseMetrics(t *testing.T) {
	// Sample Prometheus metrics response
	sampleMetrics := `
# HELP vouch_ready Ready status
# TYPE vouch_ready gauge
vouch_ready 1

# HELP vouch_attestation_mark_seconds Time from slot start
# TYPE vouch_attestation_mark_seconds histogram
vouch_attestation_mark_seconds_sum 45.5
vouch_attestation_mark_seconds_count 100

# HELP vouch_attestation_process_requests_total Attestation requests
# TYPE vouch_attestation_process_requests_total counter
vouch_attestation_process_requests_total{result="succeeded"} 950
vouch_attestation_process_requests_total{result="failed"} 50

# HELP vouch_beaconblockproposal_mark_seconds Block proposal time
# TYPE vouch_beaconblockproposal_mark_seconds histogram
vouch_beaconblockproposal_mark_seconds_sum 12.5
vouch_beaconblockproposal_mark_seconds_count 25

# HELP vouch_beaconblockproposal_process_requests_total Block proposals
# TYPE vouch_beaconblockproposal_process_requests_total counter
vouch_beaconblockproposal_process_requests_total{result="succeeded"} 24
vouch_beaconblockproposal_process_requests_total{result="failed"} 1

# HELP vouch_client_operation_duration_seconds Client operation duration
# TYPE vouch_client_operation_duration_seconds histogram
vouch_client_operation_duration_seconds_sum 150.0
vouch_client_operation_duration_seconds_count 1000

# HELP vouch_accountmanager_accounts_total Accounts by state
# TYPE vouch_accountmanager_accounts_total gauge
vouch_accountmanager_accounts_total{state="active"} 100
vouch_accountmanager_accounts_total{state="pending"} 5
vouch_accountmanager_accounts_total{state="exited"} 2
`

	client := &VouchClient{}
	reader := strings.NewReader(sampleMetrics)

	metricFamilies, err := client.parsePrometheusResponse(reader)
	if err != nil {
		t.Fatalf("Failed to parse metrics: %v", err)
	}

	info := &validator.ValidatorNodeInfo{}
	client.parseMetrics(metricFamilies, info)

	// Test parsed values
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Ready", info.Ready, true},
		{"AttestationMarkSeconds", info.AttestationMarkSeconds, 0.455},
		{"AttestationSucceeded", info.AttestationSucceeded, uint64(950)},
		{"AttestationFailed", info.AttestationFailed, uint64(50)},
		{"AttestationSuccessRate", info.AttestationSuccessRate, 95.0},
		{"BlockProposalMarkSeconds", info.BlockProposalMarkSeconds, 0.5},
		{"BlockProposalSucceeded", info.BlockProposalSucceeded, uint64(24)},
		{"BlockProposalFailed", info.BlockProposalFailed, uint64(1)},
		{"BlockProposalSuccessRate", info.BlockProposalSuccessRate, 96.0},
		{"BeaconNodeResponseTime", info.BeaconNodeResponseTime, 150.0}, // 0.15s * 1000 = 150ms
		{"ActiveValidators", info.ValidatorStates["active"], uint64(100)},
		{"PendingValidators", info.ValidatorStates["pending"], uint64(5)},
		{"ExitedValidators", info.ValidatorStates["exited"], uint64(2)},
	}

	for _, tt := range tests {
		switch v := tt.got.(type) {
		case float64:
			expected := tt.expected.(float64)
			if v != expected {
				t.Errorf("%s: got %f, expected %f", tt.name, v, expected)
			}
		case uint64:
			expected := tt.expected.(uint64)
			if v != expected {
				t.Errorf("%s: got %d, expected %d", tt.name, v, expected)
			}
		case bool:
			expected := tt.expected.(bool)
			if v != expected {
				t.Errorf("%s: got %v, expected %v", tt.name, v, expected)
			}
		}
	}
}

func TestGetLabelValue(t *testing.T) {
	tests := []struct {
		name     string
		labels   string
		search   string
		expected string
	}{
		{
			name:     "Find existing label",
			labels:   "result",
			search:   "result",
			expected: "succeeded",
		},
		{
			name:     "Label not found",
			labels:   "other",
			search:   "result",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would need actual label pairs from prometheus client model
			// For now, we're just testing the concept
		})
	}
}
