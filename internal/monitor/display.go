package monitor

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/watcheth/watcheth/internal/beacon"
)

type Display struct {
	app     *tview.Application
	table   *tview.Table
	monitor *Monitor
}

func NewDisplay(monitor *Monitor) *Display {
	return &Display{
		app:     tview.NewApplication(),
		table:   tview.NewTable(),
		monitor: monitor,
	}
}

func (d *Display) Run() error {
	d.setupTable()
	d.setupLayout()

	go d.updateLoop()

	return d.app.Run()
}

func (d *Display) setupTable() {
	d.table.SetBorders(true).
		SetFixed(1, 0).
		SetSelectable(false, false)

	headers := []string{
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

	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		d.table.SetCell(0, col, cell)
	}
}

func (d *Display) setupLayout() {
	title := tview.NewTextView().
		SetText("WatchETH - Ethereum Beacon Node Monitor").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorGreen)

	help := tview.NewTextView().
		SetText("Press 'q' to quit | Press 'r' to refresh").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorBlack)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(d.table, 0, 1, true).
		AddItem(help, 1, 0, false)

	d.app.SetRoot(flex, true).EnableMouse(false)

	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q', 'Q':
			d.app.Stop()
			return nil
		case 'r', 'R':
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
	}
}

func (d *Display) updateTable(infos []*beacon.BeaconNodeInfo) {
	d.app.QueueUpdateDraw(func() {
		// Ensure we have enough rows in the table
		currentRows := d.table.GetRowCount()
		neededRows := len(infos) + 1 // +1 for header

		// Add empty rows if needed
		for i := currentRows; i < neededRows; i++ {
			for j := 0; j < 13; j++ { // Increased to 13 columns
				d.table.SetCell(i, j, tview.NewTableCell(""))
			}
		}

		for row, info := range infos {
			tableRow := row + 1

			d.setCell(tableRow, 0, info.Name, tcell.ColorWhite)

			status, statusColor := d.getStatus(info)
			d.setCell(tableRow, 1, status, statusColor)

			// Peer count with color coding
			peerColor := tcell.ColorGreen
			if info.PeerCount < 10 {
				peerColor = tcell.ColorRed
			} else if info.PeerCount < 50 {
				peerColor = tcell.ColorYellow
			}
			if info.IsConnected && info.PeerCount > 0 {
				d.setCell(tableRow, 2, fmt.Sprintf("%d", info.PeerCount), peerColor)
			} else {
				d.setCell(tableRow, 2, "-", tcell.ColorBlack)
			}

			// Node version
			if info.NodeVersion != "" {
				d.setCell(tableRow, 3, info.NodeVersion, tcell.ColorWhite)
			} else {
				d.setCell(tableRow, 3, "-", tcell.ColorBlack)
			}

			// Fork
			if info.CurrentFork != "" {
				d.setCell(tableRow, 4, info.CurrentFork, tcell.ColorWhite)
			} else {
				d.setCell(tableRow, 4, "-", tcell.ColorBlack)
			}

			d.setCell(tableRow, 5, fmt.Sprintf("%d", info.CurrentSlot), tcell.ColorWhite)
			d.setCell(tableRow, 6, fmt.Sprintf("%d", info.HeadSlot), tcell.ColorWhite)

			syncDistance := fmt.Sprintf("%d", info.SyncDistance)
			syncColor := tcell.ColorGreen
			if info.SyncDistance > 0 {
				syncColor = tcell.ColorYellow
			}
			if info.SyncDistance > 100 {
				syncColor = tcell.ColorRed
			}
			d.setCell(tableRow, 7, syncDistance, syncColor)

			d.setCell(tableRow, 8, fmt.Sprintf("%d", info.CurrentEpoch), tcell.ColorWhite)
			d.setCell(tableRow, 9, fmt.Sprintf("%d", info.JustifiedEpoch), tcell.ColorWhite)
			d.setCell(tableRow, 10, fmt.Sprintf("%d", info.FinalizedEpoch), tcell.ColorWhite)

			if info.IsConnected {
				d.setCell(tableRow, 11, d.formatDuration(info.TimeToNextSlot), tcell.ColorWhite)
				d.setCell(tableRow, 12, d.formatDuration(info.TimeToNextEpoch), tcell.ColorWhite)
			} else {
				d.setCell(tableRow, 11, "-", tcell.ColorBlack)
				d.setCell(tableRow, 12, "-", tcell.ColorBlack)
			}
		}
	})
}

func (d *Display) setCell(row, col int, text string, color tcell.Color) {
	cell := d.table.GetCell(row, col)
	if cell == nil {
		cell = tview.NewTableCell(text).
			SetTextColor(color).
			SetAlign(tview.AlignCenter)
		d.table.SetCell(row, col, cell)
	} else {
		cell.SetText(text).SetTextColor(color)
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
