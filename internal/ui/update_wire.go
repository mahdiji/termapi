package ui

import (
	"api/internal/ui/wirescreen"
	tea "charm.land/bubbletea/v2"
)

func (m *AppModel) openWireScreen() tea.Cmd {
	if m == nil {
		return nil
	}
	url := "ws://localhost:8080/ws"
	if m.ReqState != nil && m.ReqState.Input.Value() != "" {
		url = m.ReqState.Input.Value()
	}
	m.WireScreen = wirescreen.New(m.State, url)
	if m.ScreenWidth > 0 && m.ScreenHeight > 0 {
		newModel, _ := m.WireScreen.Update(tea.WindowSizeMsg{Width: m.ScreenWidth, Height: m.ScreenHeight})
		if wsModel, ok := newModel.(*wirescreen.Model); ok {
			m.WireScreen = wsModel
		}
	}
	m.WireMode = true
	return m.WireScreen.Init()
}

func (m AppModel) updateWireMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Esc menutup layar hanya jika editor sedang tidak fokus;
		// saat editor fokus, Esc dipakai untuk keluar dari editor.
		if msg.String() == "esc" && (m.WireScreen == nil || !m.WireScreen.EditorFocused()) {
			m.WireMode = false
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.ScreenWidth = msg.Width
		m.ScreenHeight = msg.Height
		if m.WireScreen != nil {
			newModel, cmd := m.WireScreen.Update(msg)
			if wsModel, ok := newModel.(*wirescreen.Model); ok {
				m.WireScreen = wsModel
			}
			return m, cmd
		}
	}

	if m.WireScreen != nil {
		newModel, cmd := m.WireScreen.Update(msg)
		if wsModel, ok := newModel.(*wirescreen.Model); ok {
			m.WireScreen = wsModel
		}
		return m, cmd
	}
	return m, nil
}
