package envscreen

import (
	"strconv"
	"strings"

	"api/internal/ui/styles"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	leftPanelW = 26
	infoPanelW = 38
)

func (m *Model) View() string {
	w := max(m.Width, 100)
	h := max(m.Height, 28)

	header := m.headerView(w)
	status := m.statusView(w)
	footer := m.footerView(w)

	headerH := 1
	statusH := 0
	if status != "" {
		statusH = 1
	}
	panelH := max(h-headerH-statusH-1, 6)

	leftW, infoW, midW := layoutWidths(w)
	inputActive := m.Adding || m.AddingVar || m.EditingVar

	left := renderPanel("ENVIRONMENTS", m.envLines(panelH, leftW), leftW, panelH, m.FocusPane == 0 && !inputActive)
	mid := renderPanel(m.middleTitle(), m.varLines(panelH, midW), midW, panelH, m.FocusPane == 1 && !inputActive)
	info := renderPanel("USAGE & INFO", m.infoLines(), infoW, panelH, false)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", mid, " ", info)

	parts := []string{header}
	if status != "" {
		parts = append(parts, status)
	}
	parts = append(parts, panels, footer)
	return strings.Join(parts, "\n")
}

func layoutWidths(w int) (leftW, infoW, midW int) {
	leftW, infoW = leftPanelW, infoPanelW
	midW = w - leftW - infoW - 2
	if midW < 34 {
		shrink := min(34-midW, infoW-28)
		if shrink > 0 {
			infoW -= shrink
			midW += shrink
		}
	}
	if midW < 26 {
		shrink := min(26-midW, leftW-18)
		if shrink > 0 {
			leftW -= shrink
			midW += shrink
		}
	}
	if midW < 10 {
		midW = 10
	}
	return leftW, infoW, midW
}

// ── header / status / footer ────────────────────────────────────────────────

func (m *Model) headerView(w int) string {
	brand := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Padding(0, 1).
		Render("TermAPI")
	title := lipgloss.NewStyle().
		Foreground(styles.ColTextBright).
		Bold(true).
		Render("ENVIRONMENT VARIABLES")
	left := brand + " " + title

	sep := lipgloss.NewStyle().Foreground(styles.ColBorder).Render(" │ ")
	rightParts := []string{}
	if m.ActiveEnv != "" {
		dot := lipgloss.NewStyle().Foreground(styles.ColSuccess).Render(styles.IconDot)
		env := lipgloss.NewStyle().Foreground(styles.ColTextBright).Bold(true).Render(m.ActiveEnv)
		rightParts = append(rightParts, dot+" "+env)
	} else {
		rightParts = append(rightParts, lipgloss.NewStyle().Foreground(styles.ColTextDim).Render("no active environment"))
	}
	rightParts = append(rightParts, lipgloss.NewStyle().Foreground(styles.ColTextDim).Render(".termapi/envs/"))

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

	switch {
	case m.Adding:
		mode = " NEW ENVIRONMENT "
		keys = []shortcut{
			{"enter", "create"},
			{"esc", "cancel"},
		}
	case m.EditingVar:
		mode = " EDIT VARIABLE "
		keys = []shortcut{
			{"tab", "switch field"},
			{"enter", "save"},
			{"esc", "cancel"},
		}
	case m.AddingVar:
		mode = " NEW VARIABLE "
		keys = []shortcut{
			{"tab", "switch field"},
			{"enter", "save"},
			{"esc", "cancel"},
		}
	case m.FocusPane == 0:
		mode = " ENVIRONMENTS "
		keys = []shortcut{
			{"↑/↓", "navigate"},
			{"enter", "set active"},
			{"a", "new"},
			{"d", "delete"},
			{"tab", "variables"},
		}
	default:
		mode = " VARIABLES "
		keys = []shortcut{
			{"↑/↓", "navigate"},
			{"a", "add"},
			{"e", "edit"},
			{"d", "delete"},
			{"tab", "environments"},
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

	right := descSt.Render("esc close") + sepSt.Render(" │ ") + descSt.Render("ctrl+e close") + " "
	if m.Adding || m.AddingVar || m.EditingVar {
		right = descSt.Render("esc cancel") + sepSt.Render(" │ ") + descSt.Render("ctrl+e close") + " "
	}

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

// ── left panel ──────────────────────────────────────────────────────────────

func (m *Model) envLines(panelH, panelW int) []string {
	innerW := max(panelW-4, 1)
	bodyRows := max(panelH-4, 1)

	muted := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
	dim := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	success := lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true)
	accentHi := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)

	formRows := 0
	if m.Adding {
		formRows = 4
	}
	listRows := max(bodyRows-formRows, 1)

	lines := []string{}
	start, end := visibleWindow(m.EnvCursor, len(m.Envs), listRows)
	for i := start; i < end; i++ {
		name := m.Envs[i]
		active := name == m.ActiveEnv
		selected := i == m.EnvCursor

		prefix := "  "
		switch {
		case selected:
			prefix = "▸ "
		case active:
			prefix = "✓ "
		}
		row := prefix + truncate(name, innerW-2)

		switch {
		case selected:
			lines = append(lines, styles.Selected().Width(innerW).Render(fit(row, innerW)))
		case active:
			lines = append(lines, success.Render(row))
		default:
			lines = append(lines, dim.Render(row))
		}
	}

	if len(m.Envs) == 0 && !m.Adding {
		lines = append(lines, muted.Render("(no environments)"))
		lines = append(lines, dim.Render("Press 'a' to create one."))
	}

	if m.Adding {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, accentHi.Render("NEW ENVIRONMENT"))
		lines = append(lines, truncate(m.InputName.View(), innerW))
		lines = append(lines, muted.Render(truncate("[Enter] create · [Esc] cancel", innerW)))
	}

	return lines
}

// ── middle panel ────────────────────────────────────────────────────────────

func (m *Model) middleTitle() string {
	title := "VARIABLES"
	if m.Current != nil {
		title += " — " + m.Current.Name
		title += " · " + strconv.Itoa(len(m.VarKeys))
	}
	return title
}

func (m *Model) varLines(panelH, panelW int) []string {
	innerW := max(panelW-4, 1)
	bodyRows := max(panelH-4, 1)

	muted := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
	dim := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	text := lipgloss.NewStyle().Foreground(styles.ColText)
	accentHi := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	success := lipgloss.NewStyle().Foreground(styles.ColSuccess).Bold(true)

	if m.AddingVar || m.EditingVar {
		label := "NEW VARIABLE"
		if m.EditingVar {
			label = "EDIT VARIABLE"
		}
		keyLabel := muted.Render("KEY    ")
		valLabel := muted.Render("VALUE  ")
		if m.InputFocus == 0 {
			keyLabel = success.Render("KEY    ")
		} else {
			valLabel = success.Render("VALUE  ")
		}
		lines := []string{
			"",
			accentHi.Render(label),
			"",
			keyLabel + truncate(m.InputKey.View(), max(innerW-7, 4)),
			valLabel + truncate(m.InputVal.View(), max(innerW-7, 4)),
			"",
			muted.Render(truncate("[Tab] switch field · [Enter] save · [Esc] cancel", innerW)),
		}
		return lines
	}

	if m.Current == nil {
		return []string{
			muted.Render("(no environment selected)"),
			dim.Render("Select an environment on the left."),
		}
	}

	keyW := max(innerW/2-2, 10)
	if keyW > innerW-8 {
		keyW = max(innerW-8, 4)
	}
	valW := max(innerW-keyW-1, 1)

	lines := []string{
		muted.Render(fit("KEY", keyW) + " " + "VALUE"),
		dim.Render(strings.Repeat("─", innerW)),
	}

	rows := max(bodyRows-2, 1)
	start, end := visibleWindow(m.VarCursor, len(m.VarKeys), rows)
	for i := start; i < end; i++ {
		key := m.VarKeys[i]
		val := m.Current.Variables[key]
		keyCell := padRight(truncate(key, keyW), keyW)
		valCell := truncate(val, valW)

		if i == m.VarCursor && m.FocusPane == 1 {
			lines = append(lines, styles.Selected().Width(innerW).Render(fit(keyCell+" "+valCell, innerW)))
			continue
		}
		lines = append(lines, text.Render(keyCell)+" "+muted.Render(valCell))
	}

	if len(m.VarKeys) == 0 {
		lines = append(lines, muted.Render("(no variables — press 'a' to add)"))
	}

	return lines
}

// ── right panel ─────────────────────────────────────────────────────────────

func (m *Model) infoLines() []string {
	section := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
	dim := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	value := lipgloss.NewStyle().Foreground(styles.ColTextBright)
	warn := lipgloss.NewStyle().Foreground(styles.ColWarning).Bold(true)

	lines := []string{
		section.Render("SYNTAX"),
		muted.Render("  Use {{KEY}} in URLs, params,"),
		muted.Render("  headers, cookies, body, auth."),
		"",
		section.Render("PRIORITY"),
		muted.Render("  Session variables override"),
		muted.Render("  environment variables."),
		"",
		section.Render("STORAGE"),
		muted.Render("  .termapi/envs/<name>.yaml"),
		muted.Render("  Stored as plaintext YAML."),
		"",
		section.Render("ACTIVE"),
	}

	if m.ActiveEnv != "" {
		lines = append(lines, "  "+value.Render(m.ActiveEnv))
	} else {
		lines = append(lines, dim.Render("  none — press Enter to set"))
	}

	lines = append(lines,
		"",
		warn.Render("SECURITY"),
		muted.Render("  Never commit secrets."),
		muted.Render("  Use Global Scope (Ctrl+G)"),
		muted.Render("  to limit credential sharing."),
		"",
		section.Render("KEYS"),
		muted.Render("  ↑/↓    move cursor"),
		muted.Render("  Tab    switch pane"),
		muted.Render("  Enter  activate / edit"),
		muted.Render("  a      add env / variable"),
		muted.Render("  e      edit variable"),
		muted.Render("  d      delete item"),
		muted.Render("  Esc    close"),
		muted.Render("  Ctrl+E close"),
	)
	return lines
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
