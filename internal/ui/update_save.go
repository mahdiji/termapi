package ui

import (
	"strconv"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// updateSaveDialog menangani keyboard/resize saat dialog simpan request aktif.
func (m AppModel) updateSaveDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.ScreenWidth, m.ScreenHeight = size.Width, size.Height
		if m.State != nil {
			m.State.Widht, m.State.Height = size.Width, size.Height
		}
		m.SaveName.SetWidth(max(size.Width-24, 20))
		m.SaveSearch.SetWidth(max(size.Width-24, 20))
		m.keepSaveSelectionVisible()
		return m, nil
	}
	if wheel, ok := msg.(tea.MouseWheelMsg); ok {
		end := min(m.SaveScroll+m.saveListRows(), len(m.SaveVisible))
		for index := m.SaveScroll; index < end; index++ {
			if zone.Get(saveFolderZone(index)).InBounds(wheel) {
				delta := 3
				if wheel.Button == tea.MouseWheelUp {
					delta = -delta
				}
				m.SaveScroll = min(max(m.SaveScroll+delta, 0), max(len(m.SaveVisible)-m.saveListRows(), 0))
				m.SaveSelected = min(max(m.SaveScroll, 0), max(len(m.SaveVisible)-1, 0))
				return m, nil
			}
		}
		return m, nil
	}
	if click, ok := msg.(tea.MouseClickMsg); ok && click.Button == tea.MouseLeft {
		if zone.Get("save_name").InBounds(click) {
			m.SaveFocus = 0
			m.focusSaveInput()
			return m, nil
		}
		if zone.Get("save_search").InBounds(click) {
			m.SaveFocus = 1
			m.focusSaveInput()
			return m, nil
		}
		end := min(m.SaveScroll+m.saveListRows(), len(m.SaveVisible))
		for index := m.SaveScroll; index < end; index++ {
			if zone.Get(saveFolderZone(index)).InBounds(click) {
				m.SaveSelected = index
				m.SaveFocus = 2
				m.focusSaveInput()
				return m, nil
			}
		}
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+f":
			m.SaveFocus = 1
			m.focusSaveInput()
			return m, nil
		case "esc":
			m.SaveDialog = false
			return m, nil
		case "tab":
			m.SaveFocus = (m.SaveFocus + 1) % 3
			m.focusSaveInput()
			return m, nil
		case "shift+tab":
			m.SaveFocus = (m.SaveFocus + 2) % 3
			m.focusSaveInput()
			return m, nil
		case "enter":
			if m.SaveFocus == 1 && len(m.SaveVisible) > 0 {
				m.SaveFocus = 2
				m.focusSaveInput()
				return m, nil
			}
			m.saveDialogRequest()
			return m, nil
		}
		if m.SaveFocus == 2 {
			switch key.String() {
			case "up", "k":
				m.moveSaveSelection(-1)
			case "down", "j":
				m.moveSaveSelection(1)
			case "pgup":
				m.moveSaveSelection(-m.saveListRows())
			case "pgdown":
				m.moveSaveSelection(m.saveListRows())
			case "home":
				m.SaveSelected = 0
				m.keepSaveSelectionVisible()
			case "end":
				m.SaveSelected = max(len(m.SaveVisible)-1, 0)
				m.keepSaveSelectionVisible()
			}
			return m, nil
		}
		var input textinput.Model
		if m.SaveFocus == 0 {
			input = m.SaveName
		} else {
			input = m.SaveSearch
		}
		var inputCmd tea.Cmd
		input, inputCmd = input.Update(key)
		if m.SaveFocus == 0 {
			m.SaveName = input
		} else {
			m.SaveSearch = input
			m.refreshSaveFolders("")
		}
		return m, inputCmd
	}
	return m, nil
}

func saveFolderZone(row int) string {
	return "save_folder_row_" + strconv.Itoa(row)
}
