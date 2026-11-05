package wirescreen

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func fitLine(s string, width int) string {
	w := ansi.StringWidth(s)
	if w == width {
		return s
	}
	if w > width {
		return ansi.Truncate(s, width, "…")
	}
	return s + strings.Repeat(" ", width-w)
}
