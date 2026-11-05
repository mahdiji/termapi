package ui

import tea "charm.land/bubbletea/v2"

// updateEnvMode menangani layar environment variables.
func (m AppModel) updateEnvMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ScreenWidth, m.ScreenHeight = msg.Width, msg.Height
		if m.State != nil {
			m.State.Widht, m.State.Height = msg.Width, msg.Height
		}
		if m.EnvScreen != nil {
			m.EnvScreen.SetSize(msg.Width, msg.Height)
		}
		return m, nil
	case tea.KeyMsg:
		if m.EnvScreen != nil {
			active, newEnv, vars := m.EnvScreen.Update(msg)
			if !active {
				m.EnvMode = false
				m.recalculateLayout()
			}
			if m.State != nil {
				m.State.ActiveEnv = newEnv
				m.State.EnvVars = vars
			}
		}
		return m, nil
	}
	return m, nil
}
