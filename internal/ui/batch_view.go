package ui

import (
	"fmt"
	"image/color"
	"net/url"
	"strings"
	"time"

	"api/internal/ui/styles"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// ---------------------------------------------------------------------------
// Kerangka panel
// ---------------------------------------------------------------------------

// batchFrameTop merender border atas dengan judul di kiri dan meta di kanan.
func batchFrameTop(inner int, title, right string, border lipgloss.Style) string {
	titleText := " " + title + " "
	rightText := ""
	if right != "" {
		rightText = " " + right + " "
	}
	fill := inner - 1 - lipgloss.Width(titleText) - lipgloss.Width(rightText)
	if fill < 1 {
		rightText = ""
		fill = inner - 1 - lipgloss.Width(titleText)
	}
	if fill < 1 {
		titleText = ansi.Truncate(titleText, max(inner-2, 0), "")
		fill = max(inner-1-lipgloss.Width(titleText), 0)
	}
	return border.Render("╭─") +
		styles.Title(true).Render(titleText) +
		border.Render(strings.Repeat("─", fill)) +
		rightText +
		border.Render("╮")
}

// batchFrameLine membungkus satu baris konten dengan border vertikal.
func batchFrameLine(content string, inner int, border lipgloss.Style) string {
	content = ansi.Truncate(content, inner, "…")
	if w := lipgloss.Width(content); w < inner {
		content += strings.Repeat(" ", inner-w)
	}
	return border.Render("│") + content + border.Render("│")
}

// batchFrameSep merender garis pemisah horizontal di dalam panel.
func batchFrameSep(inner int, border lipgloss.Style) string {
	return border.Render("├" + strings.Repeat("─", inner) + "┤")
}

func batchFrameBottom(inner int, border lipgloss.Style) string {
	return border.Render("╰" + strings.Repeat("─", inner) + "╯")
}

// batchSplitLine menaruh teks kiri dan kanan pada satu baris dengan jarak fleksibel.
func batchSplitLine(left, right string, inner int) string {
	gap := inner - lipgloss.Width(left) - lipgloss.Width(right) - 1
	if gap < 1 || right == "" {
		return left
	}
	return left + strings.Repeat(" ", gap) + right + " "
}

// ---------------------------------------------------------------------------
// Modal konfigurasi batch
// ---------------------------------------------------------------------------

func (m AppModel) batchChooserView() string {
	boxWidth := 76
	if m.ScreenWidth > 0 && boxWidth > m.ScreenWidth-4 {
		boxWidth = max(m.ScreenWidth-4, 48)
	}
	inner := boxWidth - 2

	border := lipgloss.NewStyle().Foreground(styles.ColBorderFocus)
	labelStyle := styles.Muted()
	valStyle := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true)
	hintStyle := styles.Dim()
	sectionStyle := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)

	statusChip := lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true).Render(" READY ")
	if m.BatchError != "" {
		statusChip = lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).Render(" CHECK ")
	}

	depthText := "folder ini saja"
	if m.BatchDepth > 0 {
		depthText = fmt.Sprintf("%d level ke bawah", m.BatchDepth)
	}
	depthValue := hintStyle.Render("hanya folder ini")
	if m.BatchMaxDepth > 0 {
		depthDots := make([]string, 0, m.BatchMaxDepth+1)
		for i := 0; i <= m.BatchMaxDepth; i++ {
			if i <= m.BatchDepth {
				depthDots = append(depthDots, lipgloss.NewStyle().Foreground(styles.ColAccent).Render("●"))
				continue
			}
			depthDots = append(depthDots, lipgloss.NewStyle().Foreground(styles.ColBorder).Render("○"))
		}
		depthValue = "◂ " + strings.Join(depthDots, " ") + " ▸"
	}

	modeText := "PARALEL"
	modeHint := "semua request dikirim bersamaan"
	if m.BatchChain {
		modeText = "CHAINING"
		modeHint = "berurutan, variabel response ditangkap"
	}
	retryText := "OFF"
	if m.BatchRetries > 0 {
		retryText = fmt.Sprintf("%d×", m.BatchRetries)
	}

	rows := []string{
		"",
		"  " + sectionStyle.Render("TARGET"),
		"  " + batchBreadcrumb(m.BatchFolder),
		"",
		"  " + labelStyle.Render(fmt.Sprintf("%-10s", "KEDALAMAN")) + "  " + depthValue + "   " + hintStyle.Render(depthText),
		"  " + labelStyle.Render(fmt.Sprintf("%-10s", "MODE")) + "  " + valStyle.Render(fmt.Sprintf("%-9s", modeText)) + "   " + hintStyle.Render("C · "+modeHint),
		"  " + labelStyle.Render(fmt.Sprintf("%-10s", "RETRY")) + "  " + valStyle.Render(fmt.Sprintf("%-9s", retryText)) + "   " + hintStyle.Render("R · ulangi request gagal"),
		"",
	}

	if m.BatchError != "" {
		rows = append(rows, "  "+lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).Render("! "+m.BatchError), "")
	}

	lines := []string{batchFrameTop(inner, "BATCH RUNNER", statusChip, border)}
	for _, row := range rows {
		lines = append(lines, batchFrameLine(row, inner, border))
	}
	lines = append(lines, batchFrameSep(inner, border))
	lines = append(lines, batchFrameLine("", inner, border))
	lines = append(lines, batchFrameLine("  "+
		lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true).Render("ENTER")+
		styles.Text().Render(" jalankan   ·   ")+
		lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true).Render("ESC")+
		styles.Text().Render(" batal   ·   ")+
		lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true).Render("←/→")+
		styles.Text().Render(" kedalaman"), inner, border))
	lines = append(lines, batchFrameLine("", inner, border))
	lines = append(lines, batchFrameBottom(inner, border))

	box := strings.Join(lines, "\n")
	return lipgloss.Place(m.ScreenWidth, m.ScreenHeight, lipgloss.Center, lipgloss.Center, box)
}

// batchBreadcrumb memecah path folder menjadi breadcrumb beraksen.
func batchBreadcrumb(folder string) string {
	if folder == "" {
		return "—"
	}
	parts := strings.Split(folder, "/")
	out := make([]string, 0, len(parts)*2)
	sep := styles.Dim().Render(" › ")
	for i, part := range parts {
		if i > 0 {
			out = append(out, sep)
		}
		if i == len(parts)-1 {
			out = append(out, styles.Bright().Bold(true).Render(part))
			continue
		}
		out = append(out, styles.Muted().Render(part))
	}
	return strings.Join(out, "")
}

// ---------------------------------------------------------------------------
// Hasil eksekusi batch
// ---------------------------------------------------------------------------

// batchCell adalah satu sel tabel hasil batch.
type batchCell struct {
	text  string
	fg    color.Color
	bold  bool
	right bool
}

func (m AppModel) batchView() string {
	width := max(m.ScreenWidth, 60)
	inner := width - 2
	border := lipgloss.NewStyle().Foreground(styles.ColBorder)

	passed, failed, waiting := 0, 0, 0
	var total, slowest time.Duration
	slowestName := ""
	for i, r := range m.BatchResults {
		if r.RequestName == "" && r.URL == "" {
			waiting++
			continue
		}
		if r.Err != "" || r.Code >= 400 {
			failed++
		} else {
			passed++
		}
		total += r.Duration
		if r.Duration > slowest {
			slowest = r.Duration
			slowestName = m.batchNameAt(i)
		}
	}
	done := passed + failed

	stateText := "SELESAI"
	stateColor := styles.ColSuccess
	if waiting > 0 {
		stateText = "BERJALAN"
		stateColor = styles.ColWarning
	}
	if failed > 0 && waiting == 0 {
		stateText = "GAGAL"
		stateColor = styles.ColError
	}
	stateChip := lipgloss.NewStyle().Foreground(stateColor).Bold(true).Render("● " + stateText)

	modeText := "paralel"
	if m.BatchChain {
		modeText = "chained"
	}
	config := []string{modeText}
	if m.BatchRetries > 0 {
		config = append(config, fmt.Sprintf("%d× retry", m.BatchRetries))
	}
	config = append(config, fmt.Sprintf("%d request", m.BatchExpected))
	configText := "  " + styles.Muted().Render(strings.Join(config, styles.Dim().Render("  ·  ")))
	countInfo := styles.Muted().Render(fmt.Sprintf("%d/%d selesai", done, m.BatchExpected))

	lines := []string{batchFrameTop(inner, "RUN COLLECTION", stateChip, border)}
	lines = append(lines, batchFrameLine("  "+styles.Bright().Bold(true).Render(m.BatchFolder), inner, border))
	lines = append(lines, batchFrameLine(batchSplitLine(configText, countInfo, inner), inner, border))
	lines = append(lines, batchFrameLine("", inner, border))

	ratio := 0.0
	if m.BatchExpected > 0 {
		ratio = float64(done) / float64(m.BatchExpected)
	}
	barWidth := min(max(inner/3, 18), 46)
	filled := int(ratio * float64(barWidth))
	bar := lipgloss.NewStyle().Foreground(styles.ColSuccess).Render(strings.Repeat("━", filled)) +
		styles.Dim().Render(strings.Repeat("─", max(barWidth-filled, 0)))
	percent := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true).Render(fmt.Sprintf("%3.0f%%", ratio*100))

	chips := []string{
		lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true).Render(fmt.Sprintf("✓ %d", passed)) + styles.Dim().Render(" lulus"),
		lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).Render(fmt.Sprintf("× %d", failed)) + styles.Dim().Render(" gagal"),
		styles.Muted().Render(fmt.Sprintf("◷ %d", waiting)) + styles.Dim().Render(" menunggu"),
	}
	progressLeft := "  " + bar + "  " + percent + "   " + strings.Join(chips, styles.Dim().Render("   "))
	lines = append(lines, batchFrameLine(progressLeft, inner, border))

	metrics := []string{}
	if done > 0 {
		avg := total / time.Duration(done)
		metrics = append(metrics, styles.Muted().Render("rata-rata ")+styles.Text().Render(batchDuration(avg)))
	}
	if slowest > 0 {
		label := "terlambat "
		if slowestName != "" {
			label = "terlambat " + ansi.Truncate(slowestName, 24, "…") + " "
		}
		metrics = append(metrics, styles.Muted().Render(label)+styles.Text().Render(batchDuration(slowest)))
	}
	if total > 0 {
		metrics = append(metrics, styles.Muted().Render("akumulasi ")+styles.Text().Render(batchDuration(total)))
	}
	if len(metrics) > 0 {
		lines = append(lines, batchFrameLine("  "+strings.Join(metrics, styles.Dim().Render("  ·  ")), inner, border))
	}

	if m.BatchError != "" {
		lines = append(lines, batchFrameLine("  "+lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).Render("! "+m.BatchError), inner, border))
	}

	lines = append(lines, batchFrameSep(inner, border))

	showHost := inner >= 96
	showRetry := inner >= 72
	gutterW, statusW, methodW, timeW, retryW := 6, 8, 8, 9, 5
	if !showRetry {
		retryW = 0
	}
	fixed := gutterW + statusW + methodW + timeW + retryW + 1
	nameW := min(max(inner*35/100, 22), 56)
	hostW := 0
	if showHost {
		hostW = inner - fixed - nameW
	}
	if !showHost || hostW < 16 {
		showHost = false
		hostW = 0
		nameW = max(inner-fixed, 16)
	}

	widths := []int{gutterW, statusW, methodW, nameW}
	header := []batchCell{
		{text: "#", fg: styles.ColTextDim, right: true},
		{text: "STATUS", fg: styles.ColTextDim},
		{text: "METHOD", fg: styles.ColTextDim},
		{text: "REQUEST", fg: styles.ColTextDim},
	}
	if showHost {
		widths = append(widths, hostW)
		header = append(header, batchCell{text: "TARGET", fg: styles.ColTextDim})
	}
	widths = append(widths, timeW)
	header = append(header, batchCell{text: "TIME", fg: styles.ColTextDim, right: true})
	if showRetry {
		widths = append(widths, retryW)
		header = append(header, batchCell{text: "TRY", fg: styles.ColTextDim, right: true})
	}
	lines = append(lines, batchFrameLine(renderBatchRow(header, widths, false, true), inner, border))

	maxRows := min(max(m.ScreenHeight/3, 4), 14)
	start := 0
	if m.BatchSelected >= maxRows {
		start = m.BatchSelected - maxRows + 1
	}
	end := min(start+maxRows, len(m.BatchResults))

	if len(m.BatchResults) == 0 {
		lines = append(lines, batchFrameLine("", inner, border))
		lines = append(lines, batchFrameLine("  "+styles.Dim().Render("Menunggu response pertama…"), inner, border))
	} else {
		for i := start; i < end; i++ {
			lines = append(lines, batchFrameLine(m.batchRow(i, widths, showHost, showRetry), inner, border))
		}
		// Selama batch berjalan, tinggi panel dikunci ke jumlah request yang
		// diharapkan supaya layout tidak melompat tiap response masuk.
		padded := min(max(m.BatchExpected, end-start), maxRows)
		for i := end - start; i < padded; i++ {
			lines = append(lines, batchFrameLine("", inner, border))
		}
	}

	lines = append(lines, batchFrameSep(inner, border))

	rangeText := "—"
	if len(m.BatchResults) > 0 {
		rangeText = fmt.Sprintf("%d–%d dari %d", start+1, end, len(m.BatchResults))
	}
	footerLeft := "  " + styles.Dim().Render("baris "+rangeText)
	footerRight := ""
	if len(m.BatchResults) > maxRows {
		footerRight = styles.Dim().Render("↑/↓ scroll") + "  "
	}
	lines = append(lines, batchFrameLine(batchSplitLine(footerLeft, footerRight, inner), inner, border))
	lines = append(lines, batchFrameBottom(inner, border))

	return strings.Join(lines, "\n")
}

// batchRow merender satu baris tabel hasil batch.
func (m AppModel) batchRow(i int, widths []int, showHost, showRetry bool) string {
	r := m.BatchResults[i]
	name := m.batchNameAt(i)
	method := ""
	if i < len(m.BatchSpecs) {
		method = m.BatchSpecs[i].Request.Method
	}
	if r.Method != "" {
		method = r.Method
	}
	if method == "" {
		method = "—"
	}

	pending := r.RequestName == "" && r.URL == ""
	statusText := "WAIT"
	statusColor := styles.ColTextDim
	timeText := "—"
	retryText := "—"
	targetText := ""
	targetColor := styles.ColTextDim

	if !pending {
		switch {
		case r.Err != "":
			statusText = "ERR"
			statusColor = styles.ColError
			targetText = r.Err
			targetColor = styles.ColError
		case r.Code > 0:
			statusText = fmt.Sprintf("%d", r.Code)
			statusColor = styles.StatusColor(r.Code)
		}
		if r.Duration > 0 {
			timeText = batchDuration(r.Duration)
		}
		retryText = fmt.Sprintf("%d×", max(r.Attempts, 1))
		if targetText == "" {
			targetText = batchTarget(r.URL)
		}
	}

	cursor := "  "
	if i == m.BatchSelected {
		cursor = "▸ "
	}
	cells := []batchCell{
		{text: cursor + fmt.Sprintf("%d", i+1), fg: styles.ColTextDim, right: true},
		{text: statusText, fg: statusColor, bold: true},
		{text: method, fg: styles.MethodColor(method), bold: true},
		{text: name, fg: styles.ColText},
	}
	if showHost {
		cells = append(cells, batchCell{text: targetText, fg: targetColor})
	}
	cells = append(cells, batchCell{text: timeText, fg: styles.ColTextMuted, right: true})
	if showRetry {
		cells = append(cells, batchCell{text: retryText, fg: styles.ColTextDim, right: true})
	}

	return renderBatchRow(cells, widths, i == m.BatchSelected, false)
}

// renderBatchRow menyusun sel menjadi satu baris selebar panel. Latar seleksi
// diterapkan per sel supaya warnanya menyambung tanpa celah.
func renderBatchRow(cells []batchCell, widths []int, selected, header bool) string {
	var b strings.Builder
	for i, cell := range cells {
		if i >= len(widths) {
			break
		}
		w := max(widths[i], 1)
		text := ansi.Truncate(cell.text, max(w-1, 0), "…")
		pad := max(w-1-lipgloss.Width(text), 0)
		line := " "
		if cell.right {
			line += strings.Repeat(" ", pad) + text
		} else {
			line += text + strings.Repeat(" ", pad)
		}

		style := lipgloss.NewStyle().Foreground(cell.fg)
		if cell.bold {
			style = style.Bold(true)
		}
		switch {
		case selected:
			style = style.Background(styles.ColSelectionBg).Foreground(styles.ColTextBright)
		case header:
			style = style.Background(styles.ColSurfaceAlt)
		}
		b.WriteString(style.Render(line))
	}
	b.WriteString(" ")
	return b.String()
}

// batchNameAt mengembalikan nama request pada indeks tertentu.
func (m AppModel) batchNameAt(i int) string {
	if i < len(m.BatchResults) && m.BatchResults[i].RequestName != "" {
		return m.BatchResults[i].RequestName
	}
	if i < len(m.BatchNames) && m.BatchNames[i] != "" {
		return m.BatchNames[i]
	}
	if i < len(m.BatchResults) && m.BatchResults[i].URL != "" {
		return m.BatchResults[i].URL
	}
	return "—"
}

// batchTarget meringkas URL menjadi host + path agar kolom TARGET informatif.
func batchTarget(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return raw
	}
	return parsed.Host + parsed.Path
}

func batchDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// batchFooterView adalah status bar satu baris khusus mode batch.
func (m AppModel) batchFooterView() string {
	keyStyle := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	sep := lipgloss.NewStyle().Foreground(styles.ColBorder).Render(" │ ")

	badge := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Render(" BATCH ")

	shortcuts := [][2]string{
		{"↑/↓", "pilih"},
		{"enter", "buka"},
		{"tab", "respon"},
		{"^f", "find"},
		{"esc", "kembali"},
	}
	parts := make([]string, 0, len(shortcuts))
	for _, s := range shortcuts {
		parts = append(parts, keyStyle.Render(s[0])+descStyle.Render(" "+s[1]))
	}
	middle := " " + strings.Join(parts, sep)

	passed, failed, waiting := 0, 0, 0
	for _, r := range m.BatchResults {
		if r.RequestName == "" && r.URL == "" {
			waiting++
			continue
		}
		if r.Err != "" || r.Code >= 400 {
			failed++
		} else {
			passed++
		}
	}
	right := " " +
		lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true).Render(fmt.Sprintf("✓ %d", passed)) + sep +
		lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).Render(fmt.Sprintf("× %d", failed)) + sep +
		descStyle.Render(fmt.Sprintf("◷ %d", waiting)) + " "

	gap := m.ScreenWidth - lipgloss.Width(badge) - lipgloss.Width(middle) - lipgloss.Width(right)
	for len(parts) > 1 && gap < 1 {
		parts = parts[:len(parts)-1]
		middle = " " + strings.Join(parts, sep)
		gap = m.ScreenWidth - lipgloss.Width(badge) - lipgloss.Width(middle) - lipgloss.Width(right)
	}
	if gap < 0 {
		gap = 0
	}

	bar := badge + middle + strings.Repeat(" ", gap) + right
	return lipgloss.NewStyle().
		Background(styles.ColStatusBg).
		MaxHeight(1).
		MaxWidth(m.ScreenWidth).
		Render(bar)
}
