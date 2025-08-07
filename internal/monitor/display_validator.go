package monitor

import (
	"fmt"
	"strings"

	"github.com/watcheth/watcheth/internal/validator"
)

func (d *Display) updateValidatorTable(infos []*validator.ValidatorNodeInfo) {
	d.updateValidatorSummary(infos)
	// Individual tables removed - summary provides comprehensive overview
}

// createProgressBar creates an ASCII progress bar
func createProgressBar(percentage float64, width int) string {
	if width <= 0 {
		width = 20
	}

	// Ensure percentage is between 0 and 100
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	filled := int(percentage * float64(width) / 100)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return bar
}

// calculateAggregateMetrics aggregates metrics across all validators
func calculateAggregateMetrics(infos []*validator.ValidatorNodeInfo) map[string]interface{} {
	metrics := make(map[string]interface{})

	var (
		totalAttestSucceeded     uint64
		totalAttestFailed        uint64
		totalPropSucceeded       uint64
		totalPropFailed          uint64
		totalRelayRegSucceeded   uint64
		totalRelayRegFailed      uint64
		totalBuilderBidSucceeded uint64
		totalBuilderBidFailed    uint64
		totalExecConfigSucceeded uint64
		totalExecConfigFailed    uint64
		totalLatency             float64
		activeCount              int
		readyCount               int
	)

	for _, info := range infos {
		if info == nil {
			continue
		}

		if info.IsConnected {
			activeCount++
			if info.Ready {
				readyCount++
			}

			totalAttestSucceeded += info.AttestationSucceeded
			totalAttestFailed += info.AttestationFailed
			totalPropSucceeded += info.BlockProposalSucceeded
			totalPropFailed += info.BlockProposalFailed
			totalRelayRegSucceeded += info.RelayRegistrationSucceeded
			totalRelayRegFailed += info.RelayRegistrationFailed
			totalBuilderBidSucceeded += info.RelayBuilderBidSucceeded
			totalBuilderBidFailed += info.RelayBuilderBidFailed
			totalExecConfigSucceeded += info.RelayExecutionConfigSucceeded
			totalExecConfigFailed += info.RelayExecutionConfigFailed

			if info.BeaconNodeResponseTime > 0 {
				totalLatency += info.BeaconNodeResponseTime
			}
		}
	}

	metrics["total"] = len(infos)
	metrics["active"] = activeCount
	metrics["ready"] = readyCount

	// Attestations
	metrics["attestSucceeded"] = totalAttestSucceeded
	metrics["attestTotal"] = totalAttestSucceeded + totalAttestFailed
	if total := totalAttestSucceeded + totalAttestFailed; total > 0 {
		metrics["attestPercent"] = float64(totalAttestSucceeded) * 100 / float64(total)
	} else {
		metrics["attestPercent"] = float64(0)
	}

	// Proposals
	metrics["propSucceeded"] = totalPropSucceeded
	metrics["propTotal"] = totalPropSucceeded + totalPropFailed
	if total := totalPropSucceeded + totalPropFailed; total > 0 {
		metrics["propPercent"] = float64(totalPropSucceeded) * 100 / float64(total)
	} else {
		metrics["propPercent"] = float64(0)
	}

	// Relay Registrations
	metrics["relayRegSucceeded"] = totalRelayRegSucceeded
	metrics["relayRegTotal"] = totalRelayRegSucceeded + totalRelayRegFailed
	if total := totalRelayRegSucceeded + totalRelayRegFailed; total > 0 {
		metrics["relayRegPercent"] = float64(totalRelayRegSucceeded) * 100 / float64(total)
	} else {
		metrics["relayRegPercent"] = float64(0)
	}

	// Builder Bids
	metrics["builderSucceeded"] = totalBuilderBidSucceeded
	metrics["builderTotal"] = totalBuilderBidSucceeded + totalBuilderBidFailed
	if total := totalBuilderBidSucceeded + totalBuilderBidFailed; total > 0 {
		metrics["builderPercent"] = float64(totalBuilderBidSucceeded) * 100 / float64(total)
	} else {
		metrics["builderPercent"] = float64(0)
	}

	// Exec Config
	metrics["execConfigSucceeded"] = totalExecConfigSucceeded
	metrics["execConfigTotal"] = totalExecConfigSucceeded + totalExecConfigFailed
	if total := totalExecConfigSucceeded + totalExecConfigFailed; total > 0 {
		metrics["execConfigPercent"] = float64(totalExecConfigSucceeded) * 100 / float64(total)
	} else {
		metrics["execConfigPercent"] = float64(0)
	}

	// Average latency
	if activeCount > 0 {
		metrics["avgLatency"] = totalLatency / float64(activeCount)
	} else {
		metrics["avgLatency"] = float64(0)
	}

	return metrics
}

// updateValidatorSummary updates the summary display with aggregated metrics
func (d *Display) updateValidatorSummary(infos []*validator.ValidatorNodeInfo) {
	if len(infos) == 0 {
		d.validatorSummary.Clear()
		return
	}

	metrics := calculateAggregateMetrics(infos)

	// Build the summary text
	var summary strings.Builder

	// Header line
	summary.WriteString("  [green::b]Validator Performance Overview[white]\n")

	// Client status line
	summary.WriteString("  ")
	for i, info := range infos {
		if i > 0 {
			summary.WriteString("  ")
		}

		// Extract port from endpoint
		port := extractPort(info.Endpoint)

		// Status indicator
		var statusSymbol, statusColor string
		if info.IsConnected {
			statusSymbol = "●"
			statusColor = "green"
		} else {
			statusSymbol = "○"
			statusColor = "red"
		}

		// Ready indicator
		var readyText, readyColor string
		if info.IsConnected && info.Ready {
			readyText = "Ready"
			readyColor = "green"
		} else {
			readyText = "Not Ready"
			readyColor = "red"
		}

		summary.WriteString(fmt.Sprintf("[%s]%s[white] %s:%s [%s]%s[white]",
			statusColor, statusSymbol, info.Name, port, readyColor, readyText))
	}
	summary.WriteString("\n")

	// Separator line
	summary.WriteString("  [dim]" + strings.Repeat("─", 75) + "[white]\n")

	// Attestations
	attestPercent := metrics["attestPercent"].(float64)
	attestBar := createProgressBar(attestPercent, 20)
	attestColor := getPercentageColor(attestPercent)
	summary.WriteString(fmt.Sprintf("  Attestations: [%s]%s[white] %5.1f%% (%d/%d)\n",
		attestColor, attestBar, attestPercent,
		metrics["attestSucceeded"], metrics["attestTotal"]))

	// Proposals
	propPercent := metrics["propPercent"].(float64)
	propBar := createProgressBar(propPercent, 20)
	propColor := getPercentageColor(propPercent)
	propTotal := metrics["propTotal"].(uint64)
	var propDisplay string
	if propTotal > 0 {
		propDisplay = fmt.Sprintf("(%d/%d)", metrics["propSucceeded"], propTotal)
	} else {
		propDisplay = "(no proposals yet)"
	}
	summary.WriteString(fmt.Sprintf("  Proposals:    [%s]%s[white] %5.1f%% %s\n",
		propColor, propBar, propPercent, propDisplay))

	// Relay Registrations
	relayRegPercent := metrics["relayRegPercent"].(float64)
	relayRegBar := createProgressBar(relayRegPercent, 20)
	relayRegColor := getPercentageColor(relayRegPercent)
	summary.WriteString(fmt.Sprintf("  Relay Regs:   [%s]%s[white] %5.1f%% (%d/%d)\n",
		relayRegColor, relayRegBar, relayRegPercent,
		metrics["relayRegSucceeded"], metrics["relayRegTotal"]))

	// Builder Bids
	builderPercent := metrics["builderPercent"].(float64)
	builderBar := createProgressBar(builderPercent, 20)
	builderColor := getPercentageColor(builderPercent)
	summary.WriteString(fmt.Sprintf("  Builder Bids: [%s]%s[white] %5.1f%% (%d/%d)\n",
		builderColor, builderBar, builderPercent,
		metrics["builderSucceeded"], metrics["builderTotal"]))

	// Exec Config
	execPercent := metrics["execConfigPercent"].(float64)
	execBar := createProgressBar(execPercent, 20)
	execColor := getPercentageColor(execPercent)
	summary.WriteString(fmt.Sprintf("  Exec Config:  [%s]%s[white] %5.1f%% (%d/%d)\n",
		execColor, execBar, execPercent,
		metrics["execConfigSucceeded"], metrics["execConfigTotal"]))

	// Average Latency
	avgLatency := metrics["avgLatency"].(float64)
	latencyPercent := 100.0 - (avgLatency / 5) // Scale: 0ms = 100%, 500ms = 0%
	if latencyPercent < 0 {
		latencyPercent = 0
	}
	latencyBar := createProgressBar(latencyPercent, 20)
	latencyColor := getLatencyColor(avgLatency)
	latencyStatus := getLatencyStatus(avgLatency)
	summary.WriteString(fmt.Sprintf("  Avg Latency:  [%s]%s[white] %3.0fms (%s)",
		latencyColor, latencyBar, avgLatency, latencyStatus))

	d.validatorSummary.SetText(summary.String()).SetDynamicColors(true)
}

func getPercentageColor(percentage float64) string {
	if percentage >= 99 {
		return "green"
	} else if percentage >= 90 {
		return "yellow"
	}
	return "red"
}

func getLatencyColor(latency float64) string {
	if latency <= 50 {
		return "green"
	} else if latency <= 150 {
		return "yellow"
	}
	return "red"
}

func getLatencyStatus(latency float64) string {
	if latency <= 50 {
		return "Excellent"
	} else if latency <= 100 {
		return "Good"
	} else if latency <= 200 {
		return "Fair"
	}
	return "Poor"
}

// extractPort extracts the port number from an endpoint URL
func extractPort(endpoint string) string {
	// Try to extract port from URL like http://localhost:9095/metrics
	if strings.Contains(endpoint, "://") {
		parts := strings.Split(endpoint, "://")
		if len(parts) > 1 {
			hostPort := parts[1]
			// Remove path if present
			if idx := strings.Index(hostPort, "/"); idx != -1 {
				hostPort = hostPort[:idx]
			}
			// Extract port
			if idx := strings.LastIndex(hostPort, ":"); idx != -1 {
				return hostPort[idx+1:]
			}
		}
	}
	// If no port found, return empty
	return ""
}
