package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"api/internal/ui/nav"
	requestinput "api/internal/ui/requestInput"
	"api/internal/ui/styles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// View merender layar sesuai mode aktif (overlay) atau layout utama.
func (m AppModel) View() tea.View {
	if m.ScreenWidth > 0 && (m.ScreenWidth < minWindowWidth || m.ScreenHeight < minWindowHeight) {
		return m.viewTooSmall()
	}
	if m.NewItemMode && m.NewItemScreen != nil {
		return m.viewNewItem()
	}
	if m.SaveDialog {
		p := tea.NewView(zone.Scan(m.saveDialogView()))
		p.AltScreen = true
		return p
	}
	if m.SearchMode {
		return m.viewSearch()
	}
	if m.BatchChoosing {
		p := tea.NewView(zone.Scan(m.batchChooserView()))
		p.AltScreen = true
		return p
	}
	if m.BatchMode {
		return m.viewBatchResults()
	}
	if m.HistoryMode && m.HistoryScreen != nil {
		p := tea.NewView(zone.Scan(m.HistoryScreen.View()))
		p.AltScreen = true
		p.MouseMode = tea.MouseModeCellMotion
		return p
	}
	if m.ScopeMode && m.ScopeScreen != nil {
		p := tea.NewView(zone.Scan(m.ScopeScreen.View()))
		p.AltScreen = true
		p.MouseMode = tea.MouseModeCellMotion
		return p
	}
	if m.EnvMode && m.EnvScreen != nil {
		p := tea.NewView(zone.Scan(m.EnvScreen.View()))
		p.AltScreen = true
		p.MouseMode = tea.MouseModeCellMotion
		return p
	}
	if m.HelpMode && m.HelpScreen != nil {
		p := tea.NewView(zone.Scan(m.HelpScreen.View()))
		p.AltScreen = true
		p.MouseMode = tea.MouseModeCellMotion
		return p
	}
	if m.DiffMode && m.DiffScreen != nil {
		p := tea.NewView(zone.Scan(m.DiffScreen.View()))
		p.AltScreen = true
		p.MouseMode = tea.MouseModeCellMotion
		return p
	}
	if m.WireMode && m.WireScreen != nil {
		p := tea.NewView(zone.Scan(m.WireScreen.View().Content))
		p.AltScreen = true
		p.MouseMode = tea.MouseModeCellMotion
		return p
	}
	return m.viewMain()
}

func (m AppModel) viewTooSmall() tea.View {
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555")).
		Bold(true).
		Align(lipgloss.Center)

	warningText := msgStyle.Render(fmt.Sprintf(
		"⚠️  WINDOW TERLALU KECIL\n\nTermAPI memerlukan lebar minimal 105 kolom dan tinggi 32 baris.\n(Ukuran saat ini: %dx%d)",
		m.ScreenWidth, m.ScreenHeight,
	))

	centeredBox := lipgloss.NewStyle().
		Width(max(m.ScreenWidth, 1)).
		Height(max(m.ScreenHeight, 1)).
		Align(lipgloss.Center, lipgloss.Center).
		Render(warningText)

	p := tea.NewView(centeredBox)
	p.AltScreen = true
	p.MouseMode = tea.MouseModeCellMotion
	return p
}

func (m AppModel) viewBatchResults() tea.View {
	panel := m.batchView()
	footer := m.batchFooterView()
	if m.Response != nil {
		height := m.ScreenHeight - lipgloss.Height(panel) - lipgloss.Height(footer) - 1
		m.Response.SetSize(m.ScreenWidth, max(7, height))
	}
	response := ""
	if m.Response != nil {
		response = m.Response.View()
	}

	content := lipgloss.JoinVertical(lipgloss.Left, panel, response, footer)
	// Pastikan panel selalu mengisi tinggi layar penuh agar footer menempel di bawah.
	if h := lipgloss.Height(content); h < m.ScreenHeight {
		content += strings.Repeat("\n", m.ScreenHeight-h)
	}
	content = lipgloss.NewStyle().MaxHeight(max(m.ScreenHeight, 1)).Render(content)
	p := tea.NewView(zone.Scan(content))
	p.AltScreen = true
	p.MouseMode = tea.MouseModeCellMotion
	return p
}

func (m AppModel) viewMain() tea.View {
	nav.NavbarWidth = m.ScreenWidth
	if m.State != nil {
		nav.ActiveEnv = m.State.ActiveEnv
		nav.GlobalScope = m.State.GlobalScope
	}
	navbar := nav.NavbarContent()

	sidebarView := ""
	if m.Sidebar != nil {
		sidebarView = m.Sidebar.View()
	}

	reqHasil := requestinput.RequestInputContent(m.ReqState)

	leftWidth := 0
	if m.Sidebar != nil {
		leftWidth = m.Sidebar.Widht
	}
	if m.ReqState != nil {
		m.ReqState.SetWidth(m.State.Widht - leftWidth)
	}

	navHeight := lipgloss.Height(navbar)
	mainHeight := max(m.ScreenHeight-navHeight-1, 1)

	if m.Response != nil && m.ReqState != nil {
		reqContentHeight := lipgloss.Height(reqHasil) + lipgloss.Height(m.ReqState.EditorView())
		responseHeight := max(mainHeight-reqContentHeight, 5)
		m.Response.SetSize(m.contentWidth(), responseHeight)
	}

	rightContent := lipgloss.JoinVertical(lipgloss.Left, reqHasil, m.ReqState.EditorView())
	if m.Response != nil {
		rightContent = lipgloss.JoinVertical(lipgloss.Left, rightContent, m.Response.View())
	}
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightContent)
	main = lipgloss.NewStyle().MaxHeight(mainHeight).Render(main)

	layout := lipgloss.JoinVertical(lipgloss.Left, navbar, main)

	if m.SaveError != "" {
		layout = lipgloss.JoinVertical(lipgloss.Left, layout,
			lipgloss.NewStyle().Foreground(styles.ColError).Render("Save failed: "+m.SaveError))
	}

	layout = lipgloss.JoinVertical(lipgloss.Left, layout, m.mainFooterView())

	p := tea.NewView(zone.Scan(layout))
	p.AltScreen = true
	p.MouseMode = tea.MouseModeCellMotion
	return p
}

// mainFooterView merender status bar satu baris penuh gaya lazygit/k9s:
// mode badge (inverted) di kiri, shortcut kontekstual di tengah,
// status response + env di kanan. Selalu tepat satu baris, tidak terpotong.
func (m AppModel) mainFooterView() string {
	keyStyle := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	sepStyle := lipgloss.NewStyle().Foreground(styles.ColBorder)

	type shortcut struct {
		key  string
		desc string
	}

	var mode string
	var keys []shortcut

	sidebarFocused := m.State != nil && m.State.TabActive == 1
	focusArea := 0
	if m.ReqState != nil {
		focusArea = m.ReqState.FocusArea
	}
	responseFocused := m.Response != nil && m.Response.Focused

	switch {
	case sidebarFocused:
		mode = " SIDEBAR "
		keys = []shortcut{
			{"j/k", "navigate"},
			{"enter", "open"},
			{"space", "toggle"},
			{"^n", "new req"},
			{"^a", "new folder"},
			{"^d", "delete"},
			{"^f", "search"},
		}
	case responseFocused:
		mode = " RESPONSE "
		keys = []shortcut{
			{"tab", "tab"},
			{"^f", "find"},
			{"n/N", "next/prev"},
			{"y", "copy"},
			{"^y", "copy all"},
			{"esc", "back"},
		}
	case focusArea == 3:
		mode = " EDITOR "
		switch {
		case m.ReqState.Tab == 2 && (m.ReqState.Body.Type == "form" || m.ReqState.Body.Type == "multipart"):
			keys = []shortcut{
				{"enter", "edit"},
				{"a", "add"},
				{"d", "delete"},
				{"space", "on/off"},
				{"t", "type"},
				{"^t", "tab"},
			}
		case m.ReqState.Tab == 2:
			keys = []shortcut{
				{"^b", "type"},
				{"f6", "format"},
				{"f8", "validate"},
				{"tab", "indent"},
				{"^t", "tab"},
			}
		case m.ReqState.Tab == 3:
			keys = []shortcut{
				{"enter", "edit"},
				{"^t", "tab"},
				{"^s", "save"},
				{"^n", "new"},
			}
		default:
			keys = []shortcut{
				{"enter", "edit"},
				{"a", "add"},
				{"d", "delete"},
				{"space", "on/off"},
				{"^r", "raw"},
				{"^t", "tab"},
			}
		}
	case focusArea == 1:
		mode = " URL "
		keys = []shortcut{
			{"enter", "send"},
			{"tab", "next"},
			{"^s", "save"},
			{"^n", "new"},
		}
	case focusArea == 2:
		mode = " SEND "
		keys = []shortcut{
			{"enter", "send"},
			{"tab", "next"},
		}
	default:
		mode = " METHOD "
		keys = []shortcut{
			{"←/→", "change"},
			{"enter", "select"},
			{"tab", "next"},
		}
	}

	// Mode badge: fundo sólido invertido, estilo lazygit/k9s.
	modeBadge := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Render(mode)

	parts := []string{}
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k.key)+descStyle.Render(" "+k.desc))
	}
	sep := sepStyle.Render(" │ ")

	joinMiddle := func(ps []string) string {
		if len(ps) == 0 {
			return ""
		}
		return " " + strings.Join(ps, sep)
	}

	// Status response terakhir (kanan): badge status code + waktu, atau placeholder.
	var statusChunk string
	if m.Response != nil && m.Response.Status != "" && m.Response.Err == "" {
		codeBadge := styles.StatusBadge(m.Response.Code).Render(strconv.Itoa(m.Response.Code))
		statusChunk = codeBadge + descStyle.Render(" "+m.Response.Time.Round(time.Millisecond).String())
	} else if m.Response != nil && m.Response.Err != "" {
		statusChunk = lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).Render("ERR")
	}

	envChunk := ""
	if m.State != nil && m.State.ActiveEnv != "" {
		dot := lipgloss.NewStyle().Foreground(styles.ColSuccess).Render(styles.IconDot)
		envChunk = dot + descStyle.Render(" "+m.State.ActiveEnv)
	}

	rightParts := []string{}
	if statusChunk != "" {
		rightParts = append(rightParts, statusChunk)
	}
	if envChunk != "" {
		rightParts = append(rightParts, envChunk)
	}
	helpText := zone.Mark("help_button", descStyle.Render("F1 help"))
	rightParts = append(rightParts, helpText+sepStyle.Render(" │ ")+descStyle.Render("^q quit"))
	right := " " + strings.Join(rightParts, sepStyle.Render("  │  ")) + " "

	left := modeBadge
	middle := joinMiddle(parts)
	gap := m.ScreenWidth - lipgloss.Width(left) - lipgloss.Width(middle) - lipgloss.Width(right)
	for len(parts) > 1 && gap < 1 {
		parts = parts[:len(parts)-1]
		middle = joinMiddle(parts)
		gap = m.ScreenWidth - lipgloss.Width(left) - lipgloss.Width(middle) - lipgloss.Width(right)
	}
	if gap < 0 {
		gap = 0
	}
	bar := left + middle + strings.Repeat(" ", gap) + right

	return lipgloss.NewStyle().
		Background(styles.ColStatusBg).
		MaxHeight(1).
		MaxWidth(m.ScreenWidth).
		Render(bar)
}
