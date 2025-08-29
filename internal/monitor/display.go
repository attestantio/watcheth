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

package monitor

import (
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/watcheth/watcheth/internal/config"
	"github.com/watcheth/watcheth/internal/consensus"
	"github.com/watcheth/watcheth/internal/execution"
)

// Status symbols for visual indicators
const (
	StatusSymbolSynced     = "●"
	StatusSymbolSyncing    = "◐"
	StatusSymbolOptimistic = "◑"
	StatusSymbolOffline    = "○"
)

// Animation frames for the title
var titleAnimationFrames = []string{
	"   /\\_/\\     \n  ( o.o )    \n   > ^ <     \n  watcheth    ",
	"   /\\_/\\     \n  ( o.o )    \n   > ^ <     \n  watcheth    ",
	"   /\\_/\\     \n  ( -.- )    \n   > ^ <     \n  watcheth    ",
	"   /\\_/\\     \n  ( o.o )    \n   > ^ <     \n  watcheth    ",
}

type Display struct {
	app               *tview.Application
	consensusTable    *tview.Table
	executionTable    *tview.Table
	validatorSummary  *tview.TextView
	monitor           *Monitor
	help              *tview.TextView
	refreshInterval   time.Duration
	nextRefresh       time.Time
	countdownTicker   *time.Ticker
	title             *tview.TextView
	animationTicker   *time.Ticker
	animationFrame    int
	logView           *tview.TextView
	logReader         *LogReader
	logUpdateTicker   *time.Ticker
	showLogs          bool
	selectedLogClient int
	clientNames       []string
	nextSlotTime      time.Duration   // Time to next slot
	consensusHeader   *tview.TextView // Header for consensus section
	showVersions      bool            // Toggle for showing version columns
}

func NewDisplay(monitor *Monitor) *Display {
	return &Display{
		app:               tview.NewApplication(),
		consensusTable:    tview.NewTable(),
		executionTable:    tview.NewTable(),
		validatorSummary:  tview.NewTextView(),
		monitor:           monitor,
		help:              tview.NewTextView(),
		title:             tview.NewTextView(),
		logView:           tview.NewTextView(),
		logReader:         NewLogReader(),
		refreshInterval:   monitor.GetRefreshInterval(),
		nextRefresh:       time.Now().Add(monitor.GetRefreshInterval()),
		animationFrame:    0,
		showLogs:          false,
		selectedLogClient: 0,
		clientNames:       []string{},
		consensusHeader:   tview.NewTextView(),
		showVersions:      false, // Hidden by default
	}
}

func (d *Display) Run() error {
	d.setupTables()
	d.setupLayout()

	// Ensure log update ticker is cleaned up
	defer func() {
		if d.logUpdateTicker != nil {
			d.logUpdateTicker.Stop()
		}
	}()

	// Start countdown ticker
	d.countdownTicker = time.NewTicker(time.Second)
	go d.countdownLoop()

	// Start animation ticker
	d.animationTicker = time.NewTicker(500 * time.Millisecond)
	go d.animationLoop()

	go d.updateLoop()

	return d.app.Run()
}

func (d *Display) SetupLogPaths(clientConfigs []config.ClientConfig) {
	d.clientNames = make([]string, len(clientConfigs))

	// Set up log paths for each client
	for i, cfg := range clientConfigs {
		d.clientNames[i] = cfg.Name
		if logPath := cfg.GetLogPath(); logPath != "" {
			d.logReader.SetLogPath(cfg.Name, logPath)
		}
	}

	// Start a ticker for frequent log updates (100ms for near real-time)
	d.logUpdateTicker = time.NewTicker(100 * time.Millisecond)
	go d.logUpdateLoop()
}

func (d *Display) setupTables() {
	// Setup consensus table
	d.consensusTable.Clear()
	d.consensusTable.SetBorders(false).
		SetFixed(1, 0).
		SetSelectable(false, false)

	// Setup execution table
	d.executionTable.Clear()
	d.executionTable.SetBorders(false).
		SetFixed(1, 0).
		SetSelectable(false, false)

	// Set up header rows
	for col, header := range d.getConsensusHeaders() {
		var paddedHeader string
		if col == 0 {
			paddedHeader = "  " + header + " "
		} else {
			paddedHeader = " " + header + " "
		}
		cell := tview.NewTableCell(paddedHeader).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		d.consensusTable.SetCell(0, col, cell)
	}

	for col, header := range d.getExecutionHeaders() {
		var paddedHeader string
		if col == 0 {
			paddedHeader = "  " + header + " "
		} else {
			paddedHeader = " " + header + " "
		}
		cell := tview.NewTableCell(paddedHeader).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		d.executionTable.SetCell(0, col, cell)
	}
}

func (d *Display) getConsensusHeaders() []string {
	headers := []string{
		"Client",
		"Port",
		"Status",
		"EL Offline",
		"Slot",
		"Peers",
		"Epoch/Final",
	}
	if d.showVersions {
		headers = append(headers, "Version")
	}
	headers = append(headers, "Fork")
	return headers
}

func (d *Display) getExecutionHeaders() []string {
	headers := []string{
		"Client",
		"Port",
		"Status",
		"Block",
		"Peers",
		"Gas Price",
		"Chain ID",
	}
	if d.showVersions {
		headers = append(headers, "Version")
	}
	return headers
}

func (d *Display) setupLayout() {
	// Initialize title
	d.title.SetText(titleAnimationFrames[0]).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorGreen)

	d.updateHelpText()
	d.help.SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorBlack)

	// Setup log view
	d.logView.SetBorder(true).
		SetTitle(" Logs ").
		SetTitleAlign(tview.AlignLeft)

	d.updateLayout()
}

func (d *Display) updateLayout() {
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(d.title, 4, 0, false). // Simple cat animation
		AddItem(nil, 1, 0, false)      // Empty space

	// Check if we have validator clients for summary
	hasValidators := len(d.monitor.GetValidatorInfos()) > 0
	if hasValidators {
		// Setup validator summary box
		d.validatorSummary.SetBorder(false)
		d.validatorSummary.SetDynamicColors(true)
		d.validatorSummary.SetWrap(false)
		// Count lines in validator summary:
		// 1 header + 1 status + 1 blank + 6 metrics + 2 blanks + 1 states header + 5 state lines + 1 total = 19 lines
		validatorLines := 19                                       // Fixed height for validator summary
		flex.AddItem(d.validatorSummary, validatorLines, 0, false) // Fixed height

		// Add empty space after validator summary
		flex.AddItem(nil, 1, 0, false)
	}

	// Consensus clients section with slot countdown
	d.updateConsensusHeader()
	d.consensusHeader.SetTextColor(tcell.ColorGreen)

	// Calculate table heights based on actual row counts
	consensusRowCount := d.consensusTable.GetRowCount()
	if consensusRowCount == 0 {
		consensusRowCount = 1 // At least show header
	}
	consensusHeight := consensusRowCount + 1 // +1 for section header

	executionRowCount := d.executionTable.GetRowCount()
	if executionRowCount == 0 {
		executionRowCount = 1 // At least show header
	}
	executionHeight := executionRowCount + 2 // +2 for empty space and section header

	consensusSection := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(d.consensusHeader, 1, 0, false).
		AddItem(d.consensusTable, consensusRowCount, 0, true)

	// Execution clients section
	executionSection := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false). // Empty space for separation
		AddItem(tview.NewTextView().SetText("  ● Execution Clients").SetTextColor(tcell.ColorGreen), 1, 0, false).
		AddItem(d.executionTable, executionRowCount, 0, false)

	tablesArea := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(consensusSection, consensusHeight, 0, true).
		AddItem(executionSection, executionHeight, 0, false)

	if d.showLogs {
		// Split view: tables and logs
		mainArea := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(tablesArea, 0, 7, true). // 70% for tables
			AddItem(d.logView, 0, 3, false)  // 30% for logs

		flex.AddItem(mainArea, 0, 1, true)
	} else {
		// Tables only
		flex.AddItem(tablesArea, 0, 1, true)
	}

	flex.AddItem(d.help, 1, 0, false)

	d.app.SetRoot(flex, true).EnableMouse(false)

	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q', 'Q':
			d.app.Stop()
			return nil
		case 'r', 'R':
			go d.updateTables(d.monitor.GetNodeInfos())
			// Reset the next refresh time when manually refreshing
			d.nextRefresh = time.Now().Add(d.refreshInterval)
			return nil
		case 'L':
			// Toggle log view
			d.showLogs = !d.showLogs
			d.updateHelpText()
			d.updateLayout()
			if d.showLogs {
				d.updateLogView()
			}
			return nil
		case 'j':
			// Next client's logs (vim down)
			if d.showLogs && len(d.clientNames) > 0 {
				d.selectedLogClient = (d.selectedLogClient + 1) % len(d.clientNames)
				d.updateLogView()
			}
			return nil
		case 'k':
			// Previous client's logs (vim up)
			if d.showLogs && len(d.clientNames) > 0 {
				d.selectedLogClient = (d.selectedLogClient - 1 + len(d.clientNames)) % len(d.clientNames)
				d.updateLogView()
			}
			return nil
		case 'g':
			// First client's logs
			if d.showLogs && len(d.clientNames) > 0 {
				d.selectedLogClient = 0
				d.updateLogView()
			}
			return nil
		case 'G':
			// Last client's logs
			if d.showLogs && len(d.clientNames) > 0 {
				d.selectedLogClient = len(d.clientNames) - 1
				d.updateLogView()
			}
			return nil
		case 'v', 'V':
			// Toggle version columns
			d.showVersions = !d.showVersions
			d.setupTables()
			go d.updateTables(d.monitor.GetNodeInfos())
			d.updateHelpText()
			return nil
		}

		return event
	})
}

func (d *Display) updateLoop() {
	// Initial update
	infos := d.monitor.GetNodeInfos()
	d.updateTables(infos)

	// Listen for updates
	for infos := range d.monitor.Updates() {
		d.updateTables(infos)
		// Reset the next refresh time
		d.nextRefresh = time.Now().Add(d.refreshInterval)

		// Update logs if visible
		if d.showLogs {
			d.updateLogView()
		}
	}
}

func (d *Display) updateTables(update NodeUpdate) {
	// Validate inputs
	if d == nil || d.app == nil || d.consensusTable == nil || d.executionTable == nil {
		return
	}

	d.app.QueueUpdateDraw(func() {
		// Update consensus table
		d.updateConsensusTable(update.ConsensusInfos)

		// Update execution table
		d.updateExecutionTable(update.ExecutionInfos)

		// Update validator table
		d.updateValidatorTable(update.ValidatorInfos)

		// Update layout if validator clients were added/removed
		if len(update.ValidatorInfos) > 0 {
			d.updateLayout()
		}
	})
}

func (d *Display) updateConsensusTable(infos []*consensus.ConsensusNodeInfo) {
	if infos == nil {
		infos = []*consensus.ConsensusNodeInfo{}
	}

	// Update next slot time from any connected consensus node
	for _, info := range infos {
		if info != nil && info.IsConnected && info.TimeToNextSlot > 0 {
			d.nextSlotTime = info.TimeToNextSlot
			break // All nodes should have the same time to next slot
		}
	}

	// Ensure we have enough rows in the table
	currentRows := d.consensusTable.GetRowCount()
	neededRows := len(infos) + 1 // +1 for header

	// Add rows if needed
	columnCount := len(d.getConsensusHeaders())
	for i := currentRows; i < neededRows; i++ {
		for j := 0; j < columnCount; j++ {
			d.consensusTable.SetCell(i, j, tview.NewTableCell(""))
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
		d.setConsensusCell(tableRow, col, info.Name, tcell.ColorWhite)
		col++

		// Port
		port := parsePortFromEndpoint(info.Endpoint)
		d.setConsensusCell(tableRow, col, port, tcell.ColorWhite)
		col++

		// Status with symbol
		status, statusColor, statusSymbol := d.getStatusInfo(info)
		statusText := fmt.Sprintf("%s %s", statusSymbol, status)
		d.setConsensusCell(tableRow, col, statusText, statusColor)
		col++

		// EL Offline status
		var elOfflineText string
		var elOfflineColor tcell.Color
		if info.IsConnected {
			if info.ElOffline {
				elOfflineText = "Yes"
				elOfflineColor = tcell.ColorRed
			} else {
				elOfflineText = "No"
				elOfflineColor = tcell.ColorGreen
			}
		} else {
			elOfflineText = "-"
			elOfflineColor = tcell.ColorGray
		}
		d.setConsensusCell(tableRow, col, elOfflineText, elOfflineColor)
		col++

		// Slot with arrow notation when syncing
		if info.IsConnected {
			slotText := fmt.Sprintf("%d", info.CurrentSlot)
			d.setConsensusCellWithColoredArrow(tableRow, col, slotText, info.SyncDistance > 0, info.SyncDistance, tcell.ColorWhite, 50, 100)
		} else {
			d.setConsensusCell(tableRow, col, "-", tcell.ColorGray)
		}
		col++

		// Peers with color
		var peerText string
		var peerColor tcell.Color
		if info.IsConnected && info.PeerCount > 0 {
			peerText = fmt.Sprintf("%d", info.PeerCount)
			if info.PeerCount >= 50 {
				peerColor = tcell.ColorGreen
			} else if info.PeerCount >= 10 {
				peerColor = tcell.ColorYellow
			} else {
				peerColor = tcell.ColorRed
			}
		} else {
			peerText = "-"
			peerColor = tcell.ColorGray
		}
		d.setConsensusCell(tableRow, col, peerText, peerColor)
		col++

		// Epoch with arrow notation when behind
		if info.IsConnected {
			if info.FinalizedEpoch == info.CurrentEpoch {
				epochText := fmt.Sprintf("%d ✓", info.CurrentEpoch)
				d.setConsensusCell(tableRow, col, epochText, tcell.ColorWhite)
			} else {
				epochLag := info.CurrentEpoch - info.FinalizedEpoch
				epochText := fmt.Sprintf("%d", info.CurrentEpoch)
				d.setConsensusCellWithColoredArrow(tableRow, col, epochText, true, epochLag, tcell.ColorWhite, 2, 3)
			}
		} else {
			d.setConsensusCell(tableRow, col, "-", tcell.ColorGray)
		}
		col++

		// Node version (if enabled)
		if d.showVersions {
			var versionText string
			if info.IsConnected && info.NodeVersion != "" {
				// Extract just the client/version part (e.g., "Prysm/v4.0.8" from full version string)
				parts := strings.Split(info.NodeVersion, " ")
				if len(parts) > 0 {
					versionText = parts[0]
				} else {
					versionText = info.NodeVersion
				}
			} else {
				versionText = "-"
			}
			d.setConsensusCell(tableRow, col, versionText, tcell.ColorWhite)
			col++
		}

		// Fork version (last column)
		var forkText string
		if info.IsConnected && info.CurrentFork != "" {
			forkText = info.CurrentFork
		} else {
			forkText = "-"
		}
		d.setConsensusCell(tableRow, col, forkText, tcell.ColorWhite)
	}
}

func (d *Display) updateExecutionTable(infos []*execution.ExecutionNodeInfo) {
	if infos == nil {
		infos = []*execution.ExecutionNodeInfo{}
	}

	// Ensure we have enough rows in the table
	currentRows := d.executionTable.GetRowCount()
	neededRows := len(infos) + 1 // +1 for header

	// Add rows if needed
	columnCount := len(d.getExecutionHeaders())
	for i := currentRows; i < neededRows; i++ {
		for j := 0; j < columnCount; j++ {
			d.executionTable.SetCell(i, j, tview.NewTableCell(""))
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
		d.setExecutionCell(tableRow, col, info.Name, tcell.ColorWhite)
		col++

		// Port
		port := parsePortFromEndpoint(info.Endpoint)
		d.setExecutionCell(tableRow, col, port, tcell.ColorWhite)
		col++

		// Status with symbol
		status, statusColor, statusSymbol := d.getExecutionStatusInfo(info)
		statusText := fmt.Sprintf("%s %s", statusSymbol, status)
		d.setExecutionCell(tableRow, col, statusText, statusColor)
		col++

		// Block number with sync progress
		if info.IsConnected {
			blockText := fmt.Sprintf("%d", info.CurrentBlock)
			if info.IsSyncing && info.HighestBlock > info.CurrentBlock {
				blocksBehind := info.HighestBlock - info.CurrentBlock
				d.setExecutionCellWithColoredArrow(tableRow, col, blockText, true, blocksBehind, tcell.ColorWhite, 100, 1000)
			} else {
				d.setExecutionCell(tableRow, col, blockText, tcell.ColorWhite)
			}
		} else {
			d.setExecutionCell(tableRow, col, "-", tcell.ColorGray)
		}
		col++

		// Peers with color
		var peerText string
		var peerColor tcell.Color
		if info.IsConnected && info.PeerCount > 0 {
			peerText = fmt.Sprintf("%d", info.PeerCount)
			if info.PeerCount >= 25 {
				peerColor = tcell.ColorGreen
			} else if info.PeerCount >= 10 {
				peerColor = tcell.ColorYellow
			} else {
				peerColor = tcell.ColorRed
			}
		} else {
			peerText = "-"
			peerColor = tcell.ColorGray
		}
		d.setExecutionCell(tableRow, col, peerText, peerColor)
		col++

		// Gas price
		if info.IsConnected && info.GasPrice != nil {
			gasPrice := new(big.Int).Div(info.GasPrice, big.NewInt(1e9)) // Convert to gwei
			gasPriceText := fmt.Sprintf("%d gwei", gasPrice.Int64())
			d.setExecutionCell(tableRow, col, gasPriceText, tcell.ColorWhite)
		} else {
			d.setExecutionCell(tableRow, col, "-", tcell.ColorGray)
		}
		col++

		// Chain ID
		if info.IsConnected && info.ChainID != nil {
			chainIDText := info.ChainID.String()
			d.setExecutionCell(tableRow, col, chainIDText, tcell.ColorWhite)
		} else {
			d.setExecutionCell(tableRow, col, "-", tcell.ColorGray)
		}
		col++

		// Node version (if enabled)
		if d.showVersions {
			var versionText string
			if info.IsConnected && info.NodeVersion != "" {
				versionText = info.NodeVersion
			} else {
				versionText = "-"
			}
			d.setExecutionCell(tableRow, col, versionText, tcell.ColorWhite)
		}
	}
}

func (d *Display) setConsensusCell(row, col int, text string, color tcell.Color) {
	d.setCell(d.consensusTable, row, col, text, color)
}

func (d *Display) setExecutionCell(row, col int, text string, color tcell.Color) {
	d.setCell(d.executionTable, row, col, text, color)
}

// Removed - no longer using individual validator tables

func (d *Display) setCell(table *tview.Table, row, col int, text string, color tcell.Color) {
	// Bounds check
	if row < 0 || col < 0 {
		return
	}

	// Add padding to cell content
	// Add extra space at the beginning for first column to align with section headers
	var paddedText string
	if col == 0 {
		paddedText = "  " + text + " "
	} else {
		paddedText = " " + text + " "
	}

	cell := table.GetCell(row, col)
	if cell == nil {
		cell = tview.NewTableCell(paddedText).
			SetTextColor(color).
			SetAlign(tview.AlignLeft)
		table.SetCell(row, col, cell)
	} else {
		cell.SetText(paddedText).SetTextColor(color)
	}
}

func (d *Display) setConsensusCellWithColoredArrow(row, col int, baseText string, hasArrow bool, arrowValue uint64, baseColor tcell.Color, thresholdYellow, thresholdRed uint64) {
	d.setCellWithColoredArrow(d.consensusTable, row, col, baseText, hasArrow, arrowValue, baseColor, thresholdYellow, thresholdRed)
}

func (d *Display) setExecutionCellWithColoredArrow(row, col int, baseText string, hasArrow bool, arrowValue uint64, baseColor tcell.Color, thresholdYellow, thresholdRed uint64) {
	d.setCellWithColoredArrow(d.executionTable, row, col, baseText, hasArrow, arrowValue, baseColor, thresholdYellow, thresholdRed)
}

func (d *Display) setCellWithColoredArrow(table *tview.Table, row, col int, baseText string, hasArrow bool, arrowValue uint64, baseColor tcell.Color, thresholdYellow, thresholdRed uint64) {
	if !hasArrow {
		d.setCell(table, row, col, baseText, baseColor)
		return
	}

	// Format text with arrow
	text := fmt.Sprintf("%s ↓%d", baseText, arrowValue)

	// Determine color based on value
	var cellColor tcell.Color
	if arrowValue >= thresholdRed {
		cellColor = tcell.ColorRed
	} else if arrowValue >= thresholdYellow {
		cellColor = tcell.ColorYellow
	} else {
		cellColor = baseColor
	}

	d.setCell(table, row, col, text, cellColor)
}

func (d *Display) getStatusInfo(info *consensus.ConsensusNodeInfo) (string, tcell.Color, string) {
	if info == nil || !info.IsConnected {
		return "Offline", tcell.ColorRed, StatusSymbolOffline
	}
	if info.IsSyncing {
		return "Syncing", tcell.ColorYellow, StatusSymbolSyncing
	}
	if info.IsOptimistic {
		return "Optimistic", tcell.ColorOrange, StatusSymbolOptimistic
	}
	return "Synced", tcell.ColorGreen, StatusSymbolSynced
}

func (d *Display) getExecutionStatusInfo(info *execution.ExecutionNodeInfo) (string, tcell.Color, string) {
	if info == nil || !info.IsConnected {
		return "Offline", tcell.ColorRed, StatusSymbolOffline
	}
	if info.IsSyncing {
		syncPercent := fmt.Sprintf("%.1f%%", info.SyncProgress)
		return fmt.Sprintf("Syncing %s", syncPercent), tcell.ColorYellow, StatusSymbolSyncing
	}
	return "Synced", tcell.ColorGreen, StatusSymbolSynced
}

func (d *Display) formatDuration(duration time.Duration) string {
	if duration < 0 {
		return "0s"
	}

	seconds := int(duration.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

func (d *Display) updateConsensusHeader() {
	headerText := "  ● Consensus Clients"
	if d.nextSlotTime > 0 {
		headerText = fmt.Sprintf("  ● Consensus Clients - Next slot in: %s", d.formatDuration(d.nextSlotTime))
	}
	d.consensusHeader.SetText(headerText)
}

func (d *Display) updateHelpText() {
	// Calculate time until next refresh
	timeLeft := time.Until(d.nextRefresh)
	if timeLeft < 0 {
		timeLeft = 0
	}

	var logHelp string
	if d.showLogs {
		clientName := "[none]"
		if len(d.clientNames) > 0 && d.selectedLogClient < len(d.clientNames) {
			clientName = d.clientNames[d.selectedLogClient]
		}
		logHelp = fmt.Sprintf(" | L:Hide | j/k:Nav | g/G:First/Last | Logs:%s", clientName)
	} else {
		logHelp = " | L:Show Logs"
	}

	versionsHelp := " | v:Show Versions"
	if d.showVersions {
		versionsHelp = " | v:Hide Versions"
	}

	helpText := fmt.Sprintf("  q:Quit | r:Refresh%s%s | Next: %ds",
		versionsHelp, logHelp, int(timeLeft.Seconds()))
	d.help.SetText(helpText)
}

func (d *Display) updateLogView() {
	if !d.showLogs || len(d.clientNames) == 0 {
		return
	}

	clientName := d.clientNames[d.selectedLogClient]

	// Update title with current client
	d.logView.SetTitle(fmt.Sprintf(" Logs - %s ", clientName))

	// Always read fresh logs from file (no caching)
	logs, _ := d.logReader.ReadLogs(clientName)

	// Display logs as-is
	logText := strings.Join(logs, "\n")
	d.logView.SetText(logText)
}

func (d *Display) countdownLoop() {
	defer d.countdownTicker.Stop()

	for {
		select {
		case <-d.countdownTicker.C:
			d.app.QueueUpdateDraw(func() {
				// Decrement next slot time
				if d.nextSlotTime > 0 {
					d.nextSlotTime -= time.Second
					if d.nextSlotTime < 0 {
						d.nextSlotTime = 0
					}
				}

				d.updateConsensusHeader()
				d.updateHelpText()
			})
		}
	}
}

func (d *Display) animationLoop() {
	defer func() {
		if d.animationTicker != nil {
			d.animationTicker.Stop()
		}
	}()

	for {
		select {
		case <-d.animationTicker.C:
			d.app.QueueUpdateDraw(func() {
				d.animationFrame = (d.animationFrame + 1) % len(titleAnimationFrames)
				d.title.SetText(titleAnimationFrames[d.animationFrame])
			})
		}
	}
}

func (d *Display) logUpdateLoop() {
	if d.logUpdateTicker == nil {
		return
	}

	for range d.logUpdateTicker.C {
		// Only update if logs are visible
		if d.showLogs {
			d.app.QueueUpdateDraw(func() {
				d.updateLogView()
			})
		}
	}
}

func parsePortFromEndpoint(endpoint string) string {
	if endpoint == "" {
		return "-"
	}

	// Try to parse as URL
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}

	// If port is explicitly specified, return just the port
	if u.Port() != "" {
		return u.Port()
	}

	// If no explicit port, return the full URL
	return endpoint
}
