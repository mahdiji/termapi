package historyscreen

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func framed(text string, width int, border lipgloss.Style) string {
	text = ansi.Truncate(text, width, "…")
	text += strings.Repeat(" ", max(width-lipgloss.Width(text), 0))
	return border.Render("│") + text + border.Render("│")
}

func tableRow(cells []string, widths []int) string {
	var parts []string
	for i, width := range widths {
		value := " " + ansi.Truncate(cells[i], max(width-1, 0), "…")
		value += strings.Repeat(" ", max(width-lipgloss.Width(value), 0))
		parts = append(parts, value)
	}
	return strings.Join(parts, "")
}

func methodColor(method string) color.Color {
	switch strings.ToUpper(method) {
	case "GET":
		return lipgloss.Color("#04B575")
	case "POST":
		return lipgloss.Color("#00E5FF")
	case "PUT", "PATCH":
		return lipgloss.Color("#FFB86C")
	case "DELETE":
		return lipgloss.Color("#F38BA8")
	default:
		return lipgloss.Color("#AAB7C5")
	}
}

func display(value string) string {
	if value == "" {
		return "—"
	}
	return value
}

// formatResponseTime renders a millisecond duration using the most readable
// unit: ms below one second, seconds below one minute, then minutes/hours.
func formatResponseTime(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	seconds := float64(ms) / 1000
	switch {
	case seconds < 10:
		return formatSeconds(seconds, 2)
	case seconds < 60:
		return formatSeconds(seconds, 1)
	case seconds < 3600:
		minutes := int(seconds) / 60
		rest := int(seconds) % 60
		if rest == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm %ds", minutes, rest)
	default:
		hours := int(seconds) / 3600
		minutes := (int(seconds) % 3600) / 60
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

func formatSeconds(value float64, decimals int) string {
	text := strconv.FormatFloat(value, 'f', decimals, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimSuffix(text, ".")
	return text + "s"
}
