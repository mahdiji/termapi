package historyscreen

import (
	tea "charm.land/bubbletea/v2"
)

func (m *Model) Update(msg tea.KeyMsg) bool {
	m.Notice = ""
	if m.Search.Focused() {
		if msg.String() == "enter" || msg.String() == "esc" {
			m.Search.Blur()
			return true
		}
		m.Search, _ = m.Search.Update(msg)
		m.Query = m.Search.Value()
		m.applySearch()
		return true
	}
	if len(m.Runs) == 0 {
		return false
	}
	listRows := max((m.Height-12)/2, 1)
	switch msg.String() {
	case "up", "k":
		m.Cursor = max(m.Cursor-1, 0)
		m.updateSelectedResponse()
		return true
	case "down", "j":
		m.Cursor = min(m.Cursor+1, len(m.Runs)-1)
		m.updateSelectedResponse()
		return true
	case "pgup", "ctrl+u":
		m.Cursor = max(m.Cursor-listRows, 0)
		m.updateSelectedResponse()
		return true
	case "pgdown", "ctrl+d":
		m.Cursor = min(m.Cursor+listRows, len(m.Runs)-1)
		m.updateSelectedResponse()
		return true
	case "home":
		m.Cursor = 0
		m.updateSelectedResponse()
		return true
	case "end":
		m.Cursor = max(len(m.Runs)-1, 0)
		m.updateSelectedResponse()
		return true
	case "tab":
		// If response panel is active, let it handle tab or switch
		return false
	}
	return false
}
