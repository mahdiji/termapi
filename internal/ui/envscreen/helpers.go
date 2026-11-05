package envscreen

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// truncate cuts s to at most n terminal cells, appending an ellipsis when cut.
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return ansi.Truncate(s, n, "…")
}

// padRight pads s with spaces until it occupies exactly w terminal cells.
func padRight(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) >= w {
		return truncate(s, w)
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

// fit truncates or pads s to exactly w terminal cells.
func fit(s string, w int) string {
	return padRight(truncate(s, w), w)
}

// visibleWindow returns the slice [start, end) that keeps cursor on screen
// while showing at most rows entries.
func visibleWindow(cursor, total, rows int) (int, int) {
	if rows <= 0 || total <= 0 {
		return 0, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	start := 0
	if cursor >= rows {
		start = cursor - rows + 1
	}
	end := start + rows
	if end > total {
		end = total
	}
	return start, end
}

// joinRow renders left and right on one row, padding the gap between them.
func joinRow(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	right = truncate(right, max(width/2, 8))
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		left = truncate(left, max(width-lipgloss.Width(right)-1, 1))
		gap = max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	}
	return left + strings.Repeat(" ", gap) + right
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
