package ui

import tea "charm.land/bubbletea/v2"

// updateScopeMode menangani layar global scope (allowed origins).
func (m AppModel) updateScopeMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ScreenWidth, m.ScreenHeight = msg.Width, msg.Height
		if m.State != nil {
			m.State.Widht, m.State.Height = msg.Width, msg.Height
		}
		if m.ScopeScreen != nil {
			m.ScopeScreen.SetSize(msg.Width, msg.Height)
		}
		return m, nil
	case tea.KeyMsg:
		if m.ScopeScreen != nil {
			active, newScope := m.ScopeScreen.Update(msg)
			if !active {
				m.ScopeMode = false
				m.recalculateLayout()
			}
			if m.State != nil {
				m.State.GlobalScope = newScope
			}
		}
		return m, nil
	}
	return m, nil
}
