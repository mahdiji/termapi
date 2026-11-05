package ui

import (
	"api/internal/ui/newscreen"
	requestinput "api/internal/ui/requestInput"

	tea "charm.land/bubbletea/v2"
)

func (m AppModel) handleGlobalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.HelpMode {

		if isAltQuestion(msg) {
			m.HelpMode = false
			return m, nil
		}
		switch msg.String() {
		case "esc", "enter", "q", "f1":
			m.HelpMode = false
			return m, nil
		case "up", "k":
			if m.HelpScreen != nil {
				m.HelpScreen.ScrollBy(-1)
			}
		case "down", "j":
			if m.HelpScreen != nil {
				m.HelpScreen.ScrollBy(1)
			}
		case "pgup", "ctrl+u":
			if m.HelpScreen != nil {
				m.HelpScreen.ScrollPage(-1)
			}
		case "pgdown", "ctrl+d":
			if m.HelpScreen != nil {
				m.HelpScreen.ScrollPage(1)
			}
		case "home", "g":
			if m.HelpScreen != nil {
				m.HelpScreen.ScrollTop()
			}
		case "end", "G":
			if m.HelpScreen != nil {
				m.HelpScreen.ScrollBottom()
			}
		}
		return m, nil
	}

	// Ctrl+N is global across the main screen, including while the response
	// filter is focused.
	if msg.String() == "ctrl+n" {
		m.openNewItem(newscreen.KindRequest)
		return m, nil
	}

	// When the response gjson filter is focused, it owns all keys except Esc,
	// which returns focus to the request editor.
	if m.Response != nil && m.Response.FilterFocued {
		switch msg.String() {
		case "esc":
			m.Response.SetFilter(false)
			return m, nil
		case "enter":
			m.Response.SetFilter(false)
			return m, nil
		}
		return m, m.Response.UpdateFilter(msg)
	}

	// When the in-response find field is focused, it owns all keys except Esc
	// (close) and Enter (jump to the next match and return to the panel).
	if m.Response != nil && m.Response.FindFocused {
		switch msg.String() {
		case "esc":
			m.Response.CloseFind()
			return m, nil
		case "enter":
			m.Response.NextMatch(1)
			m.Response.Find.Blur()
			m.Response.FindFocused = false
			return m, nil
		}
		return m, m.Response.UpdateFind(msg)
	}

	// Response panel focus: navigation, find, selection and copy.
	if m.Response != nil && m.Response.Focused {
		switch msg.String() {
		case "esc":
			if m.Response.Loading && m.ReqState != nil {
				m.ReqState.CancelRequest()
				return m, nil
			}
			if m.Response.FindActive() {
				m.Response.CloseFind()
				return m, nil
			}
			if m.Response.SelActive {
				m.Response.ClearSelection()
				return m, nil
			}
			m.Response.Blur()
			return m, nil
		case "tab":
			m.Response.NextTab(1)
			return m, nil
		case "shift+tab":
			m.Response.NextTab(-1)
			return m, nil
		case "ctrl+f":
			return m, m.Response.OpenFind()
		case "n":
			m.Response.NextMatch(1)
			return m, nil
		case "N":
			m.Response.NextMatch(-1)
			return m, nil
		case "y", "enter":
			text, lines, ok := m.Response.CopySelection()
			if !ok {
				text, lines = m.Response.CopyAll()
			}
			if lines == 0 {
				return m, nil
			}
			return m, tea.Batch(tea.SetClipboard(text), m.Response.SetNotice(requestinput.CopiedNotice(lines)))
		case "up", "k":
			m.Response.ScrollUp(1)
			return m, nil
		case "down", "j":
			m.Response.ScrollDown(1, 0)
			return m, nil
		}
	}

	if isAltQuestion(msg) {
		m.openHelp()
		return m, nil
	}

	switch msg.String() {
	case "esc":
		// Cancel an in-flight request while the response pane is loading.
		if m.Response != nil && m.Response.Loading && m.ReqState != nil {
			m.ReqState.CancelRequest()
			return m, nil
		}
	case "f1":
		m.openHelp()
		return m, nil
	case "ctrl+k":
		// Command palette is a planned feature; swallow to avoid leaking the
		// key to the URL input.
		return m, nil
	case "ctrl+up":
		m.adjustSplit(-2)
		return m, nil
	case "ctrl+down":
		m.adjustSplit(2)
		return m, nil
	case "/":
		// Focus the response gjson filter when a response is available and the
		// user is not typing in the URL bar or request editors.
		responseFocused := m.Response != nil && m.Response.Focused
		typing := !responseFocused && m.State != nil && m.State.TabActive == 0 && m.ReqState != nil &&
			(m.ReqState.FocusArea == 1 || m.ReqState.FocusArea == 3)
		if !typing && m.Response != nil && m.Response.HasContent() {
			m.Response.Focus()
			m.Response.SetFilter(true)
			return m, nil
		}
	case "ctrl+f":
		return m, m.openFileSearch()
	case "ctrl+y":
		if m.Response != nil {
			text, lines := m.Response.CopyAll()
			if lines > 0 {
				return m, tea.Batch(tea.SetClipboard(text), m.Response.SetNotice(requestinput.CopiedNotice(lines)))
			}
		}
		return m, nil
	case "ctrl+d":
		if m.Sidebar != nil && m.Sidebar.IsFocused {
			deletedPath := m.Sidebar.DeleteSelectedRequest()
			if deletedPath != "" && m.ReqState != nil && m.ReqState.RequestFilePath == deletedPath {
				m.ReqState.RequestFilePath = ""
			}
		} else {
			m.openDiffViewer()
		}
		return m, nil
	case "ctrl+o":
		m.openHistory()
		return m, nil
	case "ctrl+s":
		m.openSaveDialog()
		return m, nil
	case "ctrl+g":
		m.openScopeManager()
		return m, nil
	case "ctrl+e":
		m.openEnvManager()
		return m, nil
	case "ctrl+q":
		return m, tea.Quit
	case "pgup", "ctrl+u":
		if m.Response != nil {
			m.Response.ScrollUp(5)
		}
		return m, nil
	case "pgdown":
		if m.Response != nil {
			m.Response.ScrollDown(5, 1000)
		}
		return m, nil
	case "tab", "	":
		if m.ReqState != nil && m.ReqState.FocusArea == 3 {
			break
		}
		if m.State != nil {
			if m.State.TabActive == 0 {
				m.State.TabActive = 1
				if m.Sidebar != nil {
					m.Sidebar.IsFocused = true
				}
				if m.ReqState != nil {
					m.ReqState.Blur()
				}
			} else {
				m.State.TabActive = 0
				if m.Sidebar != nil {
					m.Sidebar.IsFocused = false
				}
				if m.ReqState != nil {
					m.ReqState.SetFocused(true)
				}
			}
		}
		return m, nil
	}

	if m.State != nil && m.State.TabActive == 1 && m.Sidebar != nil {
		if msg.String() == "ctrl+l" {
			m.openBatchChooser()
			return m, nil
		}
		_, sidebarCmd := m.Sidebar.Update(msg)
		if msg.String() == "enter" {
			m.openSelectedRequest()
		}
		return m, sidebarCmd
	}
	if m.ReqState != nil {
		_, reqCmd := m.ReqState.Update(msg)
		return m, reqCmd
	}
	return m, nil
}

func isAltQuestion(msg tea.KeyMsg) bool {
	key := msg.Key()
	if !key.Mod.Contains(tea.ModAlt) {
		return false
	}
	if key.Text == "?" || key.Code == '?' {
		return true
	}
	return key.Code == '/' && key.Mod.Contains(tea.ModShift)
}
