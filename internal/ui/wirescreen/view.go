package wirescreen

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"api/internal/ui/styles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func fmtRel(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalMs := d.Milliseconds()
	ms := totalMs % 1000
	totalSec := totalMs / 1000
	sec := totalSec % 60
	min := totalSec / 60
	return fmt.Sprintf("%02d:%02d.%03d", min, sec, ms)
}

func fmtUptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, (s%3600)/60, s%60)
}

func dirLabel(d Direction) string {
	switch d {
	case DirTX:
		return "TX"
	case DirRX:
		return "RX"
	case DirControl:
		return "CTRL"
	case DirError:
		return "ERR"
	default:
		return "NONE"
	}
}

func dirArrow(d Direction) string {
	switch d {
	case DirTX:
		return "▸"
	case DirRX:
		return "◂"
	case DirControl:
		return "●"
	case DirError:
		return "✕"
	default:
		return "·"
	}
}

func dirColor(d Direction) color.Color {
	switch d {
	case DirTX:
		return styles.ColInfo
	case DirRX:
		return styles.ColSuccess
	case DirControl:
		return styles.ColWarning
	case DirError:
		return styles.ColError
	default:
		return styles.ColTextDim
	}
}

// View merender websocket/gRPC monitor sebagai TUI split-pane modern:
// header statistik, event stream, inspector opsional, dan composer besar.
func (m *Model) View() tea.View {
	w := m.Width
	if w <= 0 {
		w = 100
	}
	h := m.Height
	if h <= 0 {
		h = 30
	}

	if w < 70 || h < 20 {
		msg := lipgloss.NewStyle().
			Foreground(styles.ColError).Bold(true).
			Render(fmt.Sprintf("⚠  Window terlalu kecil (%dx%d). Minimum 70x20.", w, h))
		return tea.NewView(lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg))
	}

	// Lebar konten di dalam border (border 1 kolom tiap sisi, 1 padding).
	innerW := w - 4

	events := m.Ring.Slice()

	// Filter.
	var filteredEvents []Event
	hiddenCount := 0
	for _, ev := range events {
		if m.FilterQuery != "" && !strings.Contains(strings.ToLower(ev.Payload), strings.ToLower(m.FilterQuery)) {
			hiddenCount++
			continue
		}
		filteredEvents = append(filteredEvents, ev)
	}

	// --- Perhitungan tinggi panel ---
	// Layout: header(2) + stream(N) + [inspector(overhead)] + composer(composerBox) + statusbar(1)
	composerHeight := max(5, m.Height/4)
	composerBoxHeight := composerHeight + 1 // baris isi + border bawah
	headerHeight := 2                       // stats + judul stream
	streamMin := 4

	inspectorHeight := 0
	inspectorTitleH := 0
	if m.InspectorOpen {
		inspectorHeight = max(4, m.Height/6)
		inspectorTitleH = 1
	}

	// Semua baris non-stream: stats, judul stream, judul+isi inspector,
	// judul composer, isi composer, dan status bar.
	fixed := headerHeight + inspectorTitleH + inspectorHeight +
		1 /*composer title*/ + composerBoxHeight + 1 /*status bar*/

	// contentH adalah tinggi isi di dalam border luar (border atas+bawah).
	contentH := h - 2
	streamHeight := contentH - fixed
	if streamHeight < streamMin {
		need := streamMin - streamHeight
		if inspectorHeight > 0 {
			take := min(need, inspectorHeight-3)
			inspectorHeight -= take
			need -= take
		}
		if need > 0 && composerHeight > 4 {
			take := min(need, composerHeight-4)
			composerHeight -= take
			composerBoxHeight = composerHeight + 1
			need -= take
		}
		streamHeight = contentH - fixed
		if streamHeight < streamMin {
			streamHeight = streamMin
		}
	}

	// --- Scroll bounds ---
	if m.Scroll >= len(filteredEvents) {
		m.Scroll = max(0, len(filteredEvents)-1)
	}
	start := max(0, len(filteredEvents)-streamHeight)
	if !m.AutoFollow && !m.Scrubbing {
		start = max(0, min(m.Scroll, len(filteredEvents)-streamHeight))
	}
	end := min(len(filteredEvents), start+streamHeight)
	visibleEvents := filteredEvents[start:end]

	// --- Header ---
	tpSpark := sparkline(m.ThroughputSpark.vals, 16)
	rttPart := styles.Dim().Render("RTT –")
	if m.Protocol == ProtoWS && m.Client != nil {
		if rtt := m.Client.RTT(); rtt > 0 {
			m.LastRTT = rtt
		}
	}
	if m.LastRTT > 0 {
		rttSpark := sparkline(m.RttSpark.vals, 16)
		rttPart = styles.Text().Render("RTT ") +
			lipgloss.NewStyle().Foreground(styles.ColInfo).Render(rttSpark) +
			styles.Text().Render(fmt.Sprintf(" %dms", m.LastRTT.Milliseconds()))
	}
	headerInfo := styles.Dim().Render("TP ") +
		lipgloss.NewStyle().Foreground(styles.ColSuccess).Render(tpSpark) +
		styles.Text().Render(fmt.Sprintf(" %.1f/s", m.LastThrough)) +
		styles.Dim().Render("   ") + rttPart

	// --- Stream rows ---
	var logRows []string
	for idx, ev := range visibleEvents {
		absoluteIdx := start + idx
		isSelected := absoluteIdx == m.Scroll

		timeStr := fmtRel(ev.Elapsed)
		if m.AbsoluteTime {
			timeStr = ev.Timestamp.Format("15:04:05.000")
		}

		color := dirColor(ev.Direction)
		bar := lipgloss.NewStyle().Foreground(color).Render("▌")
		arrow := lipgloss.NewStyle().Foreground(color).Bold(true).Render(dirArrow(ev.Direction))
		label := lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%-4s", dirLabel(ev.Direction)))
		ts := styles.Dim().Render(timeStr)

		maxPayloadLen := innerW - 26
		if maxPayloadLen < 10 {
			maxPayloadLen = 10
		}
		payloadSummary := ev.Payload
		if len([]rune(payloadSummary)) > maxPayloadLen {
			payloadSummary = string([]rune(payloadSummary)[:max(1, maxPayloadLen-1)]) + "…"
		}

		statusBadge := styles.Muted().Render(fmt.Sprintf(" [%s]", ev.Status))
		payloadText := styles.Text().Render(payloadSummary)

		rowLine := bar + " " + ts + " " + label + " " + arrow + " " + payloadText
		if ev.Status != "" {
			rowLine += statusBadge
		}
		if isSelected {
			rowLine = lipgloss.NewStyle().Background(styles.ColSelectionBg).Render(rowLine)
		}
		logRows = append(logRows, fitLine(rowLine, innerW))
	}

	for len(logRows) < streamHeight {
		logRows = append(logRows, strings.Repeat(" ", innerW))
	}
	if len(logRows) > streamHeight {
		logRows = logRows[:streamHeight]
	}

	// --- Composer ---
	m.Composer.SetWidth(max(20, innerW-4))
	m.Composer.SetHeight(composerHeight)

	// --- Susun isi body (di dalam border) ---
	var body []string
	body = append(body, fitLine(headerInfo, innerW))
	body = append(body, fitLine(streamHeader(hiddenCount, len(filteredEvents), len(events)), innerW))
	body = append(body, logRows...)

	if m.InspectorOpen {
		body = append(body, fitLine(inspectorTitle(), innerW))
		var selEv *Event
		if len(filteredEvents) > 0 && m.Scroll >= 0 && m.Scroll < len(filteredEvents) {
			selEv = &filteredEvents[m.Scroll]
		}
		inspLines := formatInspector(selEv, innerW, inspectorHeight)
		for i := 0; i < inspectorHeight; i++ {
			if i < len(inspLines) {
				body = append(body, fitLine(styles.Text().Render(inspLines[i]), innerW))
			} else {
				body = append(body, strings.Repeat(" ", innerW))
			}
		}
	}

	body = append(body, fitLine(composerTitle(), innerW))
	composerBorder := styles.BorderStyle(m.Focus == FocusComposer)
	composerLines := strings.Split(m.Composer.View(), "\n")
	for _, l := range composerLines {
		body = append(body, composerBorder.Render("│ ")+fitLine(l, max(1, innerW-4))+composerBorder.Render(" │"))
	}
	body = append(body, composerBorder.Render("╰"+strings.Repeat("─", max(1, innerW-4))+"╯"))

	// --- Status bar (di dalam border) ---
	body = append(body, fitLine(m.statusBar(), innerW))

	// --- Outer border dengan judul ---
	titleColor := styles.ColSuccess
	if !m.Connected {
		titleColor = styles.ColError
	}
	title := lipgloss.NewStyle().Foreground(titleColor).Bold(true).Render("● "+protoLabel(m.Protocol)) +
		styles.Dim().Render("  "+truncate(m.URL, max(10, innerW-40))) + "  " +
		statusBadgeStyle(m.StatusText).Render(m.StatusText) + "  " +
		styles.Muted().Render(fmt.Sprintf("%d msg · %d dropped · up %s",
			m.Ring.Len(), m.Ring.Dropped(), fmtUptime(time.Since(m.StartedAt))))

	box := outerBox(title, strings.Join(body, "\n"), w, h)

	p := tea.NewView(box)
	p.AltScreen = true
	return p
}

func streamHeader(hidden, shown, total int) string {
	left := styles.Title(false).Bold(true).Render(" EVENT STREAM ")
	info := styles.Dim().Render(fmt.Sprintf("  showing %d/%d", shown, total))
	if hidden > 0 {
		info += lipgloss.NewStyle().Foreground(styles.ColWarning).Render(fmt.Sprintf("   ⌀ %d hidden by filter", hidden))
	}
	return left + info
}

func inspectorTitle() string {
	return styles.Title(false).Bold(true).Render(" INSPECTOR ") + styles.Dim().Render("  (Tab untuk tutup)")
}

func composerTitle() string {
	return styles.Title(false).Bold(true).Render(" COMPOSER ") +
		styles.Dim().Render("  Ctrl+Enter: kirim   Tab: pindah fokus")
}

func protoLabel(p ProtocolType) string {
	if p == ProtoGRPC {
		return "gRPC STREAM"
	}
	return "WEBSOCKET"
}

func statusBadgeStyle(s string) lipgloss.Style {
	st := styles.Muted()
	switch {
	case strings.Contains(s, "OPEN"):
		st = lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true)
	case strings.Contains(s, "CLOSED"):
		st = lipgloss.NewStyle().Foreground(styles.ColError).Bold(true)
	case strings.Contains(s, "CONNECT"):
		st = lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true)
	}
	return st
}

func (m *Model) statusBar() string {
	focus := func(name string, area FocusArea) string {
		if m.Focus == area {
			return lipgloss.NewStyle().
				Foreground(styles.ColBackground).
				Background(styles.ColAccentHi).Bold(true).
				Padding(0, 1).Render(name)
		}
		return styles.Dim().Padding(0, 1).Render(name)
	}
	tabs := focus("LOG", FocusLog) + focus("EDITOR", FocusComposer)
	if m.InspectorOpen {
		tabs += focus("INSPECT", FocusInspector)
	}

	var right string
	switch {
	case m.FilterActive:
		right = lipgloss.NewStyle().Foreground(styles.ColWarning).Render("/ filter: " + m.FilterQuery + "█")
	case m.Scrubbing:
		right = lipgloss.NewStyle().Foreground(styles.ColInfo).Render(
			fmt.Sprintf("scrub [%d/%d] — G resume live", m.ScrubIndex+1, m.Ring.Len()))
	case m.ExportStatus != "":
		right = styles.Muted().Render(m.ExportStatus)
	default:
		right = styles.Dim().Render(m.LatestTick)
	}
	if m.AutoFollow {
		right = lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true).Render("● LIVE  ") + right
	} else {
		right = lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true).Render("⏸ PAUSED  ") + right
	}
	return tabs + "  " + right
}

func truncate(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen-1]) + "…"
}

// outerBox menggambar border membulat dengan judul tertanam di kiri atas.
func outerBox(title, body string, width, height int) string {
	if width < 12 {
		width = 12
	}
	if height < 4 {
		height = 4
	}
	border := styles.BorderStyle(true)

	titleW := lipgloss.Width(title)
	innerW := width - 2
	topBar := max(0, innerW-3-titleW)
	top := border.Render("╭─ ") + title + border.Render(" "+strings.Repeat("─", topBar)+"╮")

	lines := strings.Split(body, "\n")
	contentH := height - 2
	var out []string
	out = append(out, top)
	for i := 0; i < contentH; i++ {
		var line string
		if i < len(lines) {
			line = fitLine(lines[i], innerW-2)
		} else {
			line = strings.Repeat(" ", innerW-2)
		}
		out = append(out, border.Render("│ ")+line+border.Render(" │"))
	}
	out = append(out, border.Render("╰"+strings.Repeat("─", innerW)+"╯"))
	return strings.Join(out, "\n")
}
