package requestinput

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// NoticeExpiredMsg clears a transient response-panel notice.
type NoticeExpiredMsg struct {
	ID int
}

// Focus gives the response panel keyboard ownership.
func (p *ResponsePanel) Focus() {
	if p == nil {
		return
	}
	p.Focused = true
}

// Blur removes keyboard focus and closes transient editors.
func (p *ResponsePanel) Blur() {
	if p == nil {
		return
	}
	p.Focused = false
	p.FindFocused = false
	p.Find.Blur()
	p.FilterFocued = false
	p.Filter.Blur()
	p.selDragging = false
}

// NextTab cycles Body / Cookies / Headers.
func (p *ResponsePanel) NextTab(delta int) {
	if p == nil {
		return
	}
	p.Tab = (p.Tab + delta + 3) % 3
	p.afterViewChange()
}

// afterViewChange resets view-dependent state (scroll, selection, matches).
func (p *ResponsePanel) afterViewChange() {
	if p == nil {
		return
	}
	p.ScrollOffset = 0
	p.vp.SetYOffset(0)
	p.resetSelection()
	p.recomputeMatches()
}

// HandlesClick reports whether the click landed anywhere inside the response
// panel, so the app can focus it instead of the request editor.
func (p *ResponsePanel) HandlesClick(msg tea.MouseMsg) bool {
	if p == nil {
		return false
	}
	ids := []string{
		"response_body",
		"response_tab_body", "response_tab_cookies", "response_tab_headers",
		"response_mode_pretty", "response_mode_raw", "response_mode_preview",
		"response_filter", "response_find",
	}
	for _, id := range ids {
		if z := zone.Get(id); !z.IsZero() && z.InBounds(msg) {
			return true
		}
	}
	return false
}

// HandlesWheel reports whether the pointer is over the response panel. This
// keeps wheel events over the collection sidebar from scrolling the response.
func (p *ResponsePanel) HandlesWheel(msg tea.MouseWheelMsg) bool {
	return p.HandlesClick(tea.MouseClickMsg(msg))
}

// SetNotice shows a short message and schedules it to disappear.
func (p *ResponsePanel) SetNotice(text string) tea.Cmd {
	if p == nil {
		return nil
	}
	p.Notice = text
	p.noticeID++
	id := p.noticeID
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return NoticeExpiredMsg{ID: id}
	})
}

// ClearNotice hides the notice if it is still the same one.
func (p *ResponsePanel) ClearNotice(id int) {
	if p == nil || id != p.noticeID {
		return
	}
	p.Notice = ""
}

// CopyAll returns every line of the active tab/mode as plain text.
func (p *ResponsePanel) CopyAll() (string, int) {
	if p == nil {
		return "", 0
	}
	lines := p.plainLines()
	if len(lines) == 0 {
		return "", 0
	}
	return strings.Join(lines, "\n"), len(lines)
}

// CopySelection returns the selected lines, or the whole tab when there is no
// active selection.
func (p *ResponsePanel) CopySelection() (string, int, bool) {
	if p == nil {
		return "", 0, false
	}
	start, end, ok := p.selectionRange()
	if !ok {
		return "", 0, false
	}
	lines := p.plainLines()
	if start < 0 || end >= len(lines) {
		return "", 0, false
	}
	return strings.Join(lines[start:end+1], "\n"), end - start + 1, true
}

// CopiedNotice builds the standard feedback text for a copy action.
func CopiedNotice(lines int) string {
	if lines == 1 {
		return "Copied 1 line"
	}
	return fmt.Sprintf("Copied %d lines", lines)
}
