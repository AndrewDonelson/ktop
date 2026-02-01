package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/nlaak/ktop/internal/config"
	"github.com/nlaak/ktop/internal/models"
)

// Colors defines the color palette for the UI
type Colors struct {
	// Resource usage colors
	Healthy  tcell.Color // Green - 0-50%
	Warning  tcell.Color // Yellow - 50-80%
	Critical tcell.Color // Red - 80%+

	// Status colors
	StatusOK  tcell.Color // Green - Ready
	StatusBad tcell.Color // Red - NotReady

	// UI element colors
	Header     tcell.Color
	Border     tcell.Color
	Background tcell.Color
	Text       tcell.Color
	TextDim    tcell.Color
	System     tcell.Color // System namespaces
	Selected   tcell.Color
	Highlight  tcell.Color

	// Pod status colors
	PodRunning   tcell.Color
	PodPending   tcell.Color
	PodSucceeded tcell.Color
	PodFailed    tcell.Color
}

// DefaultColors returns the default color scheme
func DefaultColors() Colors {
	return Colors{
		// Resource usage
		Healthy:  tcell.ColorGreen,
		Warning:  tcell.ColorYellow,
		Critical: tcell.ColorRed,

		// Status
		StatusOK:  tcell.ColorGreen,
		StatusBad: tcell.ColorRed,

		// UI elements
		Header:     tcell.ColorYellow,
		Border:     tcell.ColorWhite,
		Background: tcell.ColorDefault,
		Text:       tcell.ColorWhite,
		TextDim:    tcell.ColorGray,
		System:     tcell.ColorTeal,
		Selected:   tcell.ColorBlue,
		Highlight:  tcell.ColorAqua,

		// Pod status
		PodRunning:   tcell.ColorGreen,
		PodPending:   tcell.ColorYellow,
		PodSucceeded: tcell.ColorBlue,
		PodFailed:    tcell.ColorRed,
	}
}

// Thresholds for color coding
var thresholds = config.DefaultThresholds()

// GetResourceColor returns the appropriate color for a resource usage percentage
func (c Colors) GetResourceColor(percent float64) tcell.Color {
	switch {
	case percent >= thresholds.CriticalPercent:
		return c.Critical
	case percent >= thresholds.WarningPercent:
		return c.Warning
	default:
		return c.Healthy
	}
}

// GetNodeStatusColor returns the color for a node status
func (c Colors) GetNodeStatusColor(status models.NodeStatus) tcell.Color {
	if status == models.NodeStatusReady {
		return c.StatusOK
	}
	return c.StatusBad
}

// GetPodStatusColor returns the color for a pod status
func (c Colors) GetPodStatusColor(status models.PodStatus) tcell.Color {
	switch status {
	case models.PodStatusRunning:
		return c.PodRunning
	case models.PodStatusPending:
		return c.PodPending
	case models.PodStatusSucceeded:
		return c.PodSucceeded
	case models.PodStatusFailed:
		return c.PodFailed
	default:
		return c.TextDim
	}
}

// GetNamespaceColor returns color for namespace (system vs user)
func (c Colors) GetNamespaceColor(namespace string) tcell.Color {
	if models.IsSystemNamespace(namespace) {
		return c.System
	}
	return c.Text
}

// ColorTag returns a tview color tag string for a color
func ColorTag(color tcell.Color) string {
	switch color {
	case tcell.ColorGreen:
		return "[green]"
	case tcell.ColorYellow:
		return "[yellow]"
	case tcell.ColorRed:
		return "[red]"
	case tcell.ColorBlue:
		return "[blue]"
	case tcell.ColorTeal:
		return "[teal]"
	case tcell.ColorAqua:
		return "[aqua]"
	case tcell.ColorGray:
		return "[gray]"
	case tcell.ColorWhite:
		return "[white]"
	default:
		return "[white]"
	}
}

// ColorTagClose returns the closing color tag
func ColorTagClose() string {
	return "[-]"
}

// ColoredText wraps text with color tags
func ColoredText(text string, color tcell.Color) string {
	return ColorTag(color) + text + ColorTagClose()
}

// ProgressBar generates a simple text-based progress bar
func ProgressBar(percent float64, width int, colors Colors) string {
	if width < 3 {
		width = 3
	}

	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	color := colors.GetResourceColor(percent)
	
	bar := ColorTag(color)
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	bar += ColorTagClose()
	
	for i := filled; i < width; i++ {
		bar += "░"
	}

	return bar
}

// ProgressBarCompact generates a compact progress bar with percentage
func ProgressBarCompact(percent float64, width int, colors Colors) string {
	barWidth := width - 6 // Reserve space for percentage display
	if barWidth < 3 {
		barWidth = 3
	}

	bar := ProgressBar(percent, barWidth, colors)
	color := colors.GetResourceColor(percent)
	
	return bar + " " + ColoredText(FormatPercentCompact(percent), color)
}

// FormatPercentCompact formats percentage in compact form
func FormatPercentCompact(percent float64) string {
	if percent >= 100 {
		return "100%"
	}
	if percent >= 10 {
		return fmt.Sprintf("%.0f%%", percent)
	}
	return fmt.Sprintf("%.1f%%", percent)
}
