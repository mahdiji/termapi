package requestinput

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"api/internal/ui/styles"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/tidwall/gjson"
)

// Response pane layout (top to bottom, all inside ONE rounded border):
//
//	Row 1  header : tabs (left)            status strip (right)
//	Row 2  toolbar: Pretty · Raw · Preview  gjson filter (right)
//	Row 3  rule   : a single muted horizontal rule
//	Rest   body   : viewport (line-numbered only when content exists)
func (p *ResponsePanel) View() string {
	if p == nil {
		return ""
	}

	innerWidth := max(p.Width-2, 10)
	contentHeight := max(p.Height-2, 1)
	if p.Borderless {
		innerWidth = max(p.Width, 10)
		contentHeight = max(p.Height, 1)
	}

	// --- Header row: tabs left, status strip right ----------------------
	left := p.tabBar()
	right := p.statusStrip()
	header := p.joinRow(left, right, innerWidth)

	// --- Toolbar row: view modes left, gjson filter right ----------------
	toolbar := p.joinRow(p.viewModes(), p.filterField(), innerWidth)

	// --- Rule row --------------------------------------------------------
	rule := styles.Dim().Render(strings.Repeat("─", innerWidth))

	headerHeight := 3 // header + toolbar + rule
	bodyHeight := max(contentHeight-headerHeight, 1)

	bodyView := p.renderBody(innerWidth, bodyHeight)
	if p.HasContent() {
		bodyView = zone.Mark("response_body", bodyView)
	}

	// --- Assemble inside one rounded border ------------------------------
	border := styles.BorderStyle(p.FilterFocued)
	title := styles.Title(false).Render(" Response ")

	var out []string
	if !p.Borderless {
		topRemaining := max(innerWidth-lipgloss.Width(title)-1, 0)
		out = append(out, border.Render("╭─")+title+border.Render(strings.Repeat("─", topRemaining)+"╮"))
	}

	// Header, toolbar and rule are always rendered; the rest is the body.
	rows := []string{header, toolbar, rule}
	rows = append(rows, strings.Split(bodyView, "\n")...)
	for i := 0; i < contentHeight; i++ {
		content := ""
		if i < len(rows) {
			content = rows[i]
		}
		out = append(out, p.frameRow(border, content, innerWidth))
	}

	if !p.Borderless {
		out = append(out, border.Render("╰"+strings.Repeat("─", innerWidth)+"╯"))
	}
	return strings.Join(out, "\n")
}

// frameRow wraps a single line of content inside the left/right border.
func (p *ResponsePanel) frameRow(border lipgloss.Style, content string, innerWidth int) string {
	if lipgloss.Width(content) > innerWidth {
		content = truncateAnsi(content, innerWidth)
	}
	if w := lipgloss.Width(content); w < innerWidth {
		content += strings.Repeat(" ", innerWidth-w)
	}
	// Borderless keeps the vertical sides so it aligns with a parent frame;
	// only the top/bottom border is omitted.
	return border.Render("│") + content + border.Render("│")
}

// joinRow places left and right content on one full-width line, computing the
// gap dynamically and truncating the left side with an ellipsis when there is
// no room. Never hardcodes widths.
func (p *ResponsePanel) joinRow(left, right string, innerWidth int) string {
	gap := innerWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		// Not enough room: truncate the left side so the right side stays put.
		keep := innerWidth - lipgloss.Width(right) - 2
		if keep < 1 {
			keep = 1
		}
		left = truncateAnsi(left, keep) + styles.Dim().Render("…")
		gap = innerWidth - lipgloss.Width(left) - lipgloss.Width(right)
	}
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// statusStrip renders the right side of the header row. It always renders
// something so the layout never jumps: "Ready" before any request, a spinner
// with live elapsed time while loading, or the code pill + time + size after.
func (p *ResponsePanel) statusStrip() string {
	if p.Notice != "" {
		return styles.Bright().Bold(true).Render(p.Notice)
	}
	switch {
	case p.Loading:
		sp := lipgloss.NewStyle().Foreground(styles.ColInfo).Render(p.spinner.View())
		elapsed := styles.Text().Render(fmt.Sprintf("%d ms", p.Elapsed().Milliseconds()))
		hint := styles.Dim().Render("Esc cancel")
		return sp + " " + elapsed + styles.Dim().Render(" · ") + hint
	case p.Err != "":
		badge := lipgloss.NewStyle().
			Foreground(styles.ColBackground).
			Background(styles.ColError).
			Bold(true).Padding(0, 1).Render("ERR")
		detail := styles.Muted().Render("request failed")
		suffix := ""
		if p.Attempts > 1 {
			suffix = styles.Dim().Render(fmt.Sprintf(" · retries %d", p.Attempts))
		}
		return badge + " " + detail + suffix
	case p.Status != "":
		dot := lipgloss.NewStyle().Foreground(styles.StatusColor(p.Code)).Render("●")
		code := lipgloss.NewStyle().Foreground(styles.StatusColor(p.Code)).Bold(true).
			Render(fmt.Sprintf("%d %s", p.Code, http.StatusText(p.Code)))
		meta := styles.Dim().Render(fmt.Sprintf("%s · %s",
			p.Time.Round(time.Millisecond).String(), formatBytes(p.Size)))
		return dot + " " + code + styles.Dim().Render(" · ") + meta
	default:
		return styles.Dim().Render("Ready")
	}
}

// tabBar renders the left side of the header row: Body / Cookies (n) / Headers (n).
func (p *ResponsePanel) tabBar() string {
	type tabSpec struct {
		name  string
		id    string
		count int
	}
	tabs := []tabSpec{
		{"Body", "response_tab_body", 0},
		{"Cookies", "response_tab_cookies", len(p.Cookies)},
		{"Headers", "response_tab_headers", len(p.Headers)},
	}
	parts := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		label := tab.name
		if tab.count > 0 {
			label += fmt.Sprintf(" (%d)", tab.count)
		}
		var style lipgloss.Style
		if p.Tab == i {
			style = lipgloss.NewStyle().
				Foreground(styles.ColBackground).
				Background(styles.ColAccent).
				Bold(true).Padding(0, 1)
		} else {
			style = lipgloss.NewStyle().Foreground(styles.ColTextMuted).Padding(0, 1)
		}
		parts = append(parts, zone.Mark(tab.id, style.Render(label)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// viewModes renders the left side of the toolbar row: Pretty · Raw · Preview.
func (p *ResponsePanel) viewModes() string {
	if p.Tab != 0 {
		return styles.Dim().Render("Cookies / Headers")
	}
	mode := func(label, id string, active bool) string {
		var style lipgloss.Style
		if active {
			style = lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true)
		} else {
			style = lipgloss.NewStyle().Foreground(styles.ColTextMuted)
		}
		return zone.Mark(id, style.Render(" "+label+" "))
	}
	active := 0
	if p.RawMode {
		active = 1
	}
	if p.PreviewMode {
		active = 2
	}
	sep := styles.Dim().Render("·")
	return lipgloss.JoinHorizontal(lipgloss.Center,
		mode("Pretty", "response_mode_pretty", active == 0),
		sep,
		mode("Raw", "response_mode_raw", active == 1),
		sep,
		mode("Preview", "response_mode_preview", active == 2),
	)
}

// filterField renders the right side of the toolbar row: either the gjson
// filter or the in-response text find field.
func (p *ResponsePanel) filterField() string {
	if !p.FilterFocued && (p.FindActive() || p.FindFocused) {
		label := styles.Dim().Render("find ")
		p.Find.SetWidth(clampFilterWidth(p.Find.Width()))
		return label + zone.Mark("response_find", p.Find.View())
	}
	label := styles.Dim().Render("gjson ")
	p.Filter.SetWidth(clampFilterWidth(p.Filter.Width()))
	return label + zone.Mark("response_filter", p.Filter.View())
}

// renderBody renders the viewport area, including the empty state card when
// there is no content yet. Returns exactly bodyHeight lines.
func (p *ResponsePanel) renderBody(innerWidth, bodyHeight int) string {
	p.vp.SetWidth(innerWidth)
	p.vp.SetHeight(bodyHeight)

	if !p.HasContent() && !p.Loading {
		return p.emptyStateCard(innerWidth, bodyHeight)
	}
	if p.Loading && p.Status == "" {
		return p.centeredHint(innerWidth, bodyHeight,
			lipgloss.NewStyle().Foreground(styles.ColInfo).Render(p.spinner.View())+
				styles.Muted().Render("  Sending request…"))
	}

	// Real content: enable the line-number gutter and fill the viewport.
	p.vp.LeftGutterFunc = lineNumberGutter
	rawLines := p.contentLines()
	bodyLines := make([]string, 0, len(rawLines))
	findQuery := strings.ToLower(strings.TrimSpace(p.Find.Value()))
	activeLine := p.activeMatchLine()
	for i, line := range rawLines {
		plain := sanitizeLine(ansi.Strip(line))
		switch {
		case p.selectedLine(i):
			bodyLines = append(bodyLines, styles.Selected().Render(plain))
		case findQuery != "" && strings.Contains(strings.ToLower(plain), findQuery):
			bodyLines = append(bodyLines, p.highlightLine(plain, i == activeLine))
		default:
			bodyLines = append(bodyLines, styles.Text().Render(line))
		}
	}
	p.vp.SetContentLines(bodyLines)
	p.ScrollOffset = p.vp.YOffset()

	view := p.vp.View()
	lines := strings.Split(view, "\n")

	// Overlay a scroll indicator + percentage when content overflows.
	overflows := p.vp.TotalLineCount() > bodyHeight
	if overflows {
		lines = p.overlayScrollbar(lines, bodyHeight)
	}

	for len(lines) < bodyHeight {
		lines = append(lines, "")
	}
	return strings.Join(lines[:bodyHeight], "\n")
}

// contentLines selects and prepares the raw text lines for the active tab/mode.
func (p *ResponsePanel) contentLines() []string {
	switch {
	case p.Err != "":
		return p.errorLines()
	case p.Tab == 1:
		if len(p.Cookies) == 0 {
			return []string{"(no cookies)"}
		}
		return sanitizeLines(p.Cookies)
	case p.Tab == 2:
		if len(p.Headers) == 0 {
			return []string{"(no headers)"}
		}
		return sanitizeLines(p.Headers)
	case p.RawMode:
		body := p.RawBody
		if body == "" {
			body = p.Body
		}
		if body == "" {
			return []string{"(empty response body)"}
		}
		return sanitizeLines(strings.Split(body, "\n"))
	case p.PreviewMode:
		return p.previewLines()
	default: // Pretty mode
		body := p.Body
		if body == "" {
			body = p.RawBody
		}
		if body == "" {
			return []string{"(empty response body)"}
		}
		filtered := p.applyFilter(body)
		pretty := formatResponseBody([]byte(filtered))
		return sanitizeLines(strings.Split(pretty, "\n"))
	}
}

// sanitizeLines menormalkan karakter kontrol dari response agar lebar baris
// yang dihitung sama dengan lebar yang benar-benar dicetak terminal.
func sanitizeLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = sanitizeLine(line)
	}
	return out
}

// sanitizeLine removes ANSI escapes, carriage returns, tabs, and remaining
// control characters. Tab diperluas agar tidak menggeser border saat dirender.
func sanitizeLine(line string) string {
	line = ansi.Strip(line)
	line = strings.ReplaceAll(line, "\r", "")
	line = strings.ReplaceAll(line, "\t", "    ")
	return strings.Map(func(r rune) rune {
		if r == '\n' || unicode.IsControl(r) {
			return -1
		}
		return r
	}, line)
}

// applyFilter runs the gjson filter against the JSON body when one is set.
func (p *ResponsePanel) applyFilter(body string) string {
	expr := strings.TrimSpace(p.Filter.Value())
	if expr == "" || !gjson.Valid(body) {
		return body
	}
	res := gjson.Get(body, expr)
	if !res.Exists() {
		return body
	}
	if res.IsObject() || res.IsArray() {
		return res.Raw
	}
	return res.String()
}

// previewLines renders a compact summary of the request/response metadata.
func (p *ResponsePanel) previewLines() []string {
	label := func(name, value string) string {
		return "  " + styles.Muted().Width(16).Render(name) + styles.Text().Render(value)
	}
	lines := []string{
		styles.Title(true).Render("Response overview"),
		label("Method", styles.MethodBadge(p.Method).Render(p.Method)),
		label("URL", sanitizeLine(p.URL)),
		label("Status", styles.StatusBadge(p.Code).Render(p.Status)),
		label("Content type", sanitizeLine(p.ContentType)),
		label("Body size", formatBytes(p.Size)),
		label("Duration", p.Time.Round(time.Millisecond).String()),
		label("Headers", fmt.Sprintf("%d", len(p.Headers))),
		label("Cookies", fmt.Sprintf("%d", len(p.Cookies))),
		label("Attempts", fmt.Sprintf("%d", max(p.Attempts, 1))),
		"",
		styles.Title(false).Render("Network timing"),
		label("DNS", formatDuration(p.Timings.DNS)),
		label("Connect", formatDuration(p.Timings.Connect)),
		label("TLS handshake", formatDuration(p.Timings.TLS)),
		label("First byte", formatDuration(p.Timings.FirstByte)),
		label("Total", formatDuration(p.Timings.Total)),
		"",
		styles.Title(false).Render("Redirects"),
	}
	if len(p.Redirects) == 0 {
		lines = append(lines, "  "+styles.Dim().Render("(none)"))
	} else {
		for _, redirect := range p.Redirects {
			lines = append(lines, "  "+styles.Text().Render("→ "+sanitizeLine(redirect)))
		}
	}

	lines = append(lines, "", styles.Bright().Render("TLS"))
	if p.TLS == nil {
		lines = append(lines, "  "+styles.Dim().Render("(not available)"))
	} else {
		lines = append(lines,
			"  "+styles.Dim().Render("Version       ")+styles.Text().Render(p.TLS.Version),
			"  "+styles.Dim().Render("Server        ")+styles.Text().Render(sanitizeLine(p.TLS.ServerName)),
			"  "+styles.Dim().Render("Issuer        ")+styles.Text().Render(sanitizeLine(p.TLS.Issuer)),
		)
		if !p.TLS.NotAfter.IsZero() {
			lines = append(lines, "  "+styles.Dim().Render("Expires       ")+styles.Text().Render(p.TLS.NotAfter.Format(time.RFC3339)))
		}
	}
	return lines
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	return d.Round(time.Millisecond).String()
}

// overlayScrollbar draws a thin scroll indicator on the right edge and the
// scroll percentage in the bottom-right corner.
func (p *ResponsePanel) overlayScrollbar(lines []string, height int) []string {
	track := "▕"
	thumb := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Render("▌")
	trackStyle := styles.Dim()

	total := p.vp.TotalLineCount()
	offset := p.vp.YOffset()
	maxOffset := max(total-height, 1)

	thumbPos := offset * (height - 1) / maxOffset
	thumbPos = min(thumbPos, height-1)
	thumbPos = max(thumbPos, 0)

	for i := 0; i < height && i < len(lines); i++ {
		line := lines[i]
		// Reserve the last column for the indicator.
		if w := lipgloss.Width(line); w < p.vp.Width() {
			line += strings.Repeat(" ", p.vp.Width()-w)
		}
		glyph := trackStyle.Render(track)
		if i == thumbPos {
			glyph = thumb
		}
		// Replace the final cell with the indicator cell.
		line = truncateAnsi(line, max(p.vp.Width()-1, 0)) + glyph
		lines[i] = line
	}

	// Percentage badge in the bottom-right corner.
	pct := int(p.vp.ScrollPercent() * 100)
	badge := styles.Dim().Render(fmt.Sprintf(" %d%% ", pct))
	if len(lines) > 0 {
		last := lines[len(lines)-1]
		w := lipgloss.Width(last)
		if lipgloss.Width(badge) <= w {
			lines[len(lines)-1] = truncateAnsi(last, w-lipgloss.Width(badge)) + badge
		}
	}
	return lines
}

// emptyStateCard renders a centered getting-started card with NO line-number
// gutter, replacing the old in-viewport empty state.
func (p *ResponsePanel) emptyStateCard(width, height int) string {
	key := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true)
	dim := styles.Dim()

	body := strings.Join([]string{
		styles.Bright().Bold(true).Render("No response yet"),
		"",
		dim.Render("Send your first request:"),
		"",
		key.Render("Enter") + dim.Render("   send request from the URL bar"),
		key.Render("Ctrl+N") + dim.Render("  new request"),
		key.Render("Ctrl+K") + dim.Render("  command palette"),
		"",
		dim.Render("Enter a URL to get started."),
	}, "\n")

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColBorder).
		Padding(1, 3).
		Render(body)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, card)
}

// centeredHint centers a single hint line (used while loading).
func (p *ResponsePanel) centeredHint(width, height int, text string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, text)
}

// errorLines renders a readable error with suggestions. Raw mode keeps the
// original Go error for debugging; other modes show a friendly explanation.
func (p *ResponsePanel) errorLines() []string {
	errorTitle := lipgloss.NewStyle().Foreground(styles.ColError).Bold(true)

	if p.RawMode {
		return []string{
			errorTitle.Render("Request failed (raw)"),
			"",
			"  " + styles.Text().Render(sanitizeLine(p.Err)),
		}
	}

	info := classifyError(p.Err)
	lines := []string{
		errorTitle.Render(info.Title),
		"",
		"  " + styles.Text().Render(info.Cause),
	}
	if len(info.Suggestions) > 0 {
		lines = append(lines, "", "  "+styles.Dim().Render("Suggestions:"))
		for _, s := range info.Suggestions {
			lines = append(lines, "    "+styles.Dim().Render("· "+s))
		}
	}
	lines = append(lines, "", "  "+styles.Dim().Render("Switch to Raw mode for technical details."))
	return lines
}

// truncateAnsi truncates an ANSI-styled string to width cells.
func truncateAnsi(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	// Lipgloss handles ANSI-aware truncation via MaxWidth on a rendered string.
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}

func clampFilterWidth(w int) int {
	if w <= 0 {
		return 24
	}
	if w > 40 {
		return 40
	}
	return w
}

func formatBytes(size int) string {
	if size < 1024 {
		return strconv.Itoa(size) + " B"
	}
	if size < 1024*1024 {
		return strconv.FormatFloat(float64(size)/1024, 'f', 1, 64) + " KB"
	}
	return strconv.FormatFloat(float64(size)/(1024*1024), 'f', 1, 64) + " MB"
}
