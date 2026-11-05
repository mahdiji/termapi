package ui

import (
	"api/internal/ui/nav"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func (m AppModel) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}
	if m.ReqState != nil {
		m.ReqState.CommitActiveEdit()
	}
	if m.HelpMode {
		m.HelpMode = false
		return m, nil
	}
	if zone.Get("help_button").InBounds(msg) {
		m.openHelp()
		return m, nil
	}
	navHeight := lipgloss.Height(nav.NavbarContent())
	if m.Sidebar != nil && msg.Y >= navHeight && msg.X < m.Sidebar.Widht {
		m.Sidebar.IsFocused = true
		if m.State != nil {
			m.State.TabActive = 1
		}
		if m.ReqState != nil {
			m.ReqState.Blur()
		}
		if m.Response != nil {
			m.Response.Blur()
		}
		previousSelection := m.Sidebar.SelectedZoneID
		_, sidebarCmd := m.Sidebar.Update(msg)
		if m.Sidebar.SelectedZoneID == "accordion_history" && (m.Sidebar.SelectedZoneID != previousSelection || zone.Get("accordion_history").InBounds(msg)) {
			m.openHistory()
		} else if m.Sidebar.SelectedZoneID != previousSelection || zone.Get(m.Sidebar.SelectedZoneID).InBounds(msg) {
			m.openSelectedRequest()
		}
		return m, sidebarCmd
	}

	if m.Sidebar != nil {
		m.Sidebar.IsFocused = false
	}
	if m.Response != nil && m.Response.HandlesClick(msg) {
		m.Response.Focus()
		if m.State != nil {
			m.State.TabActive = 0
		}
		if m.ReqState != nil {
			m.ReqState.Blur()
		}
		m.Response.HandleMouse(msg)
		return m, nil
	}
	if m.Response != nil {
		m.Response.Blur()
	}
	if m.State != nil {
		m.State.TabActive = 0
	}
	if m.ReqState != nil {
		m.ReqState.SetFocused(true)
		_, reqCmd := m.ReqState.Update(msg)
		return m, reqCmd
	}
	return m, nil
}
