package ui

import tea "charm.land/bubbletea/v2"

const (
	minWindowWidth  = 105
	minWindowHeight = 32
)

func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update adalah pusat routing event Bubble Tea.
// Setiap mode/fitur punya handler sendiri agar mudah dijelaskan per file.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.NewItemMode {
		return m.updateNewItemMode(msg)
	}
	if m.SaveDialog {
		return m.updateSaveDialog(msg)
	}
	if m.BatchChoosing {
		return m.updateBatchChoosing(msg)
	}
	if m.Sidebar != nil && m.Sidebar.CreatePromptActive() {
		if key, ok := msg.(tea.KeyMsg); ok {
			_, createCmd := m.Sidebar.Update(key)
			return m, createCmd
		}
	}
	if m.BatchMode {
		return m.updateBatchMode(msg)
	}
	if m.HistoryMode {
		return m.updateHistoryMode(msg)
	}
	if m.ScopeMode {
		return m.updateScopeMode(msg)
	}
	if m.EnvMode {
		return m.updateEnvMode(msg)
	}
	if m.DiffMode {
		return m.updateDiffMode(msg)
	}
	if m.SearchMode {
		return m.updateSearchMode(msg)
	}
	if m.WireMode {
		return m.updateWireMode(msg)
	}
	return m.updateEvents(msg)
}
