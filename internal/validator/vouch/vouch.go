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
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/watcheth/watcheth/internal/common"
	"github.com/watcheth/watcheth/internal/logger"
	"github.com/watcheth/watcheth/internal/validator"
)

type VouchClient struct {
	name       string
	endpoint   string
	httpClient *http.Client
}

func NewVouchClient(name, endpoint string) *VouchClient {
	return &VouchClient{
		name:       name,
		endpoint:   endpoint,
		httpClient: common.NewHTTPClient(10 * time.Second),
	}
}

func (c *VouchClient) GetNodeInfo(ctx context.Context) (*validator.ValidatorNodeInfo, error) {
	info := &validator.ValidatorNodeInfo{
		Name:       c.name,
		Endpoint:   c.endpoint,
		LastUpdate: time.Now(),
	}

	metrics, err := c.fetchMetrics(ctx)
	if err != nil {
		info.IsConnected = false
		info.LastError = err
		logger.Error("[%s]: Failed to fetch metrics: %v", c.name, err)
		return info, nil
	}

	info.IsConnected = true
	c.parseMetrics(metrics, info)

	logger.Info("[%s]: Successfully connected and retrieved validator metrics", c.name)
	return info, nil
}

func (c *VouchClient) fetchMetrics(ctx context.Context) (map[string]*io_prometheus_client.MetricFamily, error) {
	// Don't append /metrics if it's already in the endpoint
	url := c.endpoint
	if !strings.HasSuffix(c.endpoint, "/metrics") {
		url = fmt.Sprintf("%s/metrics", c.endpoint)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from metrics endpoint", resp.StatusCode)
	}

	return c.parsePrometheusResponse(resp.Body)
}

func (c *VouchClient) parsePrometheusResponse(r io.Reader) (map[string]*io_prometheus_client.MetricFamily, error) {
	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}
	return metricFamilies, nil
}

func (c *VouchClient) parseMetrics(metricFamilies map[string]*io_prometheus_client.MetricFamily, info *validator.ValidatorNodeInfo) {
	// Service readiness
	if mf, ok := metricFamilies["vouch_ready"]; ok && len(mf.Metric) > 0 {
		if mf.Metric[0].Gauge != nil && mf.Metric[0].Gauge.Value != nil {
			info.Ready = *mf.Metric[0].Gauge.Value > 0
		}
	}

	// Attestation mark seconds (average from histogram)
	if mf, ok := metricFamilies["vouch_attestation_mark_seconds"]; ok {
		if sum, count := getHistogramSumAndCount(mf); count > 0 {
			info.AttestationMarkSeconds = sum / count
		}
	}

	// Attestation success rate and counts
	if mf, ok := metricFamilies["vouch_attestation_process_requests_total"]; ok {
		for _, m := range mf.Metric {
			result := getLabelValue(m.Label, "result")
			if m.Counter != nil && m.Counter.Value != nil {
				if result == "succeeded" {
					info.AttestationSucceeded = uint64(*m.Counter.Value)
				} else if result == "failed" {
					info.AttestationFailed = uint64(*m.Counter.Value)
				}
			}
		}
	}
	total := info.AttestationSucceeded + info.AttestationFailed
	if total > 0 {
		info.AttestationSuccessRate = float64(info.AttestationSucceeded) / float64(total) * 100
	}

	// Block proposal mark seconds
	if mf, ok := metricFamilies["vouch_beaconblockproposal_mark_seconds"]; ok {
		if sum, count := getHistogramSumAndCount(mf); count > 0 {
			info.BlockProposalMarkSeconds = sum / count
		}
	}

	// Block proposal success rate and counts
	// Note: These are the same metrics as BeaconBlockProposalSucceeded/Failed
	if mf, ok := metricFamilies["vouch_beaconblockproposal_process_requests_total"]; ok {
		for _, m := range mf.Metric {
			result := getLabelValue(m.Label, "result")
			if m.Counter != nil && m.Counter.Value != nil {
				if result == "succeeded" {
					info.BlockProposalSucceeded = uint64(*m.Counter.Value)
				} else if result == "failed" {
					info.BlockProposalFailed = uint64(*m.Counter.Value)
				}
			}
		}
	}
	proposalTotal := info.BlockProposalSucceeded + info.BlockProposalFailed
	if proposalTotal > 0 {
		info.BlockProposalSuccessRate = float64(info.BlockProposalSucceeded) / float64(proposalTotal) * 100
	}

	// Beacon node response time (average from histogram, convert to milliseconds)
	if mf, ok := metricFamilies["vouch_client_operation_duration_seconds"]; ok {
		if sum, count := getHistogramSumAndCount(mf); count > 0 {
			info.BeaconNodeResponseTime = (sum / count) * 1000
		}
	}

	// Best bid relay count
	if mf, ok := metricFamilies["vouch_beaconblockproposer_best_bid_relays"]; ok && len(mf.Metric) > 0 {
		if mf.Metric[0].Gauge != nil && mf.Metric[0].Gauge.Value != nil {
			info.BestBidRelayCount = uint64(*mf.Metric[0].Gauge.Value)
		}
	}

	// Blocks from relay
	if mf, ok := metricFamilies["vouch_beaconblockproposal_process_blocks_total"]; ok {
		for _, m := range mf.Metric {
			method := getLabelValue(m.Label, "method")
			if method == "relay" && m.Counter != nil && m.Counter.Value != nil {
				info.BlocksFromRelay = uint64(*m.Counter.Value)
			}
		}
	}

	// Relay auction duration and count (from histogram)
	if mf, ok := metricFamilies["vouch_relay_auction_block_duration_seconds"]; ok {
		if sum, count := getHistogramSumAndCount(mf); count > 0 {
			info.RelayAuctionDuration = sum / count
			info.RelayAuctionCount = uint64(count)
		}
	}

	// Relay validator registrations
	if mf, ok := metricFamilies["vouch_relay_validator_registrations_total"]; ok {
		for _, m := range mf.Metric {
			result := getLabelValue(m.Label, "result")
			if m.Counter != nil && m.Counter.Value != nil {
				if result == "succeeded" {
					info.RelayRegistrationSucceeded = uint64(*m.Counter.Value)
				} else if result == "failed" {
					info.RelayRegistrationFailed = uint64(*m.Counter.Value)
				}
			}
		}
	}

	// Relay builder bid requests
	if mf, ok := metricFamilies["vouch_relay_builder_bid_total"]; ok {
		for _, m := range mf.Metric {
			result := getLabelValue(m.Label, "result")
			if m.Counter != nil && m.Counter.Value != nil {
				if result == "succeeded" {
					info.RelayBuilderBidSucceeded = uint64(*m.Counter.Value)
				} else if result == "failed" {
					info.RelayBuilderBidFailed = uint64(*m.Counter.Value)
				}
			}
		}
	}

	// Relay execution config requests
	if mf, ok := metricFamilies["vouch_relay_execution_config_total"]; ok {
		for _, m := range mf.Metric {
			result := getLabelValue(m.Label, "result")
			if m.Counter != nil && m.Counter.Value != nil {
				if result == "succeeded" {
					info.RelayExecutionConfigSucceeded = uint64(*m.Counter.Value)
				} else if result == "failed" {
					info.RelayExecutionConfigFailed = uint64(*m.Counter.Value)
				}
			}
		}
	}

	// Validator states (vouch_accountmanager_accounts_total)
	info.ValidatorStates = make(map[string]uint64)
	if mf, ok := metricFamilies["vouch_accountmanager_accounts_total"]; ok {
		for _, m := range mf.Metric {
			state := getLabelValue(m.Label, "state")
			if state != "" && m.Gauge != nil && m.Gauge.Value != nil {
				info.ValidatorStates[state] = uint64(*m.Gauge.Value)
			}
		}
	}
}

// Helper functions

func getLabelValue(labels []*io_prometheus_client.LabelPair, name string) string {
	for _, label := range labels {
		if label.Name != nil && *label.Name == name && label.Value != nil {
			return *label.Value
		}
	}
	return ""
}

func getHistogramSumAndCount(mf *io_prometheus_client.MetricFamily) (sum float64, count float64) {
	if mf == nil || len(mf.Metric) == 0 {
		return 0, 0
	}

	for _, m := range mf.Metric {
		if m.Histogram != nil {
			if m.Histogram.SampleSum != nil {
				sum += *m.Histogram.SampleSum
			}
			if m.Histogram.SampleCount != nil {
				count += float64(*m.Histogram.SampleCount)
			}
		}
	}

	return sum, count
}
