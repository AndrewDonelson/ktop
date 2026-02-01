package ui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/nlaak/ktop/internal/config"
	"github.com/nlaak/ktop/internal/metrics"
	"github.com/nlaak/ktop/internal/models"
)

// App represents the main TUI application
type App struct {
	app       *tview.Application
	collector *metrics.Collector
	config    *config.Config
	colors    Colors

	// UI components
	mainFlex   *tview.Flex
	header     *tview.TextView
	summary    *tview.TextView
	nodesTable *tview.Table
	podsTable  *tview.Table
	footer     *tview.TextView
	helpModal  *tview.Modal

	// State
	state     models.AppState
	stateMu   sync.RWMutex
	metrics   *models.ClusterMetrics
	metricsMu sync.RWMutex

	// Control
	ctx        context.Context
	cancel     context.CancelFunc
	refreshNow chan struct{}
}

// NewApp creates a new TUI application
func NewApp(collector *metrics.Collector, cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())

	a := &App{
		app:        tview.NewApplication(),
		collector:  collector,
		config:     cfg,
		colors:     DefaultColors(),
		state:      models.DefaultAppState(),
		ctx:        ctx,
		cancel:     cancel,
		refreshNow: make(chan struct{}, 1),
	}

	a.state.ShowSystem = cfg.AllNamespaces

	a.setupUI()
	a.setupKeybindings()

	return a
}

// setupUI initializes all UI components
func (a *App) setupUI() {
	// Header
	a.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.header.SetBorder(false)

	// Summary bar (cluster totals)
	a.summary = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.summary.SetBorder(false)

	// Nodes table
	a.nodesTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	a.nodesTable.SetBorder(true).
		SetTitle(" NODES ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorWhite)

	// Pods table
	a.podsTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	a.podsTable.SetBorder(true).
		SetTitle(" PODS ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorWhite)

	// Footer
	a.footer = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.footer.SetBorder(false)

	// Help modal
	a.helpModal = tview.NewModal().
		SetText(helpText()).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(_ int, _ string) {
			a.stateMu.Lock()
			a.state.ShowHelp = false
			a.stateMu.Unlock()
			a.app.SetRoot(a.mainFlex, true)
		})

	// Main layout
	a.mainFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header, 1, 0, false).
		AddItem(a.summary, 2, 0, false).
		AddItem(a.nodesTable, 0, 1, true).
		AddItem(a.podsTable, 0, 2, false).
		AddItem(a.footer, 1, 0, false)
}

// setupKeybindings configures keyboard input handling
func (a *App) setupKeybindings() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		a.stateMu.Lock()
		defer a.stateMu.Unlock()

		// Handle help modal first
		if a.state.ShowHelp {
			if event.Key() == tcell.KeyEscape || event.Rune() == '?' || event.Rune() == 'q' {
				a.state.ShowHelp = false
				a.app.SetRoot(a.mainFlex, true)
				return nil
			}
			return event
		}

		switch event.Key() {
		case tcell.KeyEscape:
			if a.state.NamespaceFilter != "" {
				a.state.NamespaceFilter = ""
				return nil
			}
		case tcell.KeyTab:
			// Switch focus between nodes and pods tables
			if a.nodesTable.HasFocus() {
				a.app.SetFocus(a.podsTable)
			} else {
				a.app.SetFocus(a.nodesTable)
			}
			return nil
		}

		switch event.Rune() {
		case 'q', 'Q':
			a.cancel()
			a.app.Stop()
			return nil

		case 'r', 'R':
			// Force refresh
			select {
			case a.refreshNow <- struct{}{}:
			default:
			}
			return nil

		case 's', 'S':
			// Cycle node sort
			a.cycleNodeSort()
			return nil

		case 'p', 'P':
			// Cycle pod sort
			a.cyclePodSort()
			return nil

		case 'f', 'F':
			// Filter namespaces - cycle through available namespaces
			a.cycleNamespaceFilter()
			return nil

		case 'n', 'N':
			// Next namespace filter
			a.cycleNamespaceFilter()
			return nil

		case 't', 'T':
			// Toggle view mode
			a.cycleViewMode()
			return nil

		case 'a', 'A':
			// Toggle system namespaces
			a.state.ShowSystem = !a.state.ShowSystem
			return nil

		case '?':
			// Show help
			a.state.ShowHelp = true
			a.app.SetRoot(a.helpModal, true)
			return nil
		}

		return event
	})
}

// cycleNodeSort cycles through node sort options
func (a *App) cycleNodeSort() {
	switch a.state.NodeSortField {
	case models.SortNodeName:
		a.state.NodeSortField = models.SortNodeCPU
	case models.SortNodeCPU:
		a.state.NodeSortField = models.SortNodeMemory
	case models.SortNodeMemory:
		a.state.NodeSortField = models.SortNodeStatus
	case models.SortNodeStatus:
		a.state.NodeSortField = models.SortNodePods
	default:
		a.state.NodeSortField = models.SortNodeName
		a.state.NodeSortAsc = !a.state.NodeSortAsc
	}
}

// cyclePodSort cycles through pod sort options
func (a *App) cyclePodSort() {
	switch a.state.PodSortField {
	case models.SortPodNamespace:
		a.state.PodSortField = models.SortPodName
	case models.SortPodName:
		a.state.PodSortField = models.SortPodCPU
	case models.SortPodCPU:
		a.state.PodSortField = models.SortPodMemory
	case models.SortPodMemory:
		a.state.PodSortField = models.SortPodNamespace
		a.state.PodSortAsc = !a.state.PodSortAsc
	default:
		a.state.PodSortField = models.SortPodCPU
	}
}

// cycleNamespaceFilter cycles through namespace filters
func (a *App) cycleNamespaceFilter() {
	namespaces := a.collector.GetNamespaces()
	if len(namespaces) == 0 {
		return
	}

	// Find current position
	currentIdx := -1
	for i, ns := range namespaces {
		if ns == a.state.NamespaceFilter {
			currentIdx = i
			break
		}
	}

	// Move to next (or wrap to "all")
	if currentIdx == -1 || currentIdx == len(namespaces)-1 {
		if a.state.NamespaceFilter == "" {
			a.state.NamespaceFilter = namespaces[0]
		} else {
			a.state.NamespaceFilter = ""
		}
	} else {
		a.state.NamespaceFilter = namespaces[currentIdx+1]
	}
}

// cycleViewMode cycles through view modes
func (a *App) cycleViewMode() {
	switch a.state.ViewMode {
	case models.ViewModeSplit:
		a.state.ViewMode = models.ViewModeNodes
	case models.ViewModeNodes:
		a.state.ViewMode = models.ViewModePods
	case models.ViewModePods:
		a.state.ViewMode = models.ViewModeSplit
	}
	a.updateLayout()
}

// updateLayout updates the layout based on view mode
func (a *App) updateLayout() {
	a.mainFlex.Clear()
	a.mainFlex.AddItem(a.header, 1, 0, false)
	a.mainFlex.AddItem(a.summary, 2, 0, false)

	switch a.state.ViewMode {
	case models.ViewModeSplit:
		a.mainFlex.AddItem(a.nodesTable, 0, 1, true)
		a.mainFlex.AddItem(a.podsTable, 0, 2, false)
	case models.ViewModeNodes:
		a.mainFlex.AddItem(a.nodesTable, 0, 1, true)
	case models.ViewModePods:
		a.mainFlex.AddItem(a.podsTable, 0, 1, true)
	}

	a.mainFlex.AddItem(a.footer, 1, 0, false)
}

// Run starts the application
func (a *App) Run() error {
	// Start metrics collection goroutine
	go a.metricsLoop()

	// Start UI refresh goroutine
	go a.refreshLoop()

	// Run the application
	return a.app.SetRoot(a.mainFlex, true).EnableMouse(true).Run()
}

// metricsLoop periodically collects metrics
func (a *App) metricsLoop() {
	ticker := time.NewTicker(a.config.RefreshInterval)
	defer ticker.Stop()

	// Initial fetch
	a.fetchMetrics()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.refreshNow:
			a.fetchMetrics()
		case <-ticker.C:
			a.fetchMetrics()
		}
	}
}

// fetchMetrics fetches and stores new metrics
func (a *App) fetchMetrics() {
	m, err := a.collector.Collect(a.ctx)
	if err != nil && m == nil {
		return
	}

	a.metricsMu.Lock()
	a.metrics = m
	a.metricsMu.Unlock()

	// Queue UI update
	a.app.QueueUpdateDraw(func() {
		a.updateUI()
	})
}

// refreshLoop periodically updates the UI
func (a *App) refreshLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.app.QueueUpdateDraw(func() {
				a.updateUI()
			})
		}
	}
}

// updateUI updates all UI components
func (a *App) updateUI() {
	a.metricsMu.RLock()
	m := a.metrics
	a.metricsMu.RUnlock()

	a.stateMu.RLock()
	state := a.state
	a.stateMu.RUnlock()

	a.updateHeader(m, state)
	a.updateSummary(m)
	a.updateNodesTable(m, state)
	a.updatePodsTable(m, state)
	a.updateFooter(m, state)
}

// updateHeader updates the header text
func (a *App) updateHeader(m *models.ClusterMetrics, state models.AppState) {
	if m == nil {
		a.header.SetText("[yellow]ktop[-] - Kubernetes Cluster Monitor   [red]Connecting...[-]")
		return
	}

	elapsed := time.Since(m.Timestamp)
	refreshStr := formatDuration(elapsed)

	header := fmt.Sprintf("[yellow]ktop[-] - [white]%s[-] [gray](%s)[-]   Nodes: [white]%d/%d[-]   [gray]Updated: %s ago[-]",
		m.ClusterInfo.Name, m.ClusterInfo.Context, m.ReadyNodes, m.TotalNodes, refreshStr)

	if m.Error != nil {
		header += fmt.Sprintf("  [red]⚠ %s[-]", m.Error.Error())
	}

	a.header.SetText(header)
}

// updateSummary updates the cluster resource summary bar
func (a *App) updateSummary(m *models.ClusterMetrics) {
	if m == nil {
		a.summary.SetText("[gray]Loading cluster resources...[-]")
		return
	}

	// Calculate percentages
	var cpuPercent, memPercent, diskPercent float64
	if m.TotalCPUCapacity > 0 {
		cpuPercent = float64(m.TotalCPUUsed) / float64(m.TotalCPUCapacity) * 100
	}
	if m.TotalMemoryCapacity > 0 {
		memPercent = float64(m.TotalMemoryUsed) / float64(m.TotalMemoryCapacity) * 100
	}
	if m.TotalDiskCapacity > 0 {
		diskPercent = float64(m.TotalDiskUsed) / float64(m.TotalDiskCapacity) * 100
	}

	// Get colors based on usage
	cpuColor := a.colors.GetResourceColor(cpuPercent)
	memColor := a.colors.GetResourceColor(memPercent)
	diskColor := a.colors.GetResourceColor(diskPercent)

	// Build summary line 1: CPU and Memory
	line1 := fmt.Sprintf("[white]CPU:[white] [gray]%d cores[-]  %s / %s  %s   ",
		m.TotalCPUCores,
		ColoredText(metrics.FormatCPU(m.TotalCPUUsed), cpuColor),
		metrics.FormatCPU(m.TotalCPUCapacity),
		ColoredText(fmt.Sprintf("%.1f%%", cpuPercent), cpuColor))

	line1 += fmt.Sprintf("[white]RAM:[-]  %s / %s  %s   ",
		ColoredText(metrics.FormatMemory(m.TotalMemoryUsed), memColor),
		metrics.FormatMemory(m.TotalMemoryCapacity),
		ColoredText(fmt.Sprintf("%.1f%%", memPercent), memColor))

	// Add disk if available
	if m.TotalDiskCapacity > 0 {
		line1 += fmt.Sprintf("[white]DISK:[-]  %s / %s  %s   ",
			ColoredText(metrics.FormatMemory(m.TotalDiskUsed), diskColor),
			metrics.FormatMemory(m.TotalDiskCapacity),
			ColoredText(fmt.Sprintf("%.1f%%", diskPercent), diskColor))
	}

	// Add GPU count
	if m.TotalGPUs > 0 {
		line1 += fmt.Sprintf("[white]GPUs:[-] [green]%d[-]", m.TotalGPUs)
	}

	// Build summary line 2: Pods
	line2 := fmt.Sprintf("[white]Pods:[-] [cyan]%d[-] running", m.TotalPods)

	a.summary.SetText(line1 + "\n" + line2)
}

// updateNodesTable updates the nodes table
func (a *App) updateNodesTable(m *models.ClusterMetrics, state models.AppState) {
	a.nodesTable.Clear()

	// Set headers
	headers := []string{"NODE", "STATUS", "CPU", "CPU%", "MEMORY", "MEM%", "PODS", "GPU"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		if i > 1 {
			cell.SetAlign(tview.AlignRight)
		}
		a.nodesTable.SetCell(0, i, cell)
	}

	if m == nil || len(m.Nodes) == 0 {
		a.nodesTable.SetCell(1, 0, tview.NewTableCell("No nodes found").SetTextColor(tcell.ColorGray))
		return
	}

	// Sort nodes
	nodes := make([]models.Node, len(m.Nodes))
	copy(nodes, m.Nodes)
	metrics.SortNodes(nodes, state.NodeSortField, state.NodeSortAsc)

	// Update title with sort indicator
	sortIndicator := "↓"
	if state.NodeSortAsc {
		sortIndicator = "↑"
	}
	a.nodesTable.SetTitle(fmt.Sprintf(" NODES (sort: %s %s) ", state.NodeSortField.String(), sortIndicator))

	// Populate rows
	for i, node := range nodes {
		row := i + 1

		// Node name
		a.nodesTable.SetCell(row, 0, tview.NewTableCell(node.Name).
			SetTextColor(tcell.ColorWhite))

		// Status
		statusColor := a.colors.GetNodeStatusColor(node.Status)
		a.nodesTable.SetCell(row, 1, tview.NewTableCell(string(node.Status)).
			SetTextColor(statusColor))

		// CPU usage
		cpuColor := a.colors.GetResourceColor(node.CPU.Percent)
		a.nodesTable.SetCell(row, 2, tview.NewTableCell(metrics.FormatCPU(node.CPU.Current)).
			SetTextColor(cpuColor).SetAlign(tview.AlignRight))

		// CPU percent with bar
		a.nodesTable.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%.1f%%", node.CPU.Percent)).
			SetTextColor(cpuColor).SetAlign(tview.AlignRight))

		// Memory usage
		memColor := a.colors.GetResourceColor(node.Memory.Percent)
		a.nodesTable.SetCell(row, 4, tview.NewTableCell(metrics.FormatMemory(node.Memory.Current)).
			SetTextColor(memColor).SetAlign(tview.AlignRight))

		// Memory percent
		a.nodesTable.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%.1f%%", node.Memory.Percent)).
			SetTextColor(memColor).SetAlign(tview.AlignRight))

		// Pod count
		a.nodesTable.SetCell(row, 6, tview.NewTableCell(fmt.Sprintf("%d", node.PodCount)).
			SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignRight))

		// GPU
		gpuStr := "-"
		if node.GPU != nil {
			gpuStr = fmt.Sprintf("%d", node.GPU.Count)
		}
		a.nodesTable.SetCell(row, 7, tview.NewTableCell(gpuStr).
			SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignRight))
	}
}

// updatePodsTable updates the pods table
func (a *App) updatePodsTable(m *models.ClusterMetrics, state models.AppState) {
	a.podsTable.Clear()

	// Set headers
	headers := []string{"NAMESPACE", "POD", "STATUS", "CPU", "MEMORY", "RESTARTS", "NODE"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		if i >= 3 && i <= 5 {
			cell.SetAlign(tview.AlignRight)
		}
		a.podsTable.SetCell(0, i, cell)
	}

	if m == nil || len(m.Pods) == 0 {
		a.podsTable.SetCell(1, 0, tview.NewTableCell("No pods found").SetTextColor(tcell.ColorGray))
		return
	}

	// Filter and sort pods
	pods := metrics.FilterPods(m.Pods, state.NamespaceFilter, state.ShowSystem)
	metrics.SortPods(pods, state.PodSortField, state.PodSortAsc)
	pods = metrics.LimitPods(pods, a.config.TopPods)

	// Update title
	sortIndicator := "↓"
	if state.PodSortAsc {
		sortIndicator = "↑"
	}
	filterStr := "all"
	if state.NamespaceFilter != "" {
		filterStr = state.NamespaceFilter
	}
	a.podsTable.SetTitle(fmt.Sprintf(" PODS (top %d by %s %s) [filter: %s] ",
		len(pods), state.PodSortField.String(), sortIndicator, filterStr))

	// Populate rows
	for i, pod := range pods {
		row := i + 1

		// Namespace
		nsColor := a.colors.GetNamespaceColor(pod.Namespace)
		a.podsTable.SetCell(row, 0, tview.NewTableCell(truncate(pod.Namespace, 20)).
			SetTextColor(nsColor))

		// Pod name
		a.podsTable.SetCell(row, 1, tview.NewTableCell(truncate(pod.Name, 40)).
			SetTextColor(tcell.ColorWhite))

		// Status
		statusColor := a.colors.GetPodStatusColor(pod.Status)
		a.podsTable.SetCell(row, 2, tview.NewTableCell(string(pod.Status)).
			SetTextColor(statusColor))

		// CPU
		a.podsTable.SetCell(row, 3, tview.NewTableCell(metrics.FormatCPU(pod.CPU)).
			SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignRight))

		// Memory
		a.podsTable.SetCell(row, 4, tview.NewTableCell(metrics.FormatMemory(pod.Memory)).
			SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignRight))

		// Restarts
		restartColor := tcell.ColorWhite
		if pod.RestartCount > 0 {
			restartColor = tcell.ColorYellow
		}
		if pod.RestartCount > 5 {
			restartColor = tcell.ColorRed
		}
		a.podsTable.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%d", pod.RestartCount)).
			SetTextColor(restartColor).SetAlign(tview.AlignRight))

		// Node
		a.podsTable.SetCell(row, 6, tview.NewTableCell(truncate(pod.NodeName, 20)).
			SetTextColor(tcell.ColorGray))
	}
}

// updateFooter updates the footer text
func (a *App) updateFooter(m *models.ClusterMetrics, state models.AppState) {
	footer := "[yellow]q[-]uit  [yellow]r[-]efresh  [yellow]s[-]ort nodes  [yellow]p[-]od sort  "
	footer += "[yellow]f/n[-]amespace  [yellow]t[-]oggle view  [yellow]a[-]ll ns  [yellow]?[-]help"
	a.footer.SetText(footer)
}

// helpText returns the help modal text
func helpText() string {
	return `ktop - Kubernetes Cluster Monitor

Keyboard Controls:
  q     Quit
  r     Force refresh
  s     Sort nodes (cycle)
  p     Sort pods (cycle)
  f/n   Filter by namespace
  t     Toggle view mode
  a     Toggle system namespaces
  Tab   Switch focus
  ?     Show this help

Press any key to close`
}

// truncate truncates a string to max length with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
