package sidebar

import (
	"strings"

	"api/internal/ui/styles"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

func maxScrollX(items []flatItem, innerWidth int) int {
	if innerWidth < 10 {
		innerWidth = 10
	}
	maxWidth := 0
	for _, item := range items {
		if width := lipgloss.Width(item.Text) + 2; width > maxWidth {
			maxWidth = width
		}
	}
	for _, sc := range sidebarShortcuts {
		w := 2 + lipgloss.Width(sc.key) + 2 + lipgloss.Width(sc.action)
		if w > maxWidth {
			maxWidth = w
		}
	}
	if maxWidth <= innerWidth {
		return 0
	}
	return maxWidth - innerWidth
}

func visibleHeight(height int) int {
	if innerHeight := height - 2; innerHeight > 0 {
		return innerHeight
	}
	return 1
}

func listHeight(height int) int {
	// Seluruh baris dalam panel dipakai untuk daftar.
	return visibleHeight(height)
}

func (m *Model) View() string {
	borderColor := styles.ColBorder
	focused := m.IsFocused
	if focused {
		borderColor = styles.ColBorderFocus
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	titleText := " EXPLORER "
	if m.Filter != "" {
		titleText = " EXPLORER · /" + m.Filter + " "
	}
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.ColTextBright).
		Bold(true)
	if m.IsFocused {
		titleStyle = titleStyle.Foreground(styles.ColAccent)
	}
	styledTitle := titleStyle.Render(titleText)
	titleWidth := lipgloss.Width(styledTitle)

	if m.Widht < 12 {
		m.Widht = 12
	}
	innerW := m.Widht - 2
	innerH := visibleHeight(m.Height)

	availTitle := innerW - 1
	if titleWidth > availTitle {
		styledTitle = titleStyle.Render(ansi.Truncate(titleText, max(1, availTitle), ""))
		titleWidth = lipgloss.Width(styledTitle)
	}
	_ = titleWidth
	topBorder := borderStyle.Render("╭─") + styledTitle + borderStyle.Render(strings.Repeat("─", max(0, innerW-titleWidth-1))+"╮")

	if m.CreateMode != createNone {
		return m.renderCreatePrompt(topBorder, borderStyle, innerW, innerH)
	}

	items := m.getFlatItems()
	listH := listHeight(m.Height)

	maxX := maxScrollX(items, innerW)
	if m.ScrollX > maxX {
		m.ScrollX = maxX
	}
	if m.ScrollX < 0 {
		m.ScrollX = 0
	}

	if m.CursorIdx >= len(items) {
		m.CursorIdx = len(items) - 1
	}
	if m.CursorIdx < 0 {
		m.CursorIdx = 0
	}
	if len(items) > 0 {
		selectedZoneID := items[m.CursorIdx].ZoneID
		if m.SelectedZoneID != selectedZoneID {
			m.SelectedZoneID = selectedZoneID
			items = m.getFlatItems()
		}
	}

	if m.CursorIdx < m.ScrollY {
		m.ScrollY = m.CursorIdx
	} else if m.CursorIdx >= m.ScrollY+listH {
		m.ScrollY = m.CursorIdx - listH + 1
	}
	maxScrollY := len(items) - listH
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	if m.ScrollY > maxScrollY {
		m.ScrollY = maxScrollY
	}
	if m.ScrollY < 0 {
		m.ScrollY = 0
	}

	endIdx := m.ScrollY + listH
	if endIdx > len(items) {
		endIdx = len(items)
	}
	visibleItems := items[m.ScrollY:endIdx]

	var contentLines []string
	for _, item := range visibleItems {
		isSelected := item.ZoneID == m.SelectedZoneID
		clippedText := clipAndPadText(item.Text, m.ScrollX, innerW)
		styled := item.Style.Render(clippedText)
		if item.AccentText != "" {
			if accentAt := strings.Index(clippedText, item.AccentText); accentAt >= 0 {
				before := item.Style.Render(clippedText[:accentAt])
				highlight := item.AccentStyle.Render(item.AccentText)
				after := item.Style.Render(clippedText[accentAt+len(item.AccentText):])
				styled = before + highlight + after
			}
		}

		// Baris terpilih: latar belakang penuh, bukan hanya penanda teks.
		if isSelected {
			// Render ulang dengan bg seleksi; pertahankan lebar persis innerW.
			sel := styles.Selected().Width(innerW).MaxWidth(innerW)
			// Render teks polos dengan bg agar highlight method tetap terlihat,
			// kita bungkus seluruh baris yang sudah di-style.
			styled = sel.Render(styled)
		}
		marked := zone.Mark(item.ZoneID, styled)
		contentLines = append(contentLines, borderStyle.Render("│")+marked+borderStyle.Render("│"))
	}

	for len(contentLines) < listH {
		emptyLine := borderStyle.Render("│") + strings.Repeat(" ", innerW) + borderStyle.Render("│")
		contentLines = append(contentLines, emptyLine)
	}

	for len(contentLines) < innerH {
		emptyLine := borderStyle.Render("│") + strings.Repeat(" ", innerW) + borderStyle.Render("│")
		contentLines = append(contentLines, emptyLine)
	}
	if len(contentLines) > innerH {
		contentLines = contentLines[:innerH]
	}

	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", innerW) + "╯")

	allLines := append([]string{topBorder}, contentLines...)
	allLines = append(allLines, bottomBorder)

	out := strings.Join(allLines, "\n")
	if focused {
		out = lipgloss.NewStyle().Background(styles.ColPanelFocusBg).MaxWidth(m.Widht).Render(out)
	}
	return out
}

func (m *Model) shortcutLine(key, action string, scrollX, width int, border lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	keyStyle := styles.Title(true).Bold(true)
	labelStyle := styles.Bright()

	fullText := "  " + key + "  " + action
	runes := []rune(fullText)
	keyRunes := []rune(key)

	keyStart := 2
	keyEnd := keyStart + len(keyRunes)
	midEnd := keyEnd + 2
	actionEnd := len(runes)

	endX := scrollX + width

	var sb strings.Builder

	// 1. Indent: [0, keyStart)
	s, e := max(scrollX, 0), min(endX, keyStart)
	if s < e {
		sb.WriteString(string(runes[s:e]))
	}

	// 2. Key: [keyStart, keyEnd)
	s, e = max(scrollX, keyStart), min(endX, keyEnd)
	if s < e {
		sb.WriteString(keyStyle.Render(string(runes[s:e])))
	}

	// 3. Mid space: [keyEnd, midEnd)
	s, e = max(scrollX, keyEnd), min(endX, midEnd)
	if s < e {
		sb.WriteString(string(runes[s:e]))
	}

	// 4. Action: [midEnd, actionEnd)
	s, e = max(scrollX, midEnd), min(endX, actionEnd)
	if s < e {
		sb.WriteString(labelStyle.Render(string(runes[s:e])))
	}

	content := sb.String()
	w := lipgloss.Width(content)
	if w > width {
		content = ansi.Truncate(content, width, "")
		w = lipgloss.Width(content)
	}
	if w < width {
		content += strings.Repeat(" ", width-w)
	}

	return border.Render("│") + content + border.Render("│")
}

func clipAndPadText(s string, startX, maxW int) string {
	runes := []rune(s)
	col := 0
	startIdx := 0
	for i, r := range runes {
		w := lipgloss.Width(string(r))
		if w <= 0 {
			w = 1
		}
		if col >= startX {
			startIdx = i
			break
		}
		col += w
		startIdx = i + 1
	}

	var sb strings.Builder
	currentWidth := 0

	for i := startIdx; i < len(runes); i++ {
		r := runes[i]
		w := lipgloss.Width(string(r))
		if w <= 0 {
			w = 1
		}

		if currentWidth+w > maxW {
			break
		}

		sb.WriteRune(r)
		currentWidth += w
	}

	sub := sb.String()
	if currentWidth < maxW {
		sub = sub + strings.Repeat(" ", maxW-currentWidth)
	}

	return sub
}
