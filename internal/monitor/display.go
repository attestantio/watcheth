package monitor

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/watcheth/watcheth/internal/config"
	"github.com/watcheth/watcheth/internal/consensus"
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
	"┌─── watcheth ───┐\n│     /\\_/\\      │\n│    ( o.o )     │\n│     > ^ <      │\n└consensus monitor┘",
	"┌─── watcheth ───┐\n│     /\\_/\\      │\n│    ( o.o )     │\n│     > ^ <      │\n└consensus monitor┘",
	"┌─── watcheth ───┐\n│     /\\_/\\      │\n│    ( -.- )     │\n│     > ^ <      │\n└consensus monitor┘",
	"┌─── watcheth ───┐\n│     /\\_/\\      │\n│    ( o.o )     │\n│     > ^ <      │\n└consensus monitor┘",
}

type Display struct {
	app               *tview.Application
	table             *tview.Table
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
	showLogs          bool
	selectedLogClient int
	clientNames       []string
}

func NewDisplay(monitor *Monitor) *Display {
	return &Display{
		app:               tview.NewApplication(),
		table:             tview.NewTable(),
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

// SetupLogPaths configures log paths for each client
func (d *Display) SetupLogPaths(clientConfigs []config.ClientConfig) {
	d.clientNames = make([]string, len(clientConfigs))
	for i, cfg := range clientConfigs {
		d.clientNames[i] = cfg.Name
		if cfg.LogPath != "" || cfg.GetLogPath() != "" {
			d.logReader.SetLogPath(cfg.Name, cfg.GetLogPath())
		}
	}
}

func (d *Display) setupTable() {
	d.table.Clear()
	d.table.SetBorders(true).
		SetFixed(1, 0).
		SetSelectable(false, false)

	// Define headers for compact view
	headers := []string{
		"Client",
		"Status",
		"Slot",
		"Sync",
		"Peers",
		"Next",
		"Epoch",
	}

	// Set up header row with padding
	for col, header := range headers {
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

	// Setup log view
	d.logView.SetBorder(true).
		SetTitle(" Logs ").
		SetTitleAlign(tview.AlignLeft)

	d.updateLayout()
}

func (d *Display) updateLayout() {
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(d.title, 5, 0, false). // Cat face animation
		AddItem(nil, 1, 0, false)      // Empty space

	if d.showLogs {
		// Split view: table and logs
		mainArea := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(d.table, 0, 7, true).   // 70% for table
			AddItem(d.logView, 0, 3, false) // 30% for logs

		flex.AddItem(mainArea, 0, 1, true)
	} else {
		// Table only
		flex.AddItem(d.table, 0, 1, true)
	}

	flex.AddItem(d.help, 1, 0, false)

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
		return event
	})
}

func (d *Display) updateLogView() {
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

func (d *Display) updateLoop() {
	// Initial update
	d.updateTable(d.monitor.GetNodeInfos())

	// Listen for updates
	for infos := range d.monitor.Updates() {
		d.updateTable(infos)
		// Reset the next refresh time
		d.nextRefresh = time.Now().Add(d.refreshInterval)

		// Update logs if visible
		if d.showLogs {
			d.updateLogView()
		}
	}
}

func (d *Display) updateTable(infos []*consensus.ConsensusNodeInfo) {
	// Validate inputs
	if d == nil || d.app == nil || d.table == nil {
		return
	}
	if infos == nil {
		infos = []*consensus.ConsensusNodeInfo{} // Empty slice to prevent crashes
	}

	d.app.QueueUpdateDraw(func() {
		// Update table rows
		for row, info := range infos {
			if info == nil {
				continue
			}

			tableRow := row + 1 // +1 for header
			col := 0

			// Client name
			d.setCell(tableRow, col, info.Name, tcell.ColorWhite)
			col++

			// Status with symbol
			status, statusColor, statusSymbol := d.getStatusInfo(info)
			statusText := fmt.Sprintf("%s %s", statusSymbol, status)
			d.setCell(tableRow, col, statusText, statusColor)
			col++

			// Slot - show current/head when syncing, just current when synced
			var slotText string
			if info.IsConnected {
				if info.SyncDistance > 0 {
					slotText = fmt.Sprintf("%d/%d", info.CurrentSlot, info.HeadSlot)
				} else {
					slotText = fmt.Sprintf("%d", info.CurrentSlot)
				}
			} else {
				slotText = "-"
			}
			d.setCell(tableRow, col, slotText, tcell.ColorWhite)
			col++

			// Sync distance with color
			var syncText string
			var syncColor tcell.Color
			if info.IsConnected {
				syncText = fmt.Sprintf("%d", info.SyncDistance)
				if info.SyncDistance == 0 {
					syncColor = tcell.ColorGreen
				} else if info.SyncDistance < 100 {
					syncColor = tcell.ColorYellow
				} else {
					syncColor = tcell.ColorRed
				}
			} else {
				syncText = "-"
				syncColor = tcell.ColorGray
			}
			d.setCell(tableRow, col, syncText, syncColor)
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
			d.setCell(tableRow, col, peerText, peerColor)
			col++

			// Next slot time
			var nextText string
			if info.IsConnected && info.TimeToNextSlot > 0 {
				nextText = d.formatDuration(info.TimeToNextSlot)
			} else {
				nextText = "-"
			}
			d.setCell(tableRow, col, nextText, tcell.ColorWhite)
			col++

			// Epoch with finalized checkmark
			var epochText string
			if info.IsConnected {
				if info.FinalizedEpoch == info.CurrentEpoch {
					epochText = fmt.Sprintf("%d ✓%d", info.CurrentEpoch, info.FinalizedEpoch)
				} else {
					epochText = fmt.Sprintf("%d ✓%d", info.CurrentEpoch, info.FinalizedEpoch)
				}
			} else {
				epochText = "-"
			}
			d.setCell(tableRow, col, epochText, tcell.ColorWhite)
		}
	})
}

func (d *Display) setCell(row, col int, text string, color tcell.Color) {
	// Bounds check
	if row < 0 || col < 0 {
		return
	}

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

	helpText := fmt.Sprintf("q:Quit | r:Refresh%s | Next: %ds",
		logHelp, int(timeLeft.Seconds()))
	d.help.SetText(helpText)
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
