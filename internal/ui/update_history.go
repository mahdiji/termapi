package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// updateHistoryMode menangani layar history (keyboard, mouse, resize).
func (m AppModel) updateHistoryMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ScreenWidth, m.ScreenHeight = msg.Width, msg.Height
		if m.State != nil {
			m.State.Widht, m.State.Height = msg.Width, msg.Height
		}
		if m.HistoryScreen != nil {
			m.HistoryScreen.SetSize(msg.Width, msg.Height)
		}
		return m, nil
	case tea.KeyMsg:
		if m.HistoryScreen != nil && m.HistoryScreen.Search.Focused() {
			if msg.String() == "esc" {
				m.HistoryScreen.StopSearch()
				return m, nil
			}
			m.HistoryScreen.Update(msg)
			return m, nil
		}
		if msg.String() == "esc" {
			m.HistoryMode = false
			m.recalculateLayout()
			return m, nil
		}
		if msg.String() == "ctrl+f" && m.HistoryScreen != nil {
			return m, m.HistoryScreen.StartSearch()
		}
		if msg.String() == "ctrl+d" && m.HistoryScreen != nil {
			m.deleteSelectedHistoryRun()
			return m, nil
		}
		if m.HistoryScreen != nil {
			m.HistoryScreen.Update(msg)
		}
		return m, nil
	case tea.MouseClickMsg:
		if m.HistoryScreen != nil {
			m.HistoryScreen.HandleMouse(msg)
		}
		return m, nil
	case tea.MouseWheelMsg:
		if m.HistoryScreen != nil {
			m.HistoryScreen.HandleWheel(msg)
		}
		return m, nil
	}
	return m, nil
}

func (m *AppModel) deleteSelectedHistoryRun() {
	if m == nil || m.HistoryScreen == nil || m.HistoryStore == nil {
		return
	}
	run, ok := m.HistoryScreen.SelectedRun()
	if !ok {
		return
	}
	if err := m.HistoryStore.Delete(run.ID); err != nil {
		m.HistoryScreen.SetNotice("Delete failed: " + err.Error())
		return
	}
	m.reloadHistory()
	m.HistoryScreen.SetNotice(fmt.Sprintf("Deleted history run #%d (%s)", run.ID, run.RequestName))
}
