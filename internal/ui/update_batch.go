package ui

import (
	tea "charm.land/bubbletea/v2"
)

// updateBatchChoosing menangani pilihan folder/depth sebelum batch dijalankan.
func (m AppModel) updateBatchChoosing(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+f":
		return m, m.openFileSearch()
	case "esc", "backspace":
		m.BatchChoosing = false
		return m, nil
	case "left", "h":
		if m.BatchDepth > 0 {
			m.BatchDepth--
		}
		return m, nil
	case "right", "l":
		if m.BatchDepth < m.BatchMaxDepth {
			m.BatchDepth++
		}
		return m, nil
	case "c":
		m.BatchChain = !m.BatchChain
		return m, nil
	case "r":
		m.BatchRetries = (m.BatchRetries + 1) % 4
		return m, nil
	case "enter":
		return m, m.beginBatch()
	default:
		return m, nil
	}
}

// updateBatchMode menangani navigasi hasil batch setelah eksekusi.
func (m AppModel) updateBatchMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
		case "ctrl+f":
			return m, m.openFileSearch()
		case "esc", "backspace":
			m.BatchMode = false
			m.recalculateLayout()
			return m, nil
		case "up", "k":
			if m.BatchSelected > 0 {
				m.BatchSelected--
			}
			if m.Response != nil {
				m.showBatchSelection()
			}
			return m, nil
		case "tab":
			if m.Response != nil {
				m.Response.Tab = (m.Response.Tab + 1) % 3
			}
			return m, nil
		case "shift+tab":
			if m.Response != nil {
				m.Response.Tab = (m.Response.Tab + 2) % 3
			}
			return m, nil
		case "down", "j":
			if m.BatchSelected < len(m.BatchResults)-1 {
				m.BatchSelected++
			}
			if len(m.BatchResults) > 0 && m.Response != nil {
				m.showBatchSelection()
			}
			return m, nil
		}
	case tea.MouseClickMsg:
		if m.Response != nil {
			m.Response.HandleMouse(key)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.ScreenWidth, m.ScreenHeight = key.Width, key.Height
		if m.State != nil {
			m.State.Widht, m.State.Height = key.Width, key.Height
		}
		return m, nil
	}
	return m, nil
}

func (m *AppModel) showBatchSelection() {
	if m == nil || m.Response == nil || m.BatchSelected < 0 || m.BatchSelected >= len(m.BatchResults) {
		return
	}
	r := m.BatchResults[m.BatchSelected]
	if r.RequestName == "" && r.URL == "" {
		m.Response.SetLoading()
		if m.BatchSelected < len(m.BatchNames) {
			m.Response.Method = m.BatchNames[m.BatchSelected]
		}
		return
	}
	m.Response.SetResponse(r)
}
