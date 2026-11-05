package diffscreen

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"api/internal/ui/styles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/x/ansi"
)

type Mode int

const (
	ModeSideBySide Mode = iota
	ModeUnified
)

type Model struct {
	OriginalText string
	ModifiedText string
	Width        int
	Height       int
	Mode         Mode
	ScrollOffset int
	ActiveHunk   int
	ExpandAll    bool
	Loading      bool
	ShowHelp     bool
	PickerOpen   bool
	PickerCursor int
	PickerScroll int
	LabelA       string
	LabelB       string
	SourceNote   string
	Options      []BaselineOption
	SelectedA    int
	DiffResult   *DiffData
	Err          string
}

// BaselineOption adalah satu history run yang bisa dipilih sebagai A.
type BaselineOption struct {
	Label string
	Body  string
}

type DiffData struct {
	Hunks       []Hunk
	StatsAdd    int
	StatsDel    int
	StatsChange int
}

type Hunk struct {
	OldStart  int
	OldLines  int
	NewStart  int
	NewLines  int
	Lines     []DiffLine
	Collapsed bool
}

type DiffLine struct {
	Type      LineType
	OldLineNo int
	NewLineNo int
	Content   string
	Intra     []IntraSegment
}

type LineType int

const (
	LineEqual LineType = iota
	LineAdd
	LineDel
)

type IntraSegment struct {
	Text    string
	Added   bool
	Deleted bool
}

func NewModel(orig, mod string) *Model {
	m := &Model{
		OriginalText: orig,
		ModifiedText: mod,
		Mode:         ModeSideBySide,
		LabelA:       "A · baseline (run sebelumnya)",
		LabelB:       "B · terbaru",
		SelectedA:    -1,
	}
	m.ComputeDiff()
	return m
}

func (m *Model) SetSize(w, h int) {
	m.Width = w
	m.Height = h
}

func (m *Model) ComputeDiff() {
	origNorm := normalizeJSON(m.OriginalText)
	modNorm := normalizeJSON(m.ModifiedText)

	edits := udiff.Unified("original", "modified", origNorm, modNorm)
	hunks := parseUnifiedToHunks(edits)

	addCount, delCount, changeCount := 0, 0, 0
	for _, h := range hunks {
		hasAdd, hasDel := false, false
		for _, l := range h.Lines {
			if l.Type == LineAdd {
				addCount++
				hasAdd = true
			} else if l.Type == LineDel {
				delCount++
				hasDel = true
			}
		}
		if hasAdd && hasDel {
			changeCount++
		}
	}

	m.DiffResult = &DiffData{
		Hunks:       hunks,
		StatsAdd:    addCount,
		StatsDel:    delCount,
		StatsChange: changeCount,
	}
	// Default collapse large equal blocks
	for i := range m.DiffResult.Hunks {
		if countEqualLines(m.DiffResult.Hunks[i].Lines) > 10 {
			m.DiffResult.Hunks[i].Collapsed = true
		}
	}
}

func normalizeJSON(text string) string {
	var obj any
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return text // fallback to raw
	}
	// Re-marshal indented and sorted keys if possible via json
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return text
	}
	return string(b)
}

func countEqualLines(lines []DiffLine) int {
	eq := 0
	for _, l := range lines {
		if l.Type == LineEqual {
			eq++
		}
	}
	return eq
}

func parseUnifiedToHunks(unified string) []Hunk {
	lines := strings.Split(unified, "\n")
	var hunks []Hunk
	var curHunk *Hunk

	for _, l := range lines {
		if strings.HasPrefix(l, "@@") {
			if curHunk != nil {
				hunks = append(hunks, *curHunk)
			}
			curHunk = &Hunk{Collapsed: false}
			parseHunkRange(l, curHunk)
			continue
		}
		if curHunk == nil {
			continue
		}
		if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			curHunk.Lines = append(curHunk.Lines, DiffLine{
				Type:    LineAdd,
				Content: l[1:],
			})
		} else if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			curHunk.Lines = append(curHunk.Lines, DiffLine{
				Type:    LineDel,
				Content: l[1:],
			})
		} else if strings.HasPrefix(l, " ") || l == "" {
			content := l
			if strings.HasPrefix(l, " ") {
				content = l[1:]
			}
			curHunk.Lines = append(curHunk.Lines, DiffLine{
				Type:    LineEqual,
				Content: content,
			})
		}
	}
	if curHunk != nil {
		hunks = append(hunks, *curHunk)
	}

	// Assign line numbers
	oldNo, newNo := 1, 1
	for hIdx := range hunks {
		for lIdx := range hunks[hIdx].Lines {
			l := &hunks[hIdx].Lines[lIdx]
			switch l.Type {
			case LineEqual:
				l.OldLineNo = oldNo
				l.NewLineNo = newNo
				oldNo++
				newNo++
			case LineDel:
				l.OldLineNo = oldNo
				oldNo++
			case LineAdd:
				l.NewLineNo = newNo
				newNo++
			}
		}
	}
	return hunks
}

// parseHunkRange membaca header unified diff, contoh: "@@ -12,7 +12,9 @@".
func parseHunkRange(header string, h *Hunk) {
	fields := strings.Fields(header)
	if len(fields) < 3 {
		return
	}
	h.OldStart, h.OldLines = parseHunkPart(fields[1])
	h.NewStart, h.NewLines = parseHunkPart(fields[2])
}

func parseHunkPart(part string) (start, count int) {
	part = strings.TrimPrefix(strings.TrimPrefix(part, "-"), "+")
	count = 1
	if idx := strings.Index(part, ","); idx >= 0 {
		if n, err := strconv.Atoi(part[idx+1:]); err == nil {
			count = n
		}
		part = part[:idx]
	}
	if n, err := strconv.Atoi(part); err == nil {
		start = n
	}
	return start, count
}

func FitLine(s string, w int) string {
	if w <= 0 {
		return ""
	}
	currentWidth := ansi.StringWidth(s)
	if currentWidth > w {
		if w >= 1 {
			truncated := ansi.Truncate(s, w-1, "…")
			tw := ansi.StringWidth(truncated)
			if tw < w {
				truncated += strings.Repeat(" ", w-tw)
			}
			return truncated + "\x1b[0m"
		}
		return strings.Repeat(" ", w)
	} else if currentWidth < w {
		return s + strings.Repeat(" ", w-currentWidth) + "\x1b[0m"
	}
	return s + "\x1b[0m"
}

func BoxWithTitle(title, body string, w, h int) string {
	if w < 10 || h < 5 {
		return "Terminal terlalu kecil"
	}
	innerW := w - 4
	if innerW < 1 {
		innerW = 1
	}

	titleRunes := []rune(title)
	titleStr := string(titleRunes)
	titleWidth := ansi.StringWidth(titleStr)
	prefix := "╭─ "
	suffix := "╮"
	usedWidth := ansi.StringWidth(prefix) + titleWidth + 1 + ansi.StringWidth(suffix)
	rem := w - usedWidth
	if rem < 0 {
		rem = 0
	}
	topLine := prefix + titleStr + " " + strings.Repeat("─", rem) + suffix

	bodyLines := strings.Split(body, "\n")
	var formattedLines []string
	for _, line := range bodyLines {
		fitted := FitLine(line, innerW)
		formattedLines = append(formattedLines, "│ "+fitted+" │")
	}

	bottomLine := "╰" + strings.Repeat("─", w-2) + "╯"

	var result []string
	result = append(result, topLine)
	for _, fl := range formattedLines {
		result = append(result, fl)
	}

	currentH := len(result) + 1
	for currentH < h {
		result = append(result, "│"+strings.Repeat(" ", w-2)+"│")
		currentH++
	}
	result = append(result, bottomLine)

	return strings.Join(result, "\n")
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		if m.PickerOpen {
			m.handlePickerKey(key)
			return m, nil
		}

		// Saat panduan terbuka, hanya tombol tutup yang aktif.
		if m.ShowHelp {
			switch key {
			case "?", "f1", "esc", "q":
				m.ShowHelp = false
			}
			return m, nil
		}

		switch key {
		case "?", "f1":
			m.ShowHelp = true
		case "b":
			m.openPicker()
		case "m":
			if m.Mode == ModeSideBySide {
				m.Mode = ModeUnified
			} else {
				m.Mode = ModeSideBySide
			}
		case "e":
			m.ExpandAll = !m.ExpandAll
			if m.DiffResult != nil {
				for i := range m.DiffResult.Hunks {
					m.DiffResult.Hunks[i].Collapsed = !m.ExpandAll
				}
			}
		case "j", "down":
			m.ScrollOffset++
		case "k", "up":
			if m.ScrollOffset > 0 {
				m.ScrollOffset--
			}
		case "pgdown", "ctrl+d":
			m.ScrollOffset += 10
		case "pgup", "ctrl+u":
			m.ScrollOffset = max(m.ScrollOffset-10, 0)
		case "g", "home":
			m.ScrollOffset = 0
		case "G", "end":
			m.ScrollOffset = 1 << 30
		case "n", "tab":
			m.nextHunk(1)
		case "p", "shift+tab":
			m.nextHunk(-1)
		}
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.MouseWheelMsg:
		if m.PickerOpen {
			if msg.Button == tea.MouseWheelUp {
				m.movePicker(-3)
			} else if msg.Button == tea.MouseWheelDown {
				m.movePicker(3)
			}
			return m, nil
		}
		switch msg.Button {
		case tea.MouseWheelUp:
			m.ScrollOffset = max(m.ScrollOffset-3, 0)
		case tea.MouseWheelDown:
			m.ScrollOffset += 3
		}
	}
	return m, nil
}

func (m *Model) openPicker() {
	m.PickerOpen = true
	m.ShowHelp = false
	if m.SelectedA >= 0 && m.SelectedA < len(m.Options) {
		m.PickerCursor = m.SelectedA
	} else {
		m.PickerCursor = 0
	}
	m.clampPickerCursor()
}

func (m *Model) handlePickerKey(key string) {
	switch key {
	case "esc", "b", "q":
		m.PickerOpen = false
	case "j", "down":
		m.movePicker(1)
	case "k", "up":
		m.movePicker(-1)
	case "g", "home":
		m.PickerCursor = 0
	case "G", "end":
		m.PickerCursor = len(m.Options) - 1
		m.clampPickerCursor()
	case "pgdown", "ctrl+d":
		m.movePicker(10)
	case "pgup", "ctrl+u":
		m.movePicker(-10)
	case "enter":
		m.applyBaseline()
	}
}

func (m *Model) movePicker(delta int) {
	m.PickerCursor += delta
	m.clampPickerCursor()
}

func (m *Model) clampPickerCursor() {
	if len(m.Options) == 0 {
		m.PickerCursor = 0
		return
	}
	if m.PickerCursor < 0 {
		m.PickerCursor = 0
	}
	if m.PickerCursor >= len(m.Options) {
		m.PickerCursor = len(m.Options) - 1
	}
}

func (m *Model) applyBaseline() {
	if m.PickerCursor < 0 || m.PickerCursor >= len(m.Options) {
		return
	}
	opt := m.Options[m.PickerCursor]
	m.OriginalText = opt.Body
	m.LabelA = opt.Label
	m.SelectedA = m.PickerCursor
	m.ScrollOffset = 0
	m.ActiveHunk = 0
	m.ExpandAll = false
	m.ComputeDiff()
	m.PickerOpen = false
}

func (m *Model) nextHunk(delta int) {
	if m.DiffResult == nil || len(m.DiffResult.Hunks) == 0 {
		return
	}
	n := len(m.DiffResult.Hunks)
	m.ActiveHunk = (m.ActiveHunk + delta + n) % n
	_, starts := m.renderDiffLines()
	if m.ActiveHunk < len(starts) {
		m.ScrollOffset = starts[m.ActiveHunk]
	}
}

func (m *Model) View() string {
	w, h := m.Width, m.Height
	if w < 60 || h < 15 {
		msg := lipgloss.NewStyle().
			Foreground(styles.ColError).
			Bold(true).
			Render("Terminal terlalu kecil (minimal 60x15)")
		return lipgloss.Place(max(w, 1), max(h, 1), lipgloss.Center, lipgloss.Center, msg)
	}

	header := m.headerView(w)
	footer := m.footerView(w)

	border := lipgloss.NewStyle().Foreground(styles.ColBorder)
	titleText := " DIFF · A vs B "
	title := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true).Render(titleText)
	innerW := w - 2
	top := border.Render("╭─") + title +
		border.Render(strings.Repeat("─", max(innerW-lipgloss.Width(title)-1, 0))+"╮")
	bottom := border.Render("╰" + strings.Repeat("─", innerW) + "╯")

	bodyH := max(h-4, 3)
	content := m.bodyLines(innerW, bodyH)
	for len(content) < bodyH {
		content = append(content, "")
	}
	if len(content) > bodyH {
		content = content[:bodyH]
	}

	lines := make([]string, 0, h)
	lines = append(lines, header)
	lines = append(lines, top)
	for _, line := range content {
		lines = append(lines, border.Render("│")+FitLine(line, innerW)+border.Render("│"))
	}
	lines = append(lines, bottom, footer)
	return strings.Join(lines, "\n")
}

func (m *Model) bodyLines(innerW, bodyH int) []string {
	if m.PickerOpen {
		return m.pickerLines(innerW, bodyH)
	}
	if m.ShowHelp {
		return m.helpLines(innerW)
	}

	var content []string
	content = append(content, m.summaryLine())
	content = append(content, styles.Muted().Render(" "+
		labelOr(m.LabelA, "A · baseline (run sebelumnya)")+"  →  "+
		labelOr(m.LabelB, "B · terbaru")))
	if m.SourceNote != "" {
		content = append(content, styles.Dim().Render(" "+m.SourceNote))
	}
	content = append(content, styles.Dim().Render(" "+strings.Repeat("─", max(innerW-2, 0))))

	prefix := len(content)
	viewport := max(bodyH-prefix, 1)

	if m.DiffResult == nil || len(m.DiffResult.Hunks) == 0 {
		content = append(content, styles.Muted().Render(" Tidak ada perubahan terdeteksi."))
		return content
	}

	diffLines, _ := m.renderDiffLines()
	if len(diffLines) == 0 {
		content = append(content, styles.Muted().Render(" Tidak ada perubahan terdeteksi."))
		return content
	}

	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}
	maxOffset := max(len(diffLines)-viewport, 0)
	if m.ScrollOffset > maxOffset {
		m.ScrollOffset = maxOffset
	}
	end := min(m.ScrollOffset+viewport, len(diffLines))
	content = append(content, diffLines[m.ScrollOffset:end]...)
	return content
}

func (m *Model) pickerLines(innerW, bodyH int) []string {
	titleStyle := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true)
	dimStyle := styles.Dim()
	descStyle := styles.Muted()

	content := []string{
		titleStyle.Render(" PILIH BASELINE · A"),
		descStyle.Render(" Pilih run lama sebagai pembanding. B tetap response terbaru."),
		dimStyle.Render(" A sekarang: " + labelOr(m.LabelA, "—")),
		dimStyle.Render(" B:          " + labelOr(m.LabelB, "—")),
		dimStyle.Render(" " + strings.Repeat("─", max(innerW-2, 0))),
	}

	footer := dimStyle.Render(" ↑/↓ pilih · Enter terapkan · b/Esc batal")
	rows := max(bodyH-len(content)-1, 1)

	if len(m.Options) == 0 {
		content = append(content, descStyle.Render(" Belum ada history untuk request ini."))
		content = append(content, footer)
		return content
	}

	m.clampPickerCursor()
	if m.PickerCursor < m.PickerScroll {
		m.PickerScroll = m.PickerCursor
	}
	if m.PickerCursor >= m.PickerScroll+rows {
		m.PickerScroll = m.PickerCursor - rows + 1
	}
	maxScroll := max(len(m.Options)-rows, 0)
	if m.PickerScroll > maxScroll {
		m.PickerScroll = maxScroll
	}
	if m.PickerScroll < 0 {
		m.PickerScroll = 0
	}

	end := min(m.PickerScroll+rows, len(m.Options))
	for i := m.PickerScroll; i < end; i++ {
		marker := "  "
		if i == m.PickerCursor {
			marker = "▸ "
		}
		chosen := " "
		if i == m.SelectedA {
			chosen = "✓"
		}
		row := " " + marker + chosen + " " + m.Options[i].Label
		if i == m.PickerCursor {
			content = append(content, styles.Selected().Render(row))
		} else if i == m.SelectedA {
			content = append(content, lipgloss.NewStyle().Foreground(styles.ColSuccess).Render(row))
		} else {
			content = append(content, styles.Text().Render(row))
		}
	}
	content = append(content, footer)
	return content
}

func (m *Model) summaryLine() string {
	stats := " Tidak ada data diff."
	if m.DiffResult != nil {
		modeName := map[Mode]string{ModeSideBySide: "Side-by-Side", ModeUnified: "Unified"}[m.Mode]
		total := len(m.DiffResult.Hunks)
		hunkInfo := "0/0"
		if total > 0 {
			hunkInfo = fmt.Sprintf("%d/%d", m.ActiveHunk+1, total)
		}
		stats = fmt.Sprintf(" +%d added   -%d removed   ~%d hunks   mode: %s   hunk: %s",
			m.DiffResult.StatsAdd, m.DiffResult.StatsDel, m.DiffResult.StatsChange, modeName, hunkInfo)
	}
	return lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true).Render(stats)
}

func (m *Model) headerView(w int) string {
	brand := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Padding(0, 1).
		Render("TermAPI")
	title := lipgloss.NewStyle().Foreground(styles.ColTextBright).Bold(true).
		Render("DIFF VIEWER")
	left := brand + " " + title
	right := styles.Muted().Render("baseline vs response terbaru")
	gap := w - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 1 {
		return FitLine(left+" "+right, w)
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m *Model) footerView(w int) string {
	keyStyle := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	sep := styles.Dim().Render(" │ ")

	type shortcut struct{ key, desc string }
	keys := []shortcut{
		{"j/k", "scroll"},
		{"n/p", "hunk"},
		{"b", "pilih baseline"},
		{"e", "expand"},
		{"m", "mode"},
		{"g/G", "atas/bawah"},
		{"?/F1", "panduan"},
		{"Esc/q", "tutup"},
	}
	if m.PickerOpen {
		keys = []shortcut{
			{"↑/↓", "pilih"},
			{"Enter", "terapkan"},
			{"b/Esc", "batal"},
		}
	}
	if m.ShowHelp {
		keys = []shortcut{{"?/Esc", "tutup panduan"}}
	}

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k.key)+" "+descStyle.Render(k.desc))
	}
	return FitLine(" "+strings.Join(parts, sep), w)
}

func (m *Model) helpLines(innerW int) []string {
	section := lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	descStyle := styles.Text()
	dimStyle := styles.Dim()

	content := []string{
		lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true).Render(" DIFF VIEWER · PANDUAN PEMAKAIAN"),
		"",
		dimStyle.Render(" A (kiri, penanda -) = baseline, yaitu response dari run sebelumnya."),
		dimStyle.Render(" B (kanan, penanda +) = response terbaru dari request yang baru dijalankan."),
		dimStyle.Render(" Otomatis memakai 2 run terakhir dengan method + URL yang sama."),
		dimStyle.Render(" Tekan 'b' untuk memilih baseline lain dari history request ini."),
		dimStyle.Render(" Baris ~ menandai hunk, yaitu satu blok perubahan. Tanpa hunk berarti response identik."),
		"",
		section.Render(" SHORTCUT"),
	}

	shortcuts := [][2]string{
		{"j / ↓", "Scroll ke bawah satu baris"},
		{"k / ↑", "Scroll ke atas satu baris"},
		{"PgDn / Ctrl+D", "Scroll satu halaman ke bawah"},
		{"PgUp / Ctrl+U", "Scroll satu halaman ke atas"},
		{"g / Home", "Lompat ke paling atas"},
		{"G / End", "Lompat ke paling bawah"},
		{"n / Tab", "Ke hunk berikutnya (otomatis scroll)"},
		{"p / Shift+Tab", "Ke hunk sebelumnya (otomatis scroll)"},
		{"b", "Pilih baseline (A) lain dari history"},
		{"e", "Buka / tutup semua blok yang tidak berubah"},
		{"m", "Ganti tampilan Side-by-Side ↔ Unified"},
		{"Mouse wheel", "Scroll dengan roda mouse"},
		{"? / F1", "Tampilkan / tutup panduan ini"},
		{"Esc / q", "Tutup Diff Viewer"},
	}
	for _, s := range shortcuts {
		content = append(content, " "+keyStyle.Render(padRight(s[0], 16))+" "+descStyle.Render(s[1]))
	}

	content = append(content, "", dimStyle.Render(" Tips: aktifkan mode Unified dengan 'm' jika terminal sempit."))
	return content
}

func (m *Model) renderDiffLines() ([]string, []int) {
	var lines []string
	starts := []int{}
	if m.DiffResult == nil {
		return lines, starts
	}

	addStyle := lipgloss.NewStyle().Foreground(styles.ColSuccess)
	delStyle := lipgloss.NewStyle().Foreground(styles.ColError)
	dimStyle := styles.Dim()
	equalStyle := styles.Text()
	hunkStyle := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true)
	hunkActiveStyle := lipgloss.NewStyle().
		Foreground(styles.ColSelectionFg).
		Background(styles.ColSelectionBg).
		Bold(true)

	innerW := max(m.Width-2, 20)

	for hIdx, hunk := range m.DiffResult.Hunks {
		starts = append(starts, len(lines))
		hunkHeader := fmt.Sprintf(" @@ Hunk %d · old %d, new %d @@", hIdx+1, hunk.OldStart, hunk.NewStart)
		if hIdx == m.ActiveHunk {
			lines = append(lines, hunkActiveStyle.Render(hunkHeader))
		} else {
			lines = append(lines, hunkStyle.Render(hunkHeader))
		}

		if hunk.Collapsed {
			unchanged := countEqualLines(hunk.Lines)
			lines = append(lines, dimStyle.Render(fmt.Sprintf("   ┄┄ %d baris tidak berubah · tekan 'e' untuk membuka ┄┄", unchanged)))
			continue
		}

		if m.Mode == ModeUnified {
			for _, l := range hunk.Lines {
				switch l.Type {
				case LineEqual:
					lines = append(lines, " "+dimStyle.Render(fmt.Sprintf("%4d", l.OldLineNo))+
						dimStyle.Render(" | ")+dimStyle.Render(fmt.Sprintf("%4d", l.NewLineNo))+
						dimStyle.Render(" │   ")+equalStyle.Render(l.Content))
				case LineAdd:
					lines = append(lines, " "+dimStyle.Render("     |")+dimStyle.Render(fmt.Sprintf(" %4d", l.NewLineNo))+
						dimStyle.Render(" │ ")+addStyle.Render("+ "+l.Content))
				case LineDel:
					lines = append(lines, " "+dimStyle.Render(fmt.Sprintf("%4d", l.OldLineNo))+dimStyle.Render(" |      ")+
						dimStyle.Render(" │ ")+delStyle.Render("- "+l.Content))
				}
			}
		} else {
			// Side-by-Side Mode
			colW := (innerW - 3) / 2
			if colW < 14 {
				colW = 14
			}
			contentW := max(colW-8, 1)

			i := 0
			hLines := hunk.Lines
			for i < len(hLines) {
				l := hLines[i]
				switch l.Type {
				case LineEqual:
					left := " " + dimStyle.Render(fmt.Sprintf("%4d", l.OldLineNo)) + dimStyle.Render(" │ ") +
						equalStyle.Render(FitLine(l.Content, contentW))
					right := " " + dimStyle.Render(fmt.Sprintf("%4d", l.NewLineNo)) + dimStyle.Render(" │ ") +
						equalStyle.Render(FitLine(l.Content, contentW))
					lines = append(lines, left+dimStyle.Render(" │ ")+right)
					i++
				case LineDel:
					left := " " + dimStyle.Render(fmt.Sprintf("%4d", l.OldLineNo)) + dimStyle.Render(" │ ") +
						delStyle.Render(FitLine(l.Content, contentW))
					if i+1 < len(hLines) && hLines[i+1].Type == LineAdd {
						r := hLines[i+1]
						right := " " + dimStyle.Render(fmt.Sprintf("%4d", r.NewLineNo)) + dimStyle.Render(" │ ") +
							addStyle.Render(FitLine(r.Content, contentW))
						lines = append(lines, left+dimStyle.Render(" │ ")+right)
						i += 2
					} else {
						lines = append(lines, left+dimStyle.Render(" │ ")+strings.Repeat(" ", colW))
						i++
					}
				case LineAdd:
					right := " " + dimStyle.Render(fmt.Sprintf("%4d", l.NewLineNo)) + dimStyle.Render(" │ ") +
						addStyle.Render(FitLine(l.Content, contentW))
					lines = append(lines, strings.Repeat(" ", colW)+dimStyle.Render(" │ ")+right)
					i++
				}
			}
		}
	}
	return lines, starts
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func labelOr(label, fallback string) string {
	if label == "" {
		return fallback
	}
	return label
}
