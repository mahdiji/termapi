package ui

import (
	"fmt"
	"strings"

	"api/internal/ui/sidebar"
	"api/internal/ui/styles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

// viewSearch merender file finder full-screen: input query, daftar request
// file/folder, dan panel informasi file terpilih.
func (m AppModel) viewSearch() tea.View {
	w := max(m.ScreenWidth, 105)
	h := max(m.ScreenHeight, 32)

	suggestions := m.searchSuggestions()
	selected := m.SearchSelectedIndex
	if selected >= len(suggestions) {
		selected = max(len(suggestions)-1, 0)
	}
	if selected < 0 {
		selected = 0
	}

	header := m.searchHeaderView(w, suggestions)
	m.SearchInput.SetWidth(max(w-22, 20))
	searchBar := lipgloss.NewStyle().
		Border(styles.PanelBorder).
		BorderForeground(styles.ColBorderFocus).
		Background(styles.ColSurface).
		Padding(0, 1).
		Width(w).
		Render(m.SearchInput.View())

	bodyH := max(h-1-lipgloss.Height(searchBar)-1, 8)
	listW := max(w*58/100, 55)
	detailW := max(w-listW-1, 30)

	results := searchPanel(
		fmt.Sprintf("RESULTS · %d", len(suggestions)),
		m.searchResultLines(listW, bodyH, suggestions, selected),
		listW, bodyH, true,
	)
	details := searchPanel(
		"FILE INFO & SEARCH TIPS",
		m.searchDetailLines(detailW, suggestions, selected),
		detailW, bodyH, false,
	)
	body := lipgloss.JoinHorizontal(lipgloss.Top, results, " ", details)

	parts := []string{
		header,
		searchBar,
		body,
		m.searchFooterView(w, len(suggestions)),
	}

	p := tea.NewView(zone.Scan(strings.Join(parts, "\n")))
	p.AltScreen = true
	return p
}

func (m AppModel) searchHeaderView(w int, suggestions []sidebar.SearchSuggestion) string {
	files, folders := 0, 0
	for _, s := range suggestions {
		if s.IsFolder {
			folders++
		} else {
			files++
		}
	}

	brand := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Padding(0, 1).
		Render("TermAPI")
	title := "FILE SEARCH"
	if m.SearchHistory {
		title = "HISTORY SEARCH"
	}
	left := brand + " " + lipgloss.NewStyle().
		Foreground(styles.ColTextBright).
		Bold(true).
		Render(title)

	sep := lipgloss.NewStyle().Foreground(styles.ColBorder).Render(" │ ")
	rightParts := []string{
		lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true).
			Render(fmt.Sprintf("%d matches", len(suggestions))),
	}
	if len(suggestions) > 0 {
		fileLabel, folderLabel := "files", "folders"
		if files == 1 {
			fileLabel = "file"
		}
		if folders == 1 {
			folderLabel = "folder"
		}
		rightParts = append(rightParts,
			lipgloss.NewStyle().Foreground(styles.ColTextMuted).
				Render(fmt.Sprintf("%d %s · %d %s", files, fileLabel, folders, folderLabel)))
	}
	rightParts = append(rightParts,
		lipgloss.NewStyle().Foreground(styles.ColTextDim).
			Render("collections/**/*.yaml"))

	return searchJoinRow(left, strings.Join(rightParts, sep), w)
}

func (m AppModel) searchResultLines(panelW, panelH int, suggestions []sidebar.SearchSuggestion, selected int) []string {
	innerW := max(panelW-4, 1)
	bodyRows := max(panelH-4, 1)

	muted := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
	dim := lipgloss.NewStyle().Foreground(styles.ColTextDim)

	prefixW, methodW := 2, 6
	pathW := max(innerW/3, 14)
	nameW := innerW - prefixW - methodW - pathW - 2
	if nameW < 8 {
		pathW = max(innerW-prefixW-methodW-2-8, 6)
		nameW = innerW - prefixW - methodW - pathW - 2
	}

	headerRow := "  " + searchPad("METHOD", methodW) + " " +
		searchPad("REQUEST", nameW) + " " + searchPad("PATH", pathW)
	lines := []string{
		muted.Render(searchFit(headerRow, innerW)),
		dim.Render(strings.Repeat("─", innerW)),
	}

	rows := max(bodyRows-2, 1)
	start, end := searchVisibleWindow(selected, len(suggestions), rows)

	for i := start; i < end; i++ {
		s := suggestions[i]
		method := s.Method
		if s.IsFolder {
			method = "DIR"
		}
		if method == "" {
			method = "GET"
		}

		name := s.Name
		if name == "" {
			name = s.Text
		}
		if name == "" {
			name = s.Path
		}

		nameCell := searchPad(searchTruncate(name, nameW), nameW)
		pathCell := searchTruncate(s.Path, pathW)

		if i == selected {
			plain := "  " + searchPad(method, methodW) + " " + nameCell + " " + pathCell
			lines = append(lines, styles.Selected().Width(innerW).Render(searchFit(plain, innerW)))
			continue
		}

		methodStyle := styles.MethodBadge(method)
		if s.IsFolder {
			methodStyle = lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
		}
		row := "  " + methodStyle.Render(searchPad(method, methodW)) + " " +
			nameCell + " " + muted.Render(pathCell)
		lines = append(lines, searchFit(row, innerW))
	}

	if len(suggestions) == 0 {
		query := strings.TrimSpace(m.SearchInput.Value())
		lines = append(lines, "")
		lines = append(lines, muted.Render("  No files match "+searchTruncate(fmt.Sprintf("%q", query), max(innerW-18, 8))+"."))
		lines = append(lines, dim.Render("  Try a request name, file path,"))
		lines = append(lines, dim.Render("  HTTP method, or URL fragment."))
	}

	return lines
}

func (m AppModel) searchDetailLines(panelW int, suggestions []sidebar.SearchSuggestion, selected int) []string {
	innerW := max(panelW-4, 1)

	section := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
	dim := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	value := lipgloss.NewStyle().Foreground(styles.ColTextBright)
	success := lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true)

	field := func(label, val string) string {
		return "  " + muted.Render(searchPad(label, 12)) +
			value.Render(searchTruncate(val, max(innerW-15, 4)))
	}

	var lines []string
	if len(suggestions) == 0 || selected < 0 || selected >= len(suggestions) {
		lines = append(lines,
			muted.Render("(no file selected)"),
			dim.Render("Type to search collection files."),
		)
	} else {
		s := suggestions[selected]
		if s.IsFolder {
			lines = append(lines,
				section.Render("FOLDER"),
				field("TYPE", "Collection folder"),
				field("NAME", s.Name),
				field("PATH", s.Path),
				field("COLLECTION", s.Collection),
				field("ENTRIES", fmt.Sprintf("%d", s.Children)),
			)
		} else {
			folder := s.Folder
			if folder == "" {
				folder = "— (collection root)"
			}
			url := s.URL
			if url == "" {
				url = "—"
			}
			lines = append(lines,
				section.Render("REQUEST FILE"),
				field("TYPE", "Request file (.yaml)"),
				field("METHOD", s.Method),
				field("NAME", s.Name),
				field("PATH", s.Path),
				field("FOLDER", folder),
				field("COLLECTION", s.Collection),
				field("URL", url),
			)
		}
		lines = append(lines,
			"",
			section.Render("OPEN"),
		)
		if s.IsFolder {
			lines = append(lines,
				muted.Render("  Enter reveals this folder in"),
				muted.Render("  the sidebar."),
			)
		} else {
			lines = append(lines,
				muted.Render("  Enter opens the selected request."),
				success.Render("  ✓ Ready to load"),
			)
		}
	}

	lines = append(lines,
		"",
		section.Render("SEARCH SCOPE"),
		muted.Render("  All collection request files are"),
		muted.Render("  indexed, including files inside"),
		muted.Render("  collapsed folders."),
		"",
		section.Render("MATCHES"),
		muted.Render("  Request name, method, URL,"),
		muted.Render("  folder path, and file path."),
	)

	return lines
}

func (m AppModel) searchFooterView(w int, matches int) string {
	keySt := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	descSt := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	sepSt := lipgloss.NewStyle().Foreground(styles.ColBorder)

	modeBadge := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Render(" SEARCH ")

	type shortcut struct {
		key  string
		desc string
	}
	keys := []shortcut{
		{"↑/↓", "navigate"},
		{"enter", "open"},
		{"esc", "close"},
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, keySt.Render(k.key)+descSt.Render(" "+k.desc))
	}
	sep := sepSt.Render(" │ ")
	middle := " " + strings.Join(parts, sep)

	right := descSt.Render(fmt.Sprintf("%d matches", matches)) + sepSt.Render(" │ ") +
		descSt.Render("ctrl+f files") + " "

	gap := w - lipgloss.Width(modeBadge) - lipgloss.Width(middle) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	bar := modeBadge + middle + strings.Repeat(" ", gap) + right
	return lipgloss.NewStyle().
		Background(styles.ColStatusBg).
		MaxHeight(1).
		MaxWidth(w).
		Render(ansi.Truncate(bar, w, ""))
}

// ── panel helper ────────────────────────────────────────────────────────────

func searchPanel(title string, lines []string, width, height int, focused bool) string {
	width = max(width, 8)
	height = max(height, 4)

	innerW := max(width-4, 1)
	innerH := max(height-2, 2)
	bodyRows := max(innerH-2, 0)

	titleSt := lipgloss.NewStyle().Foreground(styles.ColTextMuted).Bold(true)
	if focused {
		titleSt = lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	}
	sep := lipgloss.NewStyle().Foreground(styles.ColBorder).Render(strings.Repeat("─", innerW))

	out := make([]string, 0, bodyRows+2)
	out = append(out, titleSt.Render(searchFit(title, innerW)), sep)
	for i := 0; i < bodyRows; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		out = append(out, searchFit(line, innerW))
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

func searchTruncate(s string, n int) string {
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

func searchPad(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

func searchFit(s string, w int) string {
	return searchPad(searchTruncate(s, w), w)
}

func searchVisibleWindow(cursor, total, rows int) (int, int) {
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

func searchJoinRow(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	maxRight := max(width-lipgloss.Width(left)-1, 1)
	right = searchTruncate(right, maxRight)
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}
