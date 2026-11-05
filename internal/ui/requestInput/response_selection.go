package requestinput

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

// plainLines returns the visible content as plain, sanitized text. It is the
// single source for selection, copying and text search.
func (p *ResponsePanel) plainLines() []string {
	if p == nil {
		return nil
	}
	lines := p.contentLines()
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = sanitizeLine(ansi.Strip(line))
	}
	return out
}

func (p *ResponsePanel) startSelection(msg tea.MouseClickMsg) {
	z := zone.Get("response_body")
	if z.IsZero() {
		return
	}
	lines := p.plainLines()
	if len(lines) == 0 {
		return
	}
	row := p.vp.YOffset() + msg.Y - z.StartY
	row = clampInt(row, 0, len(lines)-1)
	p.selAnchor = row
	p.SelStart = row
	p.SelEnd = row
	p.SelActive = false
	p.selDragging = true
}

func (p *ResponsePanel) extendSelection(row int) {
	lines := p.plainLines()
	if len(lines) == 0 {
		return
	}
	row = clampInt(row, 0, len(lines)-1)
	p.SelEnd = row
	if row != p.selAnchor {
		p.SelActive = true
	}
}

func (p *ResponsePanel) resetSelection() {
	if p == nil {
		return
	}
	p.SelActive = false
	p.selDragging = false
	p.selAnchor = 0
	p.SelStart = 0
	p.SelEnd = 0
}

// ClearSelection removes the current selection but keeps response focus.
func (p *ResponsePanel) ClearSelection() {
	if p == nil {
		return
	}
	p.resetSelection()
}

func (p *ResponsePanel) selectionRange() (int, int, bool) {
	if p == nil || !p.SelActive {
		return 0, 0, false
	}
	start, end := p.SelStart, p.SelEnd
	if start > end {
		start, end = end, start
	}
	return start, end, true
}

func (p *ResponsePanel) selectedLine(i int) bool {
	start, end, ok := p.selectionRange()
	return ok && i >= start && i <= end
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
