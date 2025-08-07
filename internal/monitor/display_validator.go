package monitor

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/watcheth/watcheth/internal/validator"
)

func (d *Display) updateValidatorTable(infos []*validator.ValidatorNodeInfo) {
	d.updateValidatorPerfTable(infos)
	d.updateValidatorRelayTable(infos)
}

func (d *Display) updateValidatorPerfTable(infos []*validator.ValidatorNodeInfo) {
	if infos == nil {
		infos = []*validator.ValidatorNodeInfo{}
	}

	// Ensure we have enough rows in the table
	currentRows := d.validatorTable.GetRowCount()
	neededRows := len(infos) + 1 // +1 for header

	// Add rows if needed
	columnCount := len(d.getValidatorHeaders())
	for i := currentRows; i < neededRows; i++ {
		for j := 0; j < columnCount; j++ {
			d.validatorTable.SetCell(i, j, tview.NewTableCell(""))
		}
	}

	// Update table rows
	for row, info := range infos {
		if info == nil {
			continue
		}

		tableRow := row + 1 // +1 for header
		col := 0

		// Client name
		d.setValidatorCell(tableRow, col, info.Name, tcell.ColorWhite)
		col++

		// Port
		port := parsePortFromEndpoint(info.Endpoint)
		d.setValidatorCell(tableRow, col, port, tcell.ColorWhite)
		col++

		// Status
		var status string
		var statusColor tcell.Color
		if info.IsConnected {
			status = "● Connected"
			statusColor = tcell.ColorGreen
		} else {
			status = "○ Offline"
			statusColor = tcell.ColorRed
		}
		d.setValidatorCell(tableRow, col, status, statusColor)
		col++

		// Ready status
		var readyText string
		var readyColor tcell.Color
		if info.IsConnected {
			if info.Ready {
				readyText = "Yes"
				readyColor = tcell.ColorGreen
			} else {
				readyText = "No"
				readyColor = tcell.ColorRed
			}
		} else {
			readyText = "-"
			readyColor = tcell.ColorGray
		}
		d.setValidatorCell(tableRow, col, readyText, readyColor)
		col++

		// Attestation Performance - Show numbers and percentage
		var attText string
		var attColor tcell.Color
		if info.IsConnected {
			total := info.AttestationSucceeded + info.AttestationFailed
			if total > 0 {
				successRate := float64(info.AttestationSucceeded) / float64(total) * 100
				attText = fmt.Sprintf("%d/%d (%.0f%%)", info.AttestationSucceeded, total, successRate)
				if successRate >= 95 {
					attColor = tcell.ColorGreen
				} else if successRate >= 85 {
					attColor = tcell.ColorYellow
				} else {
					attColor = tcell.ColorRed
				}
			} else {
				attText = "0/0"
				attColor = tcell.ColorGray
			}
		} else {
			attText = "-"
			attColor = tcell.ColorGray
		}
		d.setValidatorCell(tableRow, col, attText, attColor)
		col++

		// Proposal Performance - Show numbers and percentage
		var propText string
		var propColor tcell.Color
		if info.IsConnected {
			total := info.BlockProposalSucceeded + info.BlockProposalFailed
			if total > 0 {
				successRate := float64(info.BlockProposalSucceeded) / float64(total) * 100
				propText = fmt.Sprintf("%d/%d (%.0f%%)", info.BlockProposalSucceeded, total, successRate)
				if successRate >= 95 {
					propColor = tcell.ColorGreen
				} else if successRate >= 85 {
					propColor = tcell.ColorYellow
				} else {
					propColor = tcell.ColorRed
				}
			} else {
				propText = "0/0"
				propColor = tcell.ColorGray
			}
		} else {
			propText = "-"
			propColor = tcell.ColorGray
		}
		d.setValidatorCell(tableRow, col, propText, propColor)
		col++

		// Client Latency
		var clientLatencyText string
		var clientLatencyColor tcell.Color
		if info.IsConnected && info.BeaconNodeResponseTime > 0 {
			clientLatencyText = fmt.Sprintf("%.0fms", info.BeaconNodeResponseTime)
			if info.BeaconNodeResponseTime <= 100 {
				clientLatencyColor = tcell.ColorGreen
			} else if info.BeaconNodeResponseTime <= 250 {
				clientLatencyColor = tcell.ColorYellow
			} else {
				clientLatencyColor = tcell.ColorRed
			}
		} else {
			clientLatencyText = "-"
			clientLatencyColor = tcell.ColorGray
		}
		d.setValidatorCell(tableRow, col, clientLatencyText, clientLatencyColor)
		col++
	}
}

func (d *Display) updateValidatorRelayTable(infos []*validator.ValidatorNodeInfo) {
	if infos == nil {
		infos = []*validator.ValidatorNodeInfo{}
	}

	// Ensure we have enough rows in the table
	currentRows := d.validatorRelayTable.GetRowCount()
	neededRows := len(infos) + 1 // +1 for header

	// Add rows if needed
	columnCount := len(d.getValidatorRelayHeaders())
	for i := currentRows; i < neededRows; i++ {
		for j := 0; j < columnCount; j++ {
			d.validatorRelayTable.SetCell(i, j, tview.NewTableCell(""))
		}
	}

	// Update table rows
	for row, info := range infos {
		if info == nil {
			continue
		}

		tableRow := row + 1 // +1 for header
		col := 0

		// Client name
		d.setValidatorRelayCell(tableRow, col, info.Name, tcell.ColorWhite)
		col++

		// Relay Registrations - Show numbers and percentage
		var relayText string
		var relayColor tcell.Color
		if info.IsConnected {
			total := info.RelayRegistrationSucceeded + info.RelayRegistrationFailed
			if total > 0 {
				successRate := float64(info.RelayRegistrationSucceeded) / float64(total) * 100
				// Format: "1410/1430 (98.5%)"
				relayText = fmt.Sprintf("%d/%d (%.0f%%)", info.RelayRegistrationSucceeded, total, successRate)
				if successRate >= 99 {
					relayColor = tcell.ColorGreen
				} else if successRate >= 90 {
					relayColor = tcell.ColorYellow
				} else {
					relayColor = tcell.ColorRed
				}
			} else {
				relayText = "0/0"
				relayColor = tcell.ColorGray
			}
		} else {
			relayText = "-"
			relayColor = tcell.ColorGray
		}
		d.setValidatorRelayCell(tableRow, col, relayText, relayColor)
		col++

		// Builder Bids - Show numbers and percentage
		var builderBidsText string
		var builderBidsColor tcell.Color
		if info.IsConnected {
			total := info.RelayBuilderBidSucceeded + info.RelayBuilderBidFailed
			if total > 0 {
				successRate := float64(info.RelayBuilderBidSucceeded) / float64(total) * 100
				builderBidsText = fmt.Sprintf("%d/%d (%.0f%%)", info.RelayBuilderBidSucceeded, total, successRate)
				if successRate >= 99 {
					builderBidsColor = tcell.ColorGreen
				} else if successRate >= 90 {
					builderBidsColor = tcell.ColorYellow
				} else {
					builderBidsColor = tcell.ColorRed
				}
			} else {
				builderBidsText = "0/0"
				builderBidsColor = tcell.ColorGray
			}
		} else {
			builderBidsText = "-"
			builderBidsColor = tcell.ColorGray
		}
		d.setValidatorRelayCell(tableRow, col, builderBidsText, builderBidsColor)
		col++

		// Execution Config - Show numbers and percentage
		var execConfigText string
		var execConfigColor tcell.Color
		if info.IsConnected {
			total := info.RelayExecutionConfigSucceeded + info.RelayExecutionConfigFailed
			if total > 0 {
				successRate := float64(info.RelayExecutionConfigSucceeded) / float64(total) * 100
				execConfigText = fmt.Sprintf("%d/%d (%.0f%%)", info.RelayExecutionConfigSucceeded, total, successRate)
				if successRate >= 99 {
					execConfigColor = tcell.ColorGreen
				} else if successRate >= 90 {
					execConfigColor = tcell.ColorYellow
				} else {
					execConfigColor = tcell.ColorRed
				}
			} else {
				execConfigText = "0/0"
				execConfigColor = tcell.ColorGray
			}
		} else {
			execConfigText = "-"
			execConfigColor = tcell.ColorGray
		}
		d.setValidatorRelayCell(tableRow, col, execConfigText, execConfigColor)
		col++

		// Relay Auction Duration - Show average time
		var relayDurText string
		var relayDurColor tcell.Color
		if info.IsConnected {
			if info.RelayAuctionCount > 0 && info.RelayAuctionDuration > 0 {
				relayDurText = fmt.Sprintf("%.2fs", info.RelayAuctionDuration)
				// Color based on auction speed
				if info.RelayAuctionDuration <= 2.0 {
					relayDurColor = tcell.ColorGreen // Fast
				} else if info.RelayAuctionDuration <= 4.0 {
					relayDurColor = tcell.ColorYellow // Moderate
				} else {
					relayDurColor = tcell.ColorRed // Slow
				}
			} else {
				relayDurText = "-"
				relayDurColor = tcell.ColorGray
			}
		} else {
			relayDurText = "-"
			relayDurColor = tcell.ColorGray
		}
		d.setValidatorRelayCell(tableRow, col, relayDurText, relayDurColor)
		col++
	}
}

func (d *Display) setValidatorRelayCell(row, col int, text string, color tcell.Color) {
	d.setCell(d.validatorRelayTable, row, col, text, color)
}
