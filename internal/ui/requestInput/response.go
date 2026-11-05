package requestinput

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// ResponsePanel renders the response area below the request editors as a
// scrollable code viewer. Everything (tabs, status strip, view mode toolbar,
// gjson filter, thin rule and viewport) lives inside ONE rounded border, so
// the left edge never misaligns with the request editor above it.
type ResponsePanel struct {
	Width        int
	Height       int
	Method       string
	URL          string
	Status       string
	Code         int
	Body         string
	RawBody      string
	Headers      []string
	Cookies      []string
	Size         int
	ContentType  string
	Err          string
	Time         time.Duration
	Attempts     int
	Loading      bool
	Tab          int
	RawMode      bool
	PreviewMode  bool
	ScrollOffset int
	Borderless   bool
	Focused      bool

	// Selection is linewise and only active after a drag.
	SelActive   bool
	selDragging bool
	selAnchor   int
	SelStart    int
	SelEnd      int

	// In-response text search.
	Find        textinput.Model
	FindFocused bool
	Matches     []int
	MatchIdx    int

	// Request metadata shown in Preview.
	Redirects []string
	Timings   RequestTimings
	TLS       *TLSInfo

	// Transient feedback, e.g. "Copied 12 lines".
	Notice   string
	noticeID int

	vp           viewport.Model
	Filter       textinput.Model
	FilterFocued bool
	spinner      spinner.Model
	startedAt    time.Time
}

func NewResponsePanel() *ResponsePanel {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(6))
	vp.MouseWheelEnabled = false
	vp.SoftWrap = false
	vp.LeftGutterFunc = lineNumberGutter

	filter := textinput.New()
	filter.Prompt = ""
	filter.Placeholder = ".data.items[0]"
	filter.CharLimit = 128

	find := textinput.New()
	find.Prompt = ""
	find.Placeholder = "find text..."
	find.CharLimit = 128

	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))

	return &ResponsePanel{Height: 6, vp: vp, Filter: filter, Find: find, spinner: sp}
}

func lineNumberGutter(info viewport.GutterContext) string {
	if info.Soft {
		return "     │ "
	}
	if info.Index >= info.TotalLines {
		return "   ~ │ "
	}
	return fmt.Sprintf("%*d │ ", 3, info.Index+1)
}

// UpdateSpinner advances the loading spinner; returns a command to keep it
// animating while a request is in flight.
func (p *ResponsePanel) UpdateSpinner(msg tea.Msg) tea.Cmd {
	if p == nil {
		return nil
	}
	var cmd tea.Cmd
	p.spinner, cmd = p.spinner.Update(msg)
	return cmd
}

// SpinnerTick starts the spinner animation.
func (p *ResponsePanel) SpinnerTick() tea.Cmd {
	if p == nil {
		return nil
	}
	return p.spinner.Tick
}

// Elapsed returns how long the in-flight request has been running.
func (p *ResponsePanel) Elapsed() time.Duration {
	if p == nil || p.startedAt.IsZero() {
		return 0
	}
	return time.Since(p.startedAt)
}

// SetFilter focuses or blurs the gjson filter input.
func (p *ResponsePanel) SetFilter(focused bool) {
	if p == nil {
		return
	}
	p.FilterFocued = focused
	if focused {
		p.Filter.Focus()
	} else {
		p.Filter.Blur()
	}
}

// UpdateFilter routes a message to the filter input.
func (p *ResponsePanel) UpdateFilter(msg tea.Msg) tea.Cmd {
	if p == nil {
		return nil
	}
	var cmd tea.Cmd
	p.Filter, cmd = p.Filter.Update(msg)
	return cmd
}

func (p *ResponsePanel) SetSize(width, height int) {
	if p == nil {
		return
	}
	p.Width = max(width, 12)
	p.Height = max(height, 5)
}

func (p *ResponsePanel) SetLoading() {
	if p == nil {
		return
	}
	p.Loading = true
	p.Method = ""
	p.URL = ""
	p.Status = ""
	p.Body = ""
	p.Err = ""
	p.Code = 0
	p.Time = 0
	p.Attempts = 0
	p.Headers = nil
	p.Cookies = nil
	p.RawBody = ""
	p.Size = 0
	p.ContentType = ""
	p.ScrollOffset = 0
	p.Redirects = nil
	p.Timings = RequestTimings{}
	p.TLS = nil
	p.resetSelection()
	p.Matches = nil
	p.MatchIdx = 0
	p.startedAt = time.Now()
}

// HasContent reports whether a real response body is present. Used to decide
// whether the line-number gutter should appear.
func (p *ResponsePanel) HasContent() bool {
	if p == nil {
		return false
	}
	return p.Status != "" || p.Err != "" || p.Body != "" || p.RawBody != ""
}

func (p *ResponsePanel) SetResponse(response ResponseMsg) {
	if p == nil {
		return
	}
	p.Loading = false
	p.Method = response.Method
	p.URL = response.URL
	p.Status = response.Status
	p.Code = response.Code
	p.Body = response.Body
	if response.RawBody != "" {
		p.RawBody = response.RawBody
	} else {
		p.RawBody = response.Body
	}
	p.Headers = response.Headers
	p.Cookies = response.Cookies
	p.Size = response.Size
	p.ContentType = response.ContentType
	p.Err = response.Err
	p.Time = response.Duration
	p.Attempts = response.Attempts
	p.Redirects = append([]string(nil), response.Redirects...)
	p.Timings = response.Timings
	p.TLS = response.TLS
	p.ScrollOffset = 0
	p.vp.SetYOffset(0)
	p.resetSelection()
	p.recomputeMatches()
	p.startedAt = time.Time{}
}

func (p *ResponsePanel) HandleMouse(msg tea.MouseClickMsg) {
	if p == nil || msg.Button != tea.MouseLeft {
		return
	}
	switch {
	case zone.Get("response_body").InBounds(msg):
		p.Focused = true
		p.startSelection(msg)
	case zone.Get("response_tab_body").InBounds(msg):
		p.Tab = 0
		p.afterViewChange()
	case zone.Get("response_tab_cookies").InBounds(msg):
		p.Tab = 1
		p.afterViewChange()
	case zone.Get("response_tab_headers").InBounds(msg):
		p.Tab = 2
		p.afterViewChange()
	case zone.Get("response_mode_pretty").InBounds(msg):
		p.RawMode = false
		p.PreviewMode = false
		p.afterViewChange()
	case zone.Get("response_mode_raw").InBounds(msg):
		p.RawMode = true
		p.PreviewMode = false
		p.afterViewChange()
	case zone.Get("response_mode_preview").InBounds(msg):
		p.PreviewMode = true
		p.afterViewChange()
	case zone.Get("response_find").InBounds(msg):
		p.Focused = true
		_ = p.OpenFind()
	case zone.Get("response_filter").InBounds(msg):
		p.Focused = true
		p.SetFilter(true)
	}
}

// HandleMotion extends the current mouse selection while the left button is
// held. Bubble Tea only delivers these messages in CellMotion mouse mode.
func (p *ResponsePanel) HandleMotion(msg tea.MouseMotionMsg) {
	if p == nil || !p.selDragging {
		return
	}
	z := zone.Get("response_body")
	if z.IsZero() {
		return
	}
	lines := p.plainLines()
	if len(lines) == 0 {
		return
	}
	row := p.vp.YOffset() + msg.Y - z.StartY
	p.extendSelection(row)
}

func (p *ResponsePanel) HandleRelease() {
	if p == nil {
		return
	}
	if p.selDragging && !p.SelActive {
		p.resetSelection()
	}
	p.selDragging = false
}

func (p *ResponsePanel) IsDraggingSelection() bool {
	return p != nil && p.selDragging
}

func (p *ResponsePanel) HandleWheel(msg tea.MouseWheelMsg) {
	if p == nil {
		return
	}
	switch msg.Button {
	case tea.MouseWheelUp:
		p.vp.ScrollUp(3)
	case tea.MouseWheelDown:
		p.vp.ScrollDown(3)
	}
	p.ScrollOffset = p.vp.YOffset()
}

func (p *ResponsePanel) ScrollUp(n int) {
	if p == nil {
		return
	}
	p.vp.ScrollUp(n)
	p.ScrollOffset = p.vp.YOffset()
}

func (p *ResponsePanel) ScrollDown(n int, maxLen int) {
	if p == nil {
		return
	}
	p.vp.ScrollDown(n)
	p.ScrollOffset = p.vp.YOffset()
}
