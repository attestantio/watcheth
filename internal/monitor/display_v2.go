package monitor

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/watcheth/watcheth/internal/config"
	"github.com/watcheth/watcheth/internal/consensus"
	"github.com/watcheth/watcheth/internal/execution"
)

type DisplayV2 struct {
	app               *tview.Application
	consensusTable    *tview.Table
	executionTable    *tview.Table
	monitor           *MonitorV2
	help              *tview.TextView
	refreshInterval   time.Duration
	nextRefresh       time.Time
	countdownTicker   *time.Ticker
	title             *tview.TextView
	animationTicker   *time.Ticker
	animationFrame    int
	logView           *tview.TextView
	logReader         *LogReader
	showLogs          bool
	selectedLogClient int
	clientNames       []string
	focusedTable      int // 0 = consensus, 1 = execution
}

func NewDisplayV2(monitor *MonitorV2) *DisplayV2 {
	return &DisplayV2{
		app:               tview.NewApplication(),
		consensusTable:    tview.NewTable(),
		executionTable:    tview.NewTable(),
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
		focusedTable:      0,
	}
}

func (d *DisplayV2) Run() error {
	d.setupTables()
	d.setupLayout()

	// Start countdown ticker
	d.countdownTicker = time.NewTicker(time.Second)
	go d.countdownLoop()

	// Start animation ticker
	d.animationTicker = time.NewTicker(500 * time.Millisecond)
	go d.animationLoop()

	go d.updateLoop()

	return d.app.Run()
}

func (d *DisplayV2) SetupLogPaths(clientConfigs []config.ClientConfig) {
	d.clientNames = make([]string, len(clientConfigs))
	for i, cfg := range clientConfigs {
		d.clientNames[i] = cfg.Name
		if cfg.LogPath != "" || cfg.GetLogPath() != "" {
			d.logReader.SetLogPath(cfg.Name, cfg.GetLogPath())
		}
	}
}

func (d *DisplayV2) setupTables() {
	// Setup consensus table
	d.consensusTable.Clear()
	d.consensusTable.SetBorders(true).
		SetFixed(1, 0).
		SetSelectable(false, false)

	// Setup execution table
	d.executionTable.Clear()
	d.executionTable.SetBorders(true).
		SetFixed(1, 0).
		SetSelectable(false, false)

	// Set up header rows
	for col, header := range d.getConsensusHeaders() {
		paddedHeader := " " + header + " "
		cell := tview.NewTableCell(paddedHeader).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		d.consensusTable.SetCell(0, col, cell)
	}

	for col, header := range d.getExecutionHeaders() {
		paddedHeader := " " + header + " "
		cell := tview.NewTableCell(paddedHeader).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		d.executionTable.SetCell(0, col, cell)
	}
}

func (d *DisplayV2) getConsensusHeaders() []string {
	return []string{
		"Client",
		"Status",
		"Slot",
		"Peers",
		"Next In",
		"Epoch/Final",
	}
}

func (d *DisplayV2) getExecutionHeaders() []string {
	return []string{
		"Client",
		"Status",
		"Block",
		"Peers",
		"Gas Price",
		"Chain ID",
	}
}

func (d *DisplayV2) setupLayout() {
	// Initialize title with updated text
	titleFrame := titleAnimationFrames[0]
	titleFrame = strings.Replace(titleFrame, "consensus monitor", "eth node monitor", 1)
	d.title.SetText(titleFrame).
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

func (d *DisplayV2) updateLayout() {
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(d.title, 5, 0, false). // Cat face animation
		AddItem(nil, 1, 0, false)      // Empty space

	// Consensus clients section
	consensusSection := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("[Consensus Clients]").SetTextColor(tcell.ColorGreen), 1, 0, false).
		AddItem(d.consensusTable, 0, 1, d.focusedTable == 0)

	// Execution clients section
	executionSection := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false). // Spacer
		AddItem(tview.NewTextView().SetText("[Execution Clients]").SetTextColor(tcell.ColorGreen), 1, 0, false).
		AddItem(d.executionTable, 0, 1, d.focusedTable == 1)

	tablesArea := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(consensusSection, 0, 1, true).
		AddItem(executionSection, 0, 1, false)

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
		}

		// Tab key to switch between tables
		if event.Key() == tcell.KeyTab {
			d.focusedTable = (d.focusedTable + 1) % 2
			d.updateLayout()
			return nil
		}

		return event
	})
}

func (d *DisplayV2) updateLoop() {
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

func (d *DisplayV2) updateTables(update NodeUpdate) {
	// Validate inputs
	if d == nil || d.app == nil || d.consensusTable == nil || d.executionTable == nil {
		return
	}

	d.app.QueueUpdateDraw(func() {
		// Update consensus table
		d.updateConsensusTable(update.ConsensusInfos)

		// Update execution table
		d.updateExecutionTable(update.ExecutionInfos)
	})
}

func (d *DisplayV2) updateConsensusTable(infos []*consensus.ConsensusNodeInfo) {
	if infos == nil {
		infos = []*consensus.ConsensusNodeInfo{}
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

		// Status with symbol
		status, statusColor, statusSymbol := d.getStatusInfo(info)
		statusText := fmt.Sprintf("%s %s", statusSymbol, status)
		d.setConsensusCell(tableRow, col, statusText, statusColor)
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

		// Next slot time
		var nextText string
		if info.IsConnected && info.TimeToNextSlot > 0 {
			nextText = d.formatDuration(info.TimeToNextSlot)
		} else {
			nextText = "-"
		}
		d.setConsensusCell(tableRow, col, nextText, tcell.ColorWhite)
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
	}
}

func (d *DisplayV2) updateExecutionTable(infos []*execution.ExecutionNodeInfo) {
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
	}
}

func (d *DisplayV2) setConsensusCell(row, col int, text string, color tcell.Color) {
	d.setCell(d.consensusTable, row, col, text, color)
}

func (d *DisplayV2) setExecutionCell(row, col int, text string, color tcell.Color) {
	d.setCell(d.executionTable, row, col, text, color)
}

func (d *DisplayV2) setCell(table *tview.Table, row, col int, text string, color tcell.Color) {
	// Bounds check
	if row < 0 || col < 0 {
		return
	}

	// Add padding to cell content
	paddedText := " " + text + " "

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

func (d *DisplayV2) setConsensusCellWithColoredArrow(row, col int, baseText string, hasArrow bool, arrowValue uint64, baseColor tcell.Color, thresholdYellow, thresholdRed uint64) {
	d.setCellWithColoredArrow(d.consensusTable, row, col, baseText, hasArrow, arrowValue, baseColor, thresholdYellow, thresholdRed)
}

func (d *DisplayV2) setExecutionCellWithColoredArrow(row, col int, baseText string, hasArrow bool, arrowValue uint64, baseColor tcell.Color, thresholdYellow, thresholdRed uint64) {
	d.setCellWithColoredArrow(d.executionTable, row, col, baseText, hasArrow, arrowValue, baseColor, thresholdYellow, thresholdRed)
}

func (d *DisplayV2) setCellWithColoredArrow(table *tview.Table, row, col int, baseText string, hasArrow bool, arrowValue uint64, baseColor tcell.Color, thresholdYellow, thresholdRed uint64) {
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

func (d *DisplayV2) getStatusInfo(info *consensus.ConsensusNodeInfo) (string, tcell.Color, string) {
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

func (d *DisplayV2) getExecutionStatusInfo(info *execution.ExecutionNodeInfo) (string, tcell.Color, string) {
	if info == nil || !info.IsConnected {
		return "Offline", tcell.ColorRed, StatusSymbolOffline
	}
	if info.IsSyncing {
		syncPercent := fmt.Sprintf("%.1f%%", info.SyncProgress)
		return fmt.Sprintf("Syncing %s", syncPercent), tcell.ColorYellow, StatusSymbolSyncing
	}
	return "Synced", tcell.ColorGreen, StatusSymbolSynced
}

func (d *DisplayV2) formatDuration(duration time.Duration) string {
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

func (d *DisplayV2) updateHelpText() {
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

	focusedTableName := "Consensus"
	if d.focusedTable == 1 {
		focusedTableName = "Execution"
	}

	helpText := fmt.Sprintf("q:Quit | r:Refresh | Tab:Switch Table [%s]%s | Next: %ds",
		focusedTableName, logHelp, int(timeLeft.Seconds()))
	d.help.SetText(helpText)
}

func (d *DisplayV2) updateLogView() {
	if !d.showLogs || len(d.clientNames) == 0 {
		return
	}

	clientName := d.clientNames[d.selectedLogClient]

	// Update title with current client
	d.logView.SetTitle(fmt.Sprintf(" Logs - %s ", clientName))

	// Read logs for the selected client
	logs, _ := d.logReader.ReadLogs(clientName)

	// Display logs as-is
	logText := strings.Join(logs, "\n")
	d.logView.SetText(logText)
}

func (d *DisplayV2) countdownLoop() {
	defer d.countdownTicker.Stop()

	for {
		select {
		case <-d.countdownTicker.C:
			d.app.QueueUpdateDraw(func() {
				d.updateHelpText()
			})
		}
	}
}

func (d *DisplayV2) animationLoop() {
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
				titleFrame := titleAnimationFrames[d.animationFrame]
				titleFrame = strings.Replace(titleFrame, "consensus monitor", "eth node monitor", 1)
				d.title.SetText(titleFrame)
			})
		}
	}
}
