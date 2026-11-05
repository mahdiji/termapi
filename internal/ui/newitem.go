package ui

import (
	"api/internal/ui/newscreen"

	tea "charm.land/bubbletea/v2"
)

// openNewItem opens the global "new request / new folder" composer.
func (m *AppModel) openNewItem(kind newscreen.Kind) {
	if m == nil || m.Sidebar == nil || m.Sidebar.CollectionStore == nil || m.NewItemScreen == nil {
		return
	}
	preferred := m.Sidebar.SelectedFolderPath()
	if preferred == "" && m.ReqState != nil {
		preferred = m.ReqState.CollectionPath
	}
	m.NewItemScreen.SetStore(m.Sidebar.CollectionStore)
	m.NewItemScreen.Open(kind, preferred)
	m.NewItemScreen.SetSize(m.ScreenWidth, m.ScreenHeight)
	m.NewItemMode = true
}

func (m AppModel) updateNewItemMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.NewItemScreen == nil {
		m.NewItemMode = false
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ScreenWidth, m.ScreenHeight = msg.Width, msg.Height
		if m.State != nil {
			m.State.Widht, m.State.Height = msg.Width, msg.Height
		}
		m.NewItemScreen.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch m.NewItemScreen.Update(msg) {
		case newscreen.ActionClose:
			m.NewItemMode = false
			return m, nil
		case newscreen.ActionSubmit:
			entry, err := m.NewItemScreen.Submit()
			if err != nil {
				m.NewItemScreen.Err = err.Error()
				return m, nil
			}
			m.NewItemMode = false
			m.Sidebar.ReloadCollections("")
			m.Sidebar.SelectPath(entry.RelPath, entry.IsFolder)
			if entry.IsFolder {
				m.Sidebar.IsFocused = true
				if m.State != nil {
					m.State.TabActive = 1
				}
				if m.ReqState != nil {
					m.ReqState.Blur()
				}
			} else {
				m.openSelectedRequest()
			}
			return m, nil
		}
	}
	return m, nil
}

func (m AppModel) viewNewItem() tea.View {
	p := tea.NewView(m.NewItemScreen.View())
	p.AltScreen = true
	return p
}
