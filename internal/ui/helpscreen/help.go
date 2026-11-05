package helpscreen

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type Model struct {
	Width  int
	Height int
	Scroll int
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	m.clampScroll()
}

// viewportHeight adalah jumlah baris isi yang tampil. Sisanya dipakai border,
// padding vertikal, baris petunjuk, dan sedikit margin.
func (m *Model) viewportHeight() int {
	return max(m.Height-7, 3)
}

func (m *Model) clampScroll() {
	maxScroll := max(len(m.lines())-m.viewportHeight(), 0)
	if m.Scroll < 0 {
		m.Scroll = 0
	}
	if m.Scroll > maxScroll {
		m.Scroll = maxScroll
	}
}

// ScrollBy menggeser isi bantuan beberapa baris.
func (m *Model) ScrollBy(delta int) {
	if m == nil {
		return
	}
	m.Scroll += delta
	m.clampScroll()
}

// ScrollPage menggeser isi bantuan satu layar penuh.
func (m *Model) ScrollPage(delta int) {
	if m == nil {
		return
	}
	m.ScrollBy(delta * m.viewportHeight())
}

func (m *Model) ScrollTop() {
	if m == nil {
		return
	}
	m.Scroll = 0
}

func (m *Model) ScrollBottom() {
	if m == nil {
		return
	}
	m.Scroll = len(m.lines())
	m.clampScroll()
}

func (m *Model) View() string {
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7A8C")).Align(lipgloss.Center)
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#56C2D6")).
		Padding(1, 3).
		Background(lipgloss.Color("#131A21"))

	lines := m.lines()
	viewport := m.viewportHeight()
	maxScroll := max(len(lines)-viewport, 0)
	if m.Scroll > maxScroll {
		m.Scroll = maxScroll
	}
	if m.Scroll < 0 {
		m.Scroll = 0
	}
	end := min(m.Scroll+viewport, len(lines))
	visible := append([]string{}, lines[m.Scroll:end]...)
	footer := subStyle.Render("↑/↓ gulir · PgUp/PgDn · g/G atas/bawah · Esc tutup")
	content := append(visible, "", footer)

	boxWidth := min(max(m.Width-10, 60), 90)
	box := borderStyle.Width(boxWidth).Render(strings.Join(content, "\n"))

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, box)
}

func (m *Model) lines() []string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#56C2D6")).Align(lipgloss.Center)
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7A8C")).Align(lipgloss.Center)

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8ADCEB")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#C9D3DE"))
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D8B26E")).Bold(true)

	var content []string
	content = append(content, titleStyle.Render("[Alt+?] TERMAPI - HELP & KEYBINDINGS"))
	content = append(content, subStyle.Render("Lightning-fast TUI API Client"))
	content = append(content, "")

	sections := []struct {
		title    string
		bindings [][2]string
	}{
		{
			"Global Navigation & Actions",
			[][2]string{
				{"Tab", "Switch focus between Sidebar and Main Content"},
				{"F1", "Open this Help screen anywhere"},
				{"Alt+?", "Open this Help screen anywhere"},
				{"Click help", "Open this Help screen"},
				{"Esc / Enter / Q", "Close this Help screen"},
				{"Ctrl+Q", "Quit TermAPI"},
				{"Ctrl+N", "New request / folder screen"},
				{"Ctrl+S", "Save active request"},
				{"Ctrl+W", "Open Wire / Network Inspector"},
				{"Ctrl+O", "Open request history"},
				{"Ctrl+D", "Delete selected request (sidebar) / open Diff Viewer"},
				{"Ctrl+G", "Manage Global Scope"},
				{"Ctrl+E", "Manage Environments"},
				{"Ctrl+L", "Run Collection / Batch Runner"},
				{"Ctrl+F", "Search collection files and folders"},
				{"p", "Pin / unpin selected request (sidebar)"},
				{"Ctrl+↑ / Ctrl+↓", "Resize response panel"},
			},
		},
		{
			"Request Editor & URL Bar",
			[][2]string{
				{"Enter", "Send request"},
				{"← / →", "Change HTTP method (GET, POST, PUT, DELETE, etc.)"},
				{"Ctrl+T", "Cycle request tabs (Params/Headers/Body/Auth/Cookies)"},
				{"Ctrl+R", "Toggle Raw mode for Params and Headers"},
				{"Ctrl+B", "Cycle Body type (JSON, Form, Multipart, Binary, GraphQL, ...)"},
				{"Enter / Click", "Edit the selected key/value cell"},
				{"a / d", "Add / delete a key/value row"},
				{"Space", "Enable or disable a key/value row"},
				{"t", "Change type in Auth or a multipart row"},
				{"s", "Insert a suggested header name (Headers tab)"},
				{"F6 / F7", "Prettify / minify JSON body"},
				{"F8", "Validate JSON or XML body"},
				{"Tab / Shift+Tab", "Move between cells (keeps editing)"},
				{"/ (Slash)", "Focus JSON response filter (gjson)"},
			},
		},
		{
			"History",
			[][2]string{
				{"↑ / ↓ or j / k", "Select a history run"},
				{"Ctrl+F", "Search history by request, URL, method, status"},
				{"Ctrl+D", "Delete the selected history run"},
				{"Tab / Click", "Switch response Body / Cookies / Headers"},
				{"PgUp / PgDn", "Move selection a page at a time"},
				{"Esc", "Back to the main screen"},
			},
		},
		{
			"Response Viewer",
			[][2]string{
				{"Click / drag", "Focus the response and select lines"},
				{"Tab / Shift+Tab", "Switch Body / Cookies / Headers"},
				{"Ctrl+F", "Find text in the active response tab"},
				{"n / N", "Jump to the next / previous match"},
				{"y / Ctrl+Y", "Copy selection / copy the whole tab"},
				{"Esc", "Leave response focus or clear selection"},
				{"PageUp / PageDown", "Scroll response view"},
				{"Ctrl+U / Ctrl+D", "Scroll response view (fast)"},
			},
		},
		{
			"Diff Viewer (A vs B)",
			[][2]string{
				{"Ctrl+D", "Open baseline (A) vs latest response (B)"},
				{"b", "Pick another baseline from this request's history"},
				{"j / k · ↑ / ↓", "Scroll one line"},
				{"PgUp / PgDn", "Scroll one page"},
				{"g / G", "Jump to top / bottom"},
				{"n / p", "Next / previous change block (hunk)"},
				{"e", "Expand / collapse unchanged blocks"},
				{"m", "Switch Side-by-Side / Unified"},
				{"Mouse wheel", "Scroll"},
				{"? / F1", "Show the in-app diff guide"},
				{"Esc / q", "Close Diff Viewer"},
			},
		},
	}

	for _, sec := range sections {
		content = append(content, sectionStyle.Render(sec.title))
		for _, b := range sec.bindings {
			keyCol := keyStyle.Render(padRight(b[0], 22))
			descCol := descStyle.Render(b[1])
			content = append(content, "  "+keyCol+"  "+descCol)
		}
		content = append(content, "")
	}

	return content
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
