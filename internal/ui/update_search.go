package ui

import (
	"api/internal/ui/sidebar"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// openFileSearch membuka file finder dan mereset state pencarian.
func (m *AppModel) openFileSearch() tea.Cmd {
	m.SearchMode = true
	m.SearchHistory = false
	m.SearchSelectedIndex = 0
	m.SearchInput.SetValue("")
	m.SearchInput.Focus()
	if m.Sidebar != nil {
		m.Sidebar.SetFilter("")
	}
	return textinput.Blink
}

// updateSearchMode menangani pencarian file collection (dan history search).
func (m AppModel) updateSearchMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.ScreenWidth, m.ScreenHeight = size.Width, size.Height
		if m.State != nil {
			m.State.Widht, m.State.Height = size.Width, size.Height
		}
		return m, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	suggestions := m.searchSuggestions()
	if m.SearchSelectedIndex >= len(suggestions) {
		m.SearchSelectedIndex = max(len(suggestions)-1, 0)
	}
	if m.SearchSelectedIndex < 0 {
		m.SearchSelectedIndex = 0
	}

	switch key.String() {
	case "esc":
		m.SearchMode = false
		m.recalculateLayout()
		m.SearchInput.Blur()
		if m.Sidebar != nil {
			m.Sidebar.SetFilter("")
		}
		return m, nil
	case "up", "k":
		if m.SearchSelectedIndex > 0 {
			m.SearchSelectedIndex--
		}
		return m, nil
	case "down", "j":
		if m.SearchSelectedIndex < len(suggestions)-1 {
			m.SearchSelectedIndex++
		}
		return m, nil
	case "enter":
		m.SearchMode = false
		m.recalculateLayout()
		m.SearchInput.Blur()
		if !m.SearchHistory && len(suggestions) > 0 {
			sel := suggestions[m.SearchSelectedIndex]
			if m.Sidebar != nil {
				m.Sidebar.SetFilter("")
				m.Sidebar.IsFocused = true
				if m.State != nil {
					m.State.TabActive = 1
				}
				if m.ReqState != nil {
					m.ReqState.Blur()
				}
				m.Sidebar.RevealPath(sel.Path)
				m.Sidebar.SelectPath(sel.Path, sel.IsFolder)
				if !sel.IsFolder {
					m.openSelectedRequest()
				}
			}
		} else if m.SearchHistory {
			if m.HistoryScreen != nil {
				m.HistoryScreen.SetQuery(m.SearchInput.Value())
			}
		} else if m.Sidebar != nil {
			m.Sidebar.SetFilter("")
		}
		return m, nil
	}

	var searchCmd tea.Cmd
	m.SearchInput, searchCmd = m.SearchInput.Update(key)
	m.SearchSelectedIndex = 0
	if m.SearchHistory {
		if m.HistoryScreen != nil {
			m.HistoryScreen.SetQuery(m.SearchInput.Value())
		}
	}
	return m, searchCmd
}

// searchSuggestions mengembalikan seluruh request file/folder yang cocok
// dengan query, termasuk yang berada di folder tertutup.
func (m AppModel) searchSuggestions() []sidebar.SearchSuggestion {
	if m.Sidebar == nil || m.SearchHistory {
		return nil
	}
	return m.Sidebar.GetSearchSuggestions(m.SearchInput.Value())
}
