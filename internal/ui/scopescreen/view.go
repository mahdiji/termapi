package scopescreen

import (
	"fmt"
	"strings"

	"api/internal/ui/styles"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	leftPanelMin = 34
	leftPanelMax = 52
)

// View merender layar Global Scope: header, daftar origin di kiri, dan
// detail origin terpilih beserta ringkasan keamanan/penggunaan di kanan.
func (m *Model) View() string {
	w := max(m.Width, 100)
	h := max(m.Height, 28)

	header := m.headerView(w)
	status := m.statusView(w)

	headerH := 1
	statusH := 0
	if status != "" {
		statusH = 1
	}
	panelH := max(h-headerH-statusH-1, 6)

	leftW, rightW := layoutWidths(w)
	left := renderPanel(
		fmt.Sprintf("ALLOWED ORIGINS · %d", len(m.Origins)),
		m.originLines(panelH, leftW),
		leftW, panelH, true,
	)
	right := renderPanel(m.detailTitle(), m.detailLines(rightW), rightW, panelH, false)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)

	parts := []string{header}
	if status != "" {
		parts = append(parts, status)
	}
	parts = append(parts, panels, m.footerView(w))
	return strings.Join(parts, "\n")
}

// layoutWidths membagi lebar layar menjadi panel daftar (kiri) dan panel
// detail (kanan) dengan satu kolom jarak di antaranya.
func layoutWidths(w int) (leftW, rightW int) {
	leftW = min(max(w*36/100, leftPanelMin), leftPanelMax)
	rightW = max(w-leftW-1, 24)
	return leftW, rightW
}

func (m *Model) detailTitle() string {
	info := m.selectedInfo()
	if !info.Valid {
		return "ORIGIN DETAIL"
	}
	return "ORIGIN DETAIL · " + info.Host
}

// ── header / status / footer ────────────────────────────────────────────────

func (m *Model) headerView(w int) string {
	stats := buildStats(m.Origins)

	brand := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Padding(0, 1).
		Render("TermAPI")
	title := lipgloss.NewStyle().
		Foreground(styles.ColTextBright).
		Bold(true).
		Render("GLOBAL SCOPE")
	left := brand + " " + title

	sep := lipgloss.NewStyle().Foreground(styles.ColBorder).Render(" │ ")
	originLabel := "origins"
	if stats.Total == 1 {
		originLabel = "origin"
	}
	rightParts := []string{
		lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true).
			Render(fmt.Sprintf("%d %s", stats.Total, originLabel)),
	}
	if stats.HTTPS > 0 {
		rightParts = append(rightParts,
			lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true).
				Render(fmt.Sprintf("%d HTTPS", stats.HTTPS)))
	}
	if stats.HTTP > 0 {
		rightParts = append(rightParts,
			lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true).
				Render(fmt.Sprintf("%d HTTP", stats.HTTP)))
	}
	if stats.Invalid > 0 {
		rightParts = append(rightParts,
			lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).
				Render(fmt.Sprintf("%d invalid", stats.Invalid)))
	}
	rightParts = append(rightParts,
		lipgloss.NewStyle().Foreground(styles.ColTextDim).
			Render(".termapi/global.yaml"))

	return joinRow(left, strings.Join(rightParts, sep), w)
}

func (m *Model) statusView(w int) string {
	if m.Err != "" {
		return lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).
			Render(truncate("✗ "+m.Err, w))
	}
	if m.Message != "" {
		return lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true).
			Render(truncate("✓ "+m.Message, w))
	}
	return ""
}

func (m *Model) footerView(w int) string {
	keySt := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	descSt := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	sepSt := lipgloss.NewStyle().Foreground(styles.ColBorder)

	type shortcut struct {
		key  string
		desc string
	}

	var mode string
	var keys []shortcut
	if m.Adding {
		mode = " NEW ORIGIN "
		keys = []shortcut{
			{"enter", "save"},
			{"esc", "cancel"},
		}
	} else {
		mode = " GLOBAL SCOPE "
		keys = []shortcut{
			{"↑/↓", "navigate"},
			{"a", "add origin"},
			{"d", "delete"},
		}
	}

	modeBadge := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Render(mode)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, keySt.Render(k.key)+descSt.Render(" "+k.desc))
	}
	sep := sepSt.Render(" │ ")

	right := descSt.Render("esc close") + sepSt.Render(" │ ") + descSt.Render("ctrl+g close") + " "

	for {
		middle := ""
		if len(parts) > 0 {
			middle = " " + strings.Join(parts, sep)
		}
		gap := w - lipgloss.Width(modeBadge) - lipgloss.Width(middle) - lipgloss.Width(right)
		if gap >= 1 || len(parts) <= 1 {
			if gap < 0 {
				gap = 0
			}
			bar := modeBadge + middle + strings.Repeat(" ", gap) + right
			bar = ansi.Truncate(bar, w, "")
			return lipgloss.NewStyle().
				Background(styles.ColStatusBg).
				MaxHeight(1).
				MaxWidth(w).
				Render(bar)
		}
		parts = parts[:len(parts)-1]
	}
}

// ── left panel: daftar origin ───────────────────────────────────────────────

func (m *Model) originLines(panelH, panelW int) []string {
	innerW := max(panelW-4, 1)
	bodyRows := max(panelH-4, 1)

	text := lipgloss.NewStyle().Foreground(styles.ColText)
	muted := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
	dim := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	warn := lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true)
	errSt := lipgloss.NewStyle().Foreground(styles.ColError).Bold(true)
	accentHi := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)

	formRows := 0
	if m.Adding {
		formRows = 4
	}
	listRows := max(bodyRows-formRows, 1)

	lines := []string{}
	start, end := visibleWindow(m.Cursor, len(m.Origins), listRows)
	for i := start; i < end; i++ {
		raw := m.Origins[i]
		info := parseOrigin(raw)
		selected := i == m.Cursor

		prefix := "  "
		if selected {
			prefix = "▸ "
		}
		badge := schemeBadge(info)
		avail := innerW - lipgloss.Width(prefix) - lipgloss.Width(badge) - 1
		if avail < 4 {
			avail = 4
		}
		row := prefix + truncate(raw, avail) + " " + badge

		switch {
		case selected:
			lines = append(lines, styles.Selected().Width(innerW).Render(fit(row, innerW)))
		case !info.Valid:
			lines = append(lines, errSt.Render(fit(row, innerW)))
		default:
			lines = append(lines, text.Render(fit(row, innerW)))
		}
	}

	if len(m.Origins) == 0 && !m.Adding {
		lines = append(lines, warn.Render("No allowed origins yet."))
		lines = append(lines, muted.Render("Requests with tokens may be"))
		lines = append(lines, muted.Render("blocked until one is added."))
		lines = append(lines, "")
		lines = append(lines, dim.Render("Press 'a' to add an origin."))
	}

	if m.Adding {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, accentHi.Render("NEW ORIGIN"))
		lines = append(lines, truncate(m.Input.View(), innerW))
		lines = append(lines, muted.Render(truncate("[Enter] save · [Esc] cancel", innerW)))
	}

	return lines
}

func schemeBadge(info OriginInfo) string {
	if !info.Valid {
		return lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).
			Render("INVALID")
	}
	if info.Scheme == "https" {
		return lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true).
			Render("HTTPS")
	}
	return lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true).
		Render("HTTP")
}

// ── right panel: detail origin terpilih ─────────────────────────────────────

func (m *Model) selectedInfo() OriginInfo {
	if len(m.Origins) == 0 {
		return OriginInfo{}
	}
	cursor := m.Cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(m.Origins) {
		cursor = len(m.Origins) - 1
	}
	return parseOrigin(m.Origins[cursor])
}

func (m *Model) detailLines(panelW int) []string {
	innerW := max(panelW-4, 1)

	section := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
	dim := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	value := lipgloss.NewStyle().Foreground(styles.ColTextBright)
	success := lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true)
	warn := lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true)
	errSt := lipgloss.NewStyle().Foreground(styles.ColError).Bold(true)
	plain := lipgloss.NewStyle()

	field := func(label, val string, valSt lipgloss.Style) string {
		return fieldCell(label, val, innerW, muted, valSt)
	}
	grid := func(a, b string) []string { return gridLines(a, b, innerW) }
	rule := func() string {
		return lipgloss.NewStyle().Foreground(styles.ColBorder).
			Render(strings.Repeat("─", innerW))
	}

	info := m.selectedInfo()
	if len(m.Origins) == 0 {
		return []string{
			muted.Render("(no origin selected)"),
			"",
			section.Render("SECURITY & USAGE"),
			muted.Render("  Credentials (tokens, cookies, auth"),
			muted.Render("  headers) are sent only to origins"),
			muted.Render("  listed here."),
			"",
			dim.Render("  Press 'a' to add your first origin."),
		}
	}

	if !info.Valid {
		return []string{
			joinRow(errSt.Render("INVALID ORIGIN"), dim.Render("not usable as a scope"), innerW),
			rule(),
			"",
			field("VALUE", info.Raw, value),
			"",
			section.Render("EXPECTED FORMAT"),
			field("ORIGIN", "scheme://host[:port]", value),
			"",
			section.Render("EXAMPLES"),
			"  " + dim.Render("http://localhost:8080"),
			"  " + dim.Render("https://api.example.com"),
			"",
			section.Render("SECURITY & USAGE"),
			muted.Render("  Credentials (tokens, cookies, auth"),
			muted.Render("  headers) are sent only to origins"),
			muted.Render("  listed here."),
			"",
			dim.Render("  Press 'd' to remove this entry."),
		}
	}

	lines := []string{
		joinRow(value.Bold(true).Render(truncate(info.Raw, max(innerW-12, 8))), schemeBadge(info), innerW),
		rule(),
		"",
		section.Render("PARSED"),
	}
	lines = append(lines, grid(
		field("SCHEME", info.Scheme, value),
		field("PORT", info.Port, value),
	)...)
	lines = append(lines, grid(
		field("HOST", info.Host, value),
		field("PATH", info.Path, value),
	)...)
	lines = append(lines,
		"",
		section.Render("CLASSIFICATION"),
	)
	lines = append(lines, grid(
		field("TYPE", typeBadge(info.Type), plain),
		field("RISK", riskBadge(info.Risk), plain),
	)...)
	lines = append(lines,
		field("STATUS", success.Render("TRUSTED"), plain),
		"",
		rule(),
		"",
		section.Render("SECURITY & USAGE"),
		muted.Render("  Credentials (tokens, cookies, auth"),
		muted.Render("  headers) are sent only to origins"),
		muted.Render("  listed here. Matching is exact"),
		muted.Render("  scheme + host + port."),
	)
	lines = append(lines, transportHint(info, muted, warn)...)
	return append(lines,
		"",
		section.Render("SUMMARY"),
		statStrip(buildStats(m.Origins)),
	)
}

func typeBadge(hostType string) string {
	switch hostType {
	case "LOCAL":
		return lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true).Render("LOCAL")
	case "LAN":
		return lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true).Render("LAN")
	case "PUBLIC":
		return lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true).Render("PUBLIC")
	default:
		return lipgloss.NewStyle().Foreground(styles.ColTextMuted).Render("UNKNOWN")
	}
}

func riskBadge(risk string) string {
	switch risk {
	case "LOW":
		return lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true).Render("LOW")
	case "MEDIUM":
		return lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true).Render("MEDIUM")
	case "HIGH":
		return lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).Render("HIGH")
	default:
		return lipgloss.NewStyle().Foreground(styles.ColTextMuted).Render("—")
	}
}

func transportHint(info OriginInfo, muted, warn lipgloss.Style) []string {
	if info.Scheme == "https" {
		return []string{
			muted.Render("  Encrypted transport (HTTPS)."),
		}
	}
	if info.Type == "PUBLIC" {
		return []string{
			warn.Render("  Plaintext HTTP on a public host is interceptable."),
		}
	}
	return []string{
		muted.Render("  Plaintext HTTP; acceptable for local/LAN hosts."),
	}
}

// statStrip merangkum jumlah origin dalam satu baris ringkas.
func statStrip(stats ScopeStats) string {
	chip := func(label string, n int, st lipgloss.Style) string {
		return lipgloss.NewStyle().Foreground(styles.ColTextMuted).Render(label+" ") +
			st.Render(fmt.Sprintf("%d", n))
	}
	bright := lipgloss.NewStyle().Foreground(styles.ColTextBright).Bold(true)
	ok := lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true)
	bad := lipgloss.NewStyle().Foreground(styles.ColError).Bold(true)
	warn := lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true)

	parts := []string{chip("Origins", stats.Total, bright)}
	if stats.HTTPS > 0 {
		parts = append(parts, chip("HTTPS", stats.HTTPS, ok))
	}
	if stats.HTTP > 0 {
		parts = append(parts, chip("HTTP", stats.HTTP, warn))
	}
	if stats.Invalid > 0 {
		parts = append(parts, chip("Invalid", stats.Invalid, bad))
	}
	return "  " + strings.Join(parts, "  ·  ")
}

// ── panel helper ────────────────────────────────────────────────────────────

func renderPanel(title string, lines []string, width, height int, focused bool) string {
	width = max(width, 8)
	height = max(height, 4)

	innerW := max(width-4, 1)  // border (2) + horizontal padding (2)
	innerH := max(height-2, 2) // border top/bottom
	bodyRows := max(innerH-2, 0)

	titleSt := lipgloss.NewStyle().Foreground(styles.ColTextMuted).Bold(true)
	if focused {
		titleSt = lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	}
	sep := lipgloss.NewStyle().Foreground(styles.ColBorder).Render(strings.Repeat("─", innerW))

	out := make([]string, 0, bodyRows+2)
	out = append(out, titleSt.Render(fit(title, innerW)), sep)
	for i := 0; i < bodyRows; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		out = append(out, fit(line, innerW))
	}

	style := lipgloss.NewStyle().
		Border(styles.PanelBorder).
		BorderForeground(styles.ColBorder).
		Background(styles.ColSurface).
		Padding(0, 1).
		Width(width).
		Height(height)
	if focused {
		style = style.BorderForeground(styles.ColBorderFocus)
	}
	return style.Render(strings.Join(out, "\n"))
}

// ── util ────────────────────────────────────────────────────────────────────

// fieldCell merender satu pasangan label/nilai dengan label selebar tetap.
func fieldCell(label, val string, width int, labelSt, valSt lipgloss.Style) string {
	const labelW = 8
	avail := max(width-2-labelW, 4)
	return "  " + labelSt.Render(padRight(label, labelW)) + valSt.Render(truncate(val, avail))
}

// gridLines menyusun dua field berdampingan, atau bertumpuk bila panel sempit.
func gridLines(a, b string, width int) []string {
	if b == "" {
		return []string{a}
	}
	if width < 44 {
		return []string{a, b}
	}
	colW := (width - 2) / 2
	return []string{fit(a, colW) + "  " + fit(b, width-colW-2)}
}

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

// padRight pads s with spaces until it occupies at least w terminal cells.
func padRight(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

// fit truncates or pads s to exactly w terminal cells.
func fit(s string, w int) string {
	return padRight(truncate(s, w), w)
}

// visibleWindow returns the slice [start, end) that keeps cursor on screen.
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
	maxRight := max(width-lipgloss.Width(left)-1, 1)
	right = truncate(right, maxRight)
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}
