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
	mainView          *tview.Flex
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
	clientCards       []*tview.TextView
}

func NewDisplay(monitor *Monitor) *Display {
	return &Display{
		app:               tview.NewApplication(),
		mainView:          tview.NewFlex(),
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
		clientCards:       []*tview.TextView{},
	}
}

func (d *Display) Run() error {
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

// createClientCard creates a card view for a single client
func (d *Display) createClientCard(info *consensus.ConsensusNodeInfo) *tview.TextView {
	card := tview.NewTextView().
		SetDynamicColors(true)
	
	card.SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetTitle(fmt.Sprintf(" %s ", info.Name))

	// Update card content
	d.updateClientCard(card, info)

	return card
}

// updateClientCard updates the content of a client card
func (d *Display) updateClientCard(card *tview.TextView, info *consensus.ConsensusNodeInfo) {
	var status string
	var statusColor string
	var statusSymbol string

	if !info.IsConnected {
		status = "Offline"
		statusColor = "red"
		statusSymbol = StatusSymbolOffline
	} else if info.IsSyncing {
		status = "Syncing"
		statusColor = "yellow"
		statusSymbol = StatusSymbolSyncing
	} else if info.IsOptimistic {
		status = "Optimistic"
		statusColor = "orange"
		statusSymbol = StatusSymbolOptimistic
	} else {
		status = "Synced"
		statusColor = "green"
		statusSymbol = StatusSymbolSynced
	}

	// Build card content
	var content strings.Builder

	// Status line
	content.WriteString(fmt.Sprintf("[%s]%s %s[white]\n", statusColor, statusSymbol, status))

	if info.IsConnected {
		// Peer count
		peerColor := "green"
		if info.PeerCount < 10 {
			peerColor = "red"
		} else if info.PeerCount < 50 {
			peerColor = "yellow"
		}
		content.WriteString(fmt.Sprintf("[%s]Peers: %d[white]\n", peerColor, info.PeerCount))

		// Slot info
		content.WriteString(fmt.Sprintf("Slot: %d/%d\n", info.CurrentSlot, info.HeadSlot))

		// Sync info
		if info.SyncDistance > 0 {
			// Show progress bar for syncing
			progress := float64(info.HeadSlot) / float64(info.CurrentSlot)
			barWidth := 10
			filled := int(progress * float64(barWidth))
			progressBar := strings.Repeat("█", filled) + strings.Repeat("·", barWidth-filled)
			content.WriteString(fmt.Sprintf("Sync: %d %s\n", info.SyncDistance, progressBar))
		} else {
			content.WriteString(fmt.Sprintf("Sync: 0 · Next: %s\n", d.formatDuration(info.TimeToNextSlot)))
		}

		// Epoch info
		if info.FinalizedEpoch == info.CurrentEpoch {
			content.WriteString(fmt.Sprintf("Epoch: %d [green]✓%d[white]", info.CurrentEpoch, info.FinalizedEpoch))
		} else {
			content.WriteString(fmt.Sprintf("Epoch: %d [yellow]✓%d[white]", info.CurrentEpoch, info.FinalizedEpoch))
		}
	} else {
		// Disconnected state
		content.WriteString("[gray]Peers: -\n")
		content.WriteString("Slot: -\n")
		content.WriteString("Sync: -\n")
		content.WriteString("Epoch: -[white]")
	}

	card.SetText(content.String())
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
		// Split view: cards and logs
		mainArea := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(d.mainView, 0, 7, true).   // 70% for cards
			AddItem(d.logView, 0, 3, false) // 30% for logs

		flex.AddItem(mainArea, 0, 1, true)
	} else {
		// Cards only
		flex.AddItem(d.mainView, 0, 1, true)
	}

	flex.AddItem(d.help, 1, 0, false)

	d.app.SetRoot(flex, true).EnableMouse(false)

	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q', 'Q':
			d.app.Stop()
			return nil
		case 'r', 'R':
			go d.updateCards(d.monitor.GetNodeInfos())
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
	d.updateCards(d.monitor.GetNodeInfos())

	// Listen for updates
	for infos := range d.monitor.Updates() {
		d.updateCards(infos)
		// Reset the next refresh time
		d.nextRefresh = time.Now().Add(d.refreshInterval)

		// Update logs if visible
		if d.showLogs {
			d.updateLogView()
		}
	}
}

func (d *Display) updateCards(infos []*consensus.ConsensusNodeInfo) {
	// Validate inputs
	if d == nil || d.app == nil {
		return
	}
	if infos == nil {
		infos = []*consensus.ConsensusNodeInfo{} // Empty slice to prevent crashes
	}

	d.app.QueueUpdateDraw(func() {
		// Clear existing cards
		d.mainView.Clear()
		d.clientCards = d.clientCards[:0]

		// Determine layout based on terminal width
		_, _, width, _ := d.mainView.GetInnerRect()
		cardsPerRow := 1
		if width > 100 {
			cardsPerRow = 2
		}
		if width > 150 {
			cardsPerRow = 3
		}

		// Create grid layout
		grid := tview.NewGrid()
		rows := (len(infos) + cardsPerRow - 1) / cardsPerRow

		// Set up grid dimensions - all rows and columns have equal size (0 means flexible)
		rowSizes := make([]int, rows)
		colSizes := make([]int, cardsPerRow)
		grid.SetRows(rowSizes...)
		grid.SetColumns(colSizes...)

		// Add cards to grid
		for i, info := range infos {
			if info == nil {
				continue
			}

			card := d.createClientCard(info)
			d.clientCards = append(d.clientCards, card)

			row := i / cardsPerRow
			col := i % cardsPerRow
			grid.AddItem(card, row, col, 1, 1, 0, 0, false)
		}

		// Add grid to main view
		d.mainView.AddItem(grid, 0, 1, false)
	})
}


func (d *Display) getStatus(info *consensus.ConsensusNodeInfo) (string, tcell.Color) {
	if info == nil {
		return "Unknown", tcell.ColorGray
	}
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
