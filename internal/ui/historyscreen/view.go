package historyscreen

import (
	"fmt"
	"strings"

	"api/internal/historydb"
	requestinput "api/internal/ui/requestInput"
	"api/internal/ui/styles"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

func (m *Model) View() string {
	width := max(m.Width, 60)
	height := max(m.Height, 18)
	border := lipgloss.NewStyle().Foreground(styles.ColBorder)
	titleText := " HISTORY · RUNS & RESPONSES "
	title := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true).Render(titleText)
	innerWidth := width - 2
	top := border.Render("╭─") + title + border.Render(strings.Repeat("─", max(innerWidth-lipgloss.Width(title)-1, 0))+"╮")
	lines := []string{top}

	subtitle := fmt.Sprintf("  Local SQLite history · %d runs · %d collections", len(m.allRuns), countCollections(m.allRuns))
	lines = append(lines, framed(styles.Muted().Render(subtitle), innerWidth, border))
	if m.Notice != "" {
		lines = append(lines, framed("  "+styles.Bright().Bold(true).Render(m.Notice), innerWidth, border))
	}
	if m.Search.Focused() || m.Query != "" {
		lines = append(lines, framed("  "+m.Search.View(), innerWidth, border))
	}
	if m.Err != "" {
		lines = append(lines, framed("  "+styles.StatusBadge(500).Render("History database: "+m.Err), innerWidth, border))
	}
	if run, ok := m.SelectedRun(); ok {
		lines = append(lines, framed("  "+runDetail(run), innerWidth, border))
	}

	cols, colWidths, showTarget, showVars := historyColumns(innerWidth)
	headerCells := make([]tableCell, 0, len(cols))
	for _, col := range cols {
		headerCells = append(headerCells, styledCell(col, styles.Dim()))
	}
	lines = append(lines, framed(renderCells(headerCells, colWidths, false), innerWidth, border))

	headerHeight := len(lines)
	tableHeight := max((height-headerHeight-8)/2, 4)
	start := 0
	if m.Cursor >= tableHeight {
		start = m.Cursor - tableHeight + 1
	}
	end := min(start+tableHeight, len(m.Runs))
	if len(m.Runs) == 0 {
		lines = append(lines, framed("  "+styles.Dim().Render("No runs recorded yet. Send a request to populate history."), innerWidth, border))
	}
	for i := start; i < end; i++ {
		run := m.Runs[i]
		cells := []tableCell{styledCell(run.Method, styles.MethodBadge(run.Method))}
		cells = append(cells, tableCell{text: display(run.RequestName)})
		if showTarget {
			cells = append(cells, styledCell(display(run.TargetDomain), styles.Muted()))
		}
		status := "—"
		statusStyle := styles.Dim()
		if run.StatusCode > 0 {
			status = fmt.Sprint(run.StatusCode)
			statusStyle = styles.StatusBadge(run.StatusCode)
		}
		cells = append(cells, styledCell(status, statusStyle))
		cells = append(cells, styledCell(formatResponseTime(run.ResponseTime), styles.Muted()))
		if showVars {
			cells = append(cells, styledCell(fmt.Sprint(len(run.CapturedVariables)), styles.Muted()))
		}
		executed := run.ExecutedAt.Local().Format("2006-01-02 15:04:05")
		if run.ExecutedAt.IsZero() {
			executed = "—"
		}
		cells = append(cells, styledCell(executed, styles.Dim()))
		row := renderCells(cells, colWidths, i == m.Cursor)
		lines = append(lines, zone.Mark(fmt.Sprintf("history_run_%d", i), framed(row, innerWidth, border)))
	}
	for end-start < tableHeight {
		lines = append(lines, framed(strings.Repeat(" ", innerWidth), innerWidth, border))
		end++
	}
	lines = append(lines, framed(strings.Repeat("─", innerWidth), innerWidth, border))

	// Bottom Response Panel
	if m.Response == nil {
		m.Response = requestinput.NewResponsePanel()
	}
	m.Response.Borderless = true
	responseHeight := max(height-len(lines)-2, 6)
	m.Response.SetSize(innerWidth, responseHeight)
	m.updateSelectedResponse()
	responseView := m.Response.View()
	for _, l := range strings.Split(responseView, "\n") {
		lines = append(lines, l)
	}

	for len(lines) < height-2 {
		lines = append(lines, framed("", innerWidth, border))
	}
	footerText := "  ↑/↓ select   Ctrl+F search   Ctrl+D delete   Tab response tabs   Esc back"
	lines = append(lines, framed(styles.Dim().Render(footerText), innerWidth, border))
	if len(lines) > height-1 {
		lines = lines[:height-1]
	}
	lines = append(lines, border.Render("╰"+strings.Repeat("─", innerWidth)+"╯"))
	return strings.Join(lines, "\n")
}

type tableCell struct {
	text  string
	style *lipgloss.Style
}

func styledCell(text string, style lipgloss.Style) tableCell {
	return tableCell{text: text, style: &style}
}

func renderCells(cells []tableCell, widths []int, selected bool) string {
	var b strings.Builder
	for i, cell := range cells {
		if i >= len(widths) {
			break
		}
		text := " " + ansi.Truncate(cell.text, max(widths[i]-1, 0), "…")
		text += strings.Repeat(" ", max(widths[i]-lipgloss.Width(text), 0))
		switch {
		case selected:
			b.WriteString(styles.Selected().Render(text))
		case cell.style != nil:
			b.WriteString(cell.style.Render(text))
		default:
			b.WriteString(text)
		}
	}
	return b.String()
}

func historyColumns(innerWidth int) (labels []string, widths []int, showTarget, showVars bool) {
	showTarget = innerWidth >= 78
	showVars = innerWidth >= 104

	methodW, statusW, timeW, executedW := 8, 8, 9, 20
	targetW, varsW := 0, 0
	if showTarget {
		targetW = max(innerWidth/6, 12)
	}
	if showVars {
		varsW = 6
	}
	requestW := innerWidth - (methodW + targetW + statusW + timeW + varsW + executedW)
	if requestW < 12 {
		if showVars {
			showVars = false
			varsW = 0
			requestW = innerWidth - (methodW + targetW + statusW + timeW + executedW)
		}
		if requestW < 12 && showTarget {
			showTarget = false
			targetW = 0
			requestW = innerWidth - (methodW + statusW + timeW + executedW)
		}
	}
	requestW = max(requestW, 10)

	labels = []string{"METHOD", "REQUEST"}
	widths = []int{methodW, requestW}
	if showTarget {
		labels = append(labels, "TARGET")
		widths = append(widths, targetW)
	}
	labels = append(labels, "STATUS", "TIME")
	widths = append(widths, statusW, timeW)
	if showVars {
		labels = append(labels, "VARS")
		widths = append(widths, varsW)
	}
	labels = append(labels, "EXECUTED")
	widths = append(widths, executedW)
	return labels, widths, showTarget, showVars
}

func runDetail(run historydb.Run) string {
	collection := run.Collection
	if collection == "" {
		collection = "unsaved"
	}
	parts := []string{
		styles.Bright().Bold(true).Render(collection),
		styles.Muted().Render(display(run.RequestName)),
		styles.Dim().Render("·"),
		styles.Muted().Render(fmt.Sprintf("run #%d", run.ID)),
	}
	if n := len(run.CapturedVariables); n > 0 {
		names := make([]string, 0, min(n, 3))
		for i, variable := range run.CapturedVariables {
			if i == 3 {
				names = append(names, "…")
				break
			}
			names = append(names, variable.Name)
		}
		parts = append(parts,
			styles.Dim().Render("·"),
			styles.Muted().Render(fmt.Sprintf("%d vars: %s", n, strings.Join(names, ", "))),
		)
	}
	return strings.Join(parts, " ")
}

func countCollections(runs []historydb.Run) int {
	seen := make(map[string]struct{})
	for _, run := range runs {
		name := strings.TrimSpace(run.Collection)
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}
	return len(seen)
}
