package requestinput

import (
	"fmt"
	"strings"

	"api/internal/collectionstore"
	"api/internal/ui/styles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

const requestTabCount = 5

func (m *Model) updateFocusState() {
	if m.FocusArea == 1 {
		m.Input.Focus()
	} else if m.FocusArea == 3 {
		m.focusActiveTab()
	} else {
		m.Input.Blur()
		m.blurTabs()
	}
}

func (m *Model) activeKV() *KVEditor {
	switch m.Tab {
	case 0:
		return &m.Params
	case 1:
		return &m.Headers
	case 2:
		if m.Body.Type == "form" || m.Body.Type == "multipart" {
			return &m.Body.Fields
		}
	case 4:
		return &m.Cookies
	}
	return nil
}

func (m *Model) blurTabs() {
	m.Params.Blur()
	m.Headers.Blur()
	m.Body.Blur()
	m.Auth.Blur()
	m.Cookies.Blur()
}

func (m *Model) focusActiveTab() tea.Cmd {
	m.Input.Blur()
	switch m.Tab {
	case 0:
		return m.Params.Focus()
	case 1:
		return m.Headers.Focus()
	case 2:
		return m.Body.Focus()
	case 3:
		return m.Auth.Focus()
	case 4:
		return m.Cookies.Focus()
	}
	return nil
}

func (m *Model) updateActiveTab(msg tea.Msg) tea.Cmd {
	switch m.Tab {
	case 0:
		return m.Params.Update(msg)
	case 1:
		return m.Headers.Update(msg)
	case 2:
		return m.Body.Update(msg)
	case 3:
		return m.Auth.Update(msg)
	case 4:
		return m.Cookies.Update(msg)
	}
	return nil
}

func (m *Model) commitActiveEdit() {
	switch m.Tab {
	case 0:
		m.Params.commitIfEditing()
	case 1:
		m.Headers.commitIfEditing()
	case 2:
		m.Body.commitIfEditing()
	case 3:
		m.Auth.commitIfEditing()
	case 4:
		m.Cookies.commitIfEditing()
	}
}

// CommitActiveEdit menyimpan nilai sel yang sedang diedit. Dipakai saat
// fokus berpindah keluar dari request editor (mis. klik sidebar/response).
func (m *Model) CommitActiveEdit() {
	m.commitActiveEdit()
}

func (m *Model) activeTabView() string {
	switch m.Tab {
	case 0:
		return m.Params.View()
	case 1:
		return m.Headers.View()
	case 2:
		return m.Body.View()
	case 3:
		return m.Auth.View()
	case 4:
		return m.Cookies.View()
	}
	return ""
}

func (m *Model) SetEditorSize(width, height int) {
	if height < 3 {
		height = 3
	}
	m.EditorHeight = height
	// Editor dirender di dalam panel berbingkai; lebar yang dipakai tabel
	// harus lebar konten (setelah border + padding) agar kolom terakhir tidak
	// terpotong dan tetap bisa diklik.
	inner := max(width-styles.BaseBoxStyle.GetHorizontalFrameSize(), 20)
	m.Params.SetSize(inner, height)
	m.Headers.SetSize(inner, height)
	m.Body.SetSize(inner, height)
	m.Auth.SetSize(inner, height)
	m.Cookies.SetSize(inner, height)
}

// countNonEmptyLines menghitung baris non-kosong untuk badge tab.
func countNonEmptyLines(s string) int {
	if s == "" {
		return 0
	}
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

func (m *Model) bodyCount() int {
	switch m.Body.Type {
	case "none":
		return 0
	case "form", "multipart":
		return m.Body.Fields.Count()
	default:
		if strings.TrimSpace(m.Body.Raw.Value()) != "" || m.Body.File.Value() != "" {
			return 1
		}
	}
	return 0
}

func (m *Model) authCount() int {
	if m.Auth.Type == "none" || m.Auth.Type == "" {
		return 0
	}
	return 1
}

func (m *Model) EditorView() string {
	// Tab gaya lazygit: tab aktif = pill solid aksen; non-aktif = teks redup.
	type tabSpec struct {
		name  string
		count int
	}
	tabs := []tabSpec{
		{name: "Params", count: m.Params.Count()},
		{name: "Headers", count: m.Headers.Count()},
		{name: "Body", count: m.bodyCount()},
		{name: "Auth", count: m.authCount()},
		{name: "Cookies", count: m.Cookies.Count()},
	}
	parts := make([]string, 0, len(tabs))
	for i, t := range tabs {
		name := " " + t.name
		if t.count > 0 {
			name += fmt.Sprintf(" (%d)", t.count)
		}
		name += " "
		var style lipgloss.Style
		if i == m.Tab {
			style = lipgloss.NewStyle().
				Foreground(styles.ColBackground).
				Background(styles.ColAccent).
				Bold(true)
		} else {
			style = lipgloss.NewStyle().Foreground(styles.ColTextMuted)
		}
		parts = append(parts, zone.Mark(fmt.Sprintf("request_tab_%d", i), style.Render(name)))
	}
	tabsLine := lipgloss.JoinHorizontal(lipgloss.Center, parts...)
	active := m.activeTabView()
	box := styles.BaseBoxStyle
	if m.FocusArea == 3 && (m.State == nil || m.State.TabActive == 0) {
		box = styles.ActiveBoxStyle
	}
	boxWidth := max(m.lastWidth, 1)
	return lipgloss.JoinVertical(lipgloss.Left, " "+tabsLine, zone.Mark("request_editor", box.Width(boxWidth).MaxWidth(boxWidth).Height(m.EditorHeight+2).Render(active)))
}

func (m *Model) Focus() tea.Cmd {
	m.Focused = true
	m.FocusArea = 1
	return m.Input.Focus()
}

func (m *Model) Blur() {
	m.commitActiveEdit()
	m.Focused = false
	m.Input.Blur()
}

func (m *Model) SetFocused(focused bool) {
	m.Focused = focused
	if focused {
		m.Input.Focus()
		m.FocusArea = 1
	} else {
		m.Input.Blur()
	}
}

func (m *Model) Value() string {
	return m.Input.Value()
}

// ResolvedURL mengembalikan URL dengan variabel environment/session yang
// sudah diganti, sama seperti saat request dikirim. Dipakai untuk mencocokkan
// history run dengan request yang sedang aktif.
func (m *Model) ResolvedURL() string {
	if m == nil {
		return ""
	}
	return m.variableReplacer().Replace(m.Input.Value())
}

func (m *Model) SetValue(val string) {
	m.Input.SetValue(val)
}

// LoadRequest fills every request editor from a saved collection entry.
func (m *Model) LoadRequest(request *collectionstore.Request) {
	if m == nil || request == nil {
		return
	}
	request.Normalize()
	m.Method = request.Method
	m.Input.SetValue(request.URL)
	m.Params.SetRows(request.ParamsList)
	m.Headers.SetRows(request.HeadersList)
	m.Body.SetSpec(request.BodySpec)
	m.Auth.SetSpec(request.Auth)
	m.Cookies.SetRows(request.Cookies)
	m.Tab = 0
}

// Snapshot extracts the current editor contents in the persisted format.
func (m *Model) Snapshot() (collectionstore.Request, error) {
	m.commitActiveEdit()
	m.syncParamsFromURL()
	params, err := m.Params.EffectiveRows()
	if err != nil {
		return collectionstore.Request{}, fmt.Errorf("params: %w", err)
	}
	headers, err := m.Headers.EffectiveRows()
	if err != nil {
		return collectionstore.Request{}, fmt.Errorf("headers: %w", err)
	}
	cookies, err := m.Cookies.EffectiveRows()
	if err != nil {
		return collectionstore.Request{}, fmt.Errorf("cookies: %w", err)
	}
	return collectionstore.Request{
		Method:      m.Method,
		URL:         m.Input.Value(),
		ParamsList:  params,
		HeadersList: headers,
		BodySpec:    m.Body.Spec(),
		Auth:        m.Auth.Spec(),
		Cookies:     cookies,
	}, nil
}
