package monitor

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/watcheth/watcheth/internal/beacon"
)

type ViewMode int

const (
	ViewCompact ViewMode = iota
	ViewNetwork
	ViewConsensus
	ViewFull
)

// Animation frames for the title
var titleAnimationFrames = []string{
	"┌─── watcheth ───┐\n│     /\\_/\\      │\n│    ( o.o )     │\n│     > ^ <      │\n└ beacon monitor ┘",
	"┌─── watcheth ───┐\n│     /\\_/\\      │\n│    ( o.o )     │\n│     > ^ <      │\n└ beacon monitor ┘",
	"┌─── watcheth ───┐\n│     /\\_/\\      │\n│    ( -.- )     │\n│     > ^ <      │\n└ beacon monitor ┘",
	"┌─── watcheth ───┐\n│     /\\_/\\      │\n│    ( o.o )     │\n│     > ^ <      │\n└ beacon monitor ┘",
}

type Display struct {
	app             *tview.Application
	table           *tview.Table
	monitor         *Monitor
	viewMode        ViewMode
	help            *tview.TextView
	refreshInterval time.Duration
	nextRefresh     time.Time
	countdownTicker *time.Ticker
	title           *tview.TextView
	animationTicker *time.Ticker
	animationFrame  int
}

func NewDisplay(monitor *Monitor) *Display {
	return &Display{
		app:             tview.NewApplication(),
		table:           tview.NewTable(),
		monitor:         monitor,
		viewMode:        ViewCompact,
		help:            tview.NewTextView(),
		title:           tview.NewTextView(),
		refreshInterval: monitor.GetRefreshInterval(),
		nextRefresh:     time.Now().Add(monitor.GetRefreshInterval()),
		animationFrame:  0,
	}
}

func (d *Display) Run() error {
	d.setupTable()
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

func (d *Display) setupTable() {
	d.table.Clear()
	d.table.SetBorders(true).
		SetFixed(1, 0).
		SetSelectable(false, false)

	var headers []string
	switch d.viewMode {
	case ViewCompact:
		headers = []string{
			"Client",
			"Status",
			"Peers",
			"Current Slot",
			"Head Slot",
			"Sync",
			"Next Slot",
		}
	case ViewNetwork:
		headers = []string{
			"Client",
			"Status",
			"Peers",
			"Version",
			"Fork",
		}
	case ViewConsensus:
		headers = []string{
			"Client",
			"Current Epoch",
			"Justified",
			"Finalized",
			"Next Epoch In",
		}
	case ViewFull:
		headers = []string{
			"Client",
			"Status",
			"Peers",
			"Version",
			"Fork",
			"Current Slot",
			"Head Slot",
			"Sync Distance",
			"Current Epoch",
			"Justified Epoch",
			"Finalized Epoch",
			"Next Slot In",
			"Next Epoch In",
		}
	}

	for col, header := range headers {
		// Add padding to headers
		paddedHeader := " " + header + " "
		cell := tview.NewTableCell(paddedHeader).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		d.table.SetCell(0, col, cell)
	}
}

func (d *Display) setupLayout() {
	// Initialize title with first animation frame
	d.title.SetText(titleAnimationFrames[0]).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorGreen)

	d.updateHelpText()
	d.help.SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorBlack)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(d.title, 5, 0, false). // Changed to 5 lines for the cat face animation
		AddItem(nil, 1, 0, false).     // Empty space between title and table
		AddItem(d.table, 0, 1, true).
		AddItem(d.help, 1, 0, false)

	d.app.SetRoot(flex, true).EnableMouse(false)

	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q', 'Q':
			d.app.Stop()
			return nil
		case 'r', 'R':
			go d.updateTable(d.monitor.GetNodeInfos())
			// Reset the next refresh time when manually refreshing
			d.nextRefresh = time.Now().Add(d.refreshInterval)
			return nil
		case '1':
			d.viewMode = ViewCompact
			d.updateHelpText()
			d.setupTable()
			go d.updateTable(d.monitor.GetNodeInfos())
			return nil
		case '2':
			d.viewMode = ViewNetwork
			d.updateHelpText()
			d.setupTable()
			go d.updateTable(d.monitor.GetNodeInfos())
			return nil
		case '3':
			d.viewMode = ViewConsensus
			d.updateHelpText()
			d.setupTable()
			go d.updateTable(d.monitor.GetNodeInfos())
			return nil
		case '4':
			d.viewMode = ViewFull
			d.updateHelpText()
			d.setupTable()
			go d.updateTable(d.monitor.GetNodeInfos())
			return nil
		}
		return event
	})
}

func (d *Display) updateLoop() {
	// Initial update
	d.updateTable(d.monitor.GetNodeInfos())

	// Listen for updates
	for infos := range d.monitor.Updates() {
		d.updateTable(infos)
		// Reset the next refresh time
		d.nextRefresh = time.Now().Add(d.refreshInterval)
	}
}

func (d *Display) updateTable(infos []*beacon.BeaconNodeInfo) {
	d.app.QueueUpdateDraw(func() {
		// Determine column count based on view mode
		var columnCount int
		switch d.viewMode {
		case ViewCompact:
			columnCount = 7
		case ViewNetwork:
			columnCount = 5
		case ViewConsensus:
			columnCount = 5
		case ViewFull:
			columnCount = 13
		}

		// Ensure we have enough rows in the table
		currentRows := d.table.GetRowCount()
		neededRows := len(infos) + 1 // +1 for header

		// Add rows if needed
		for i := currentRows; i < neededRows; i++ {
			for j := 0; j < columnCount; j++ {
				d.table.SetCell(i, j, tview.NewTableCell(""))
			}
		}

		// Update each row with node info
		for row, info := range infos {
			tableRow := row + 1

			switch d.viewMode {
			case ViewCompact:
				d.updateCompactView(tableRow, info)
			case ViewNetwork:
				d.updateNetworkView(tableRow, info)
			case ViewConsensus:
				d.updateConsensusView(tableRow, info)
			case ViewFull:
				d.updateFullView(tableRow, info)
			}
		}
	})
}

func (d *Display) setCell(row, col int, text string, color tcell.Color) {
	// Add padding to cell content
	paddedText := " " + text + " "
	cell := d.table.GetCell(row, col)
	if cell == nil {
		cell = tview.NewTableCell(paddedText).
			SetTextColor(color).
			SetAlign(tview.AlignLeft)
		d.table.SetCell(row, col, cell)
	} else {
		cell.SetText(paddedText).SetTextColor(color)
	}
}

func (d *Display) getStatus(info *beacon.BeaconNodeInfo) (string, tcell.Color) {
	if !info.IsConnected {
		return "Offline", tcell.ColorRed
	}
	if info.IsSyncing {
		return "Syncing", tcell.ColorYellow
	}
	if info.IsOptimistic {
		return "Optimistic", tcell.ColorOrange
	}
	return "Synced", tcell.ColorGreen
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

func (d *Display) updateHelpText() {
	var viewName string
	switch d.viewMode {
	case ViewCompact:
		viewName = "Compact"
	case ViewNetwork:
		viewName = "Network"
	case ViewConsensus:
		viewName = "Consensus"
	case ViewFull:
		viewName = "Full"
	}

	// Calculate time until next refresh
	timeLeft := time.Until(d.nextRefresh)
	if timeLeft < 0 {
		timeLeft = 0
	}

	helpText := fmt.Sprintf("[%s View] q:Quit | r:Refresh | 1:Compact | 2:Network | 3:Consensus | 4:Full | Next refresh: %ds",
		viewName, int(timeLeft.Seconds()))
	d.help.SetText(helpText)
}

func (d *Display) updateCompactView(row int, info *beacon.BeaconNodeInfo) {
	col := 0

	// Client name
	d.setCell(row, col, info.Name, tcell.ColorWhite)
	col++

	// Status
	status, statusColor := d.getStatus(info)
	d.setCell(row, col, status, statusColor)
	col++

	// Peers
	peerColor := tcell.ColorGreen
	if info.PeerCount < 10 {
		peerColor = tcell.ColorRed
	} else if info.PeerCount < 50 {
		peerColor = tcell.ColorYellow
	}
	if info.IsConnected && info.PeerCount > 0 {
		d.setCell(row, col, fmt.Sprintf("%d", info.PeerCount), peerColor)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
	col++

	// Current Slot
	d.setCell(row, col, fmt.Sprintf("%d", info.CurrentSlot), tcell.ColorWhite)
	col++

	// Head Slot
	d.setCell(row, col, fmt.Sprintf("%d", info.HeadSlot), tcell.ColorWhite)
	col++

	// Sync Distance
	syncDistance := fmt.Sprintf("%d", info.SyncDistance)
	syncColor := tcell.ColorGreen
	if info.SyncDistance > 0 {
		syncColor = tcell.ColorYellow
	}
	if info.SyncDistance > 100 {
		syncColor = tcell.ColorRed
	}
	d.setCell(row, col, syncDistance, syncColor)
	col++

	// Next Slot
	if info.IsConnected {
		d.setCell(row, col, d.formatDuration(info.TimeToNextSlot), tcell.ColorWhite)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
}

func (d *Display) updateNetworkView(row int, info *beacon.BeaconNodeInfo) {
	col := 0

	// Client name
	d.setCell(row, col, info.Name, tcell.ColorWhite)
	col++

	// Status
	status, statusColor := d.getStatus(info)
	d.setCell(row, col, status, statusColor)
	col++

	// Peers
	peerColor := tcell.ColorGreen
	if info.PeerCount < 10 {
		peerColor = tcell.ColorRed
	} else if info.PeerCount < 50 {
		peerColor = tcell.ColorYellow
	}
	if info.IsConnected && info.PeerCount > 0 {
		d.setCell(row, col, fmt.Sprintf("%d", info.PeerCount), peerColor)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
	col++

	// Version
	if info.NodeVersion != "" {
		d.setCell(row, col, info.NodeVersion, tcell.ColorWhite)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
	col++

	// Fork
	if info.CurrentFork != "" {
		d.setCell(row, col, info.CurrentFork, tcell.ColorWhite)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
}

func (d *Display) updateConsensusView(row int, info *beacon.BeaconNodeInfo) {
	col := 0

	// Client name
	d.setCell(row, col, info.Name, tcell.ColorWhite)
	col++

	// Current Epoch
	d.setCell(row, col, fmt.Sprintf("%d", info.CurrentEpoch), tcell.ColorWhite)
	col++

	// Justified
	d.setCell(row, col, fmt.Sprintf("%d", info.JustifiedEpoch), tcell.ColorWhite)
	col++

	// Finalized
	d.setCell(row, col, fmt.Sprintf("%d", info.FinalizedEpoch), tcell.ColorWhite)
	col++

	// Next Epoch In
	if info.IsConnected {
		d.setCell(row, col, d.formatDuration(info.TimeToNextEpoch), tcell.ColorWhite)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
}

func (d *Display) updateFullView(row int, info *beacon.BeaconNodeInfo) {
	col := 0

	// Client name
	d.setCell(row, col, info.Name, tcell.ColorWhite)
	col++

	// Status
	status, statusColor := d.getStatus(info)
	d.setCell(row, col, status, statusColor)
	col++

	// Peers
	peerColor := tcell.ColorGreen
	if info.PeerCount < 10 {
		peerColor = tcell.ColorRed
	} else if info.PeerCount < 50 {
		peerColor = tcell.ColorYellow
	}
	if info.IsConnected && info.PeerCount > 0 {
		d.setCell(row, col, fmt.Sprintf("%d", info.PeerCount), peerColor)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
	col++

	// Version
	if info.NodeVersion != "" {
		d.setCell(row, col, info.NodeVersion, tcell.ColorWhite)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
	col++

	// Fork
	if info.CurrentFork != "" {
		d.setCell(row, col, info.CurrentFork, tcell.ColorWhite)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
	col++

	// Current Slot
	d.setCell(row, col, fmt.Sprintf("%d", info.CurrentSlot), tcell.ColorWhite)
	col++

	// Head Slot
	d.setCell(row, col, fmt.Sprintf("%d", info.HeadSlot), tcell.ColorWhite)
	col++

	// Sync Distance
	syncDistance := fmt.Sprintf("%d", info.SyncDistance)
	syncColor := tcell.ColorGreen
	if info.SyncDistance > 0 {
		syncColor = tcell.ColorYellow
	}
	if info.SyncDistance > 100 {
		syncColor = tcell.ColorRed
	}
	d.setCell(row, col, syncDistance, syncColor)
	col++

	// Current Epoch
	d.setCell(row, col, fmt.Sprintf("%d", info.CurrentEpoch), tcell.ColorWhite)
	col++

	// Justified Epoch
	d.setCell(row, col, fmt.Sprintf("%d", info.JustifiedEpoch), tcell.ColorWhite)
	col++

	// Finalized Epoch
	d.setCell(row, col, fmt.Sprintf("%d", info.FinalizedEpoch), tcell.ColorWhite)
	col++

	// Next Slot In
	if info.IsConnected {
		d.setCell(row, col, d.formatDuration(info.TimeToNextSlot), tcell.ColorWhite)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
	col++

	// Next Epoch In
	if info.IsConnected {
		d.setCell(row, col, d.formatDuration(info.TimeToNextEpoch), tcell.ColorWhite)
	} else {
		d.setCell(row, col, "-", tcell.ColorBlack)
	}
}

func (d *Display) countdownLoop() {
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
