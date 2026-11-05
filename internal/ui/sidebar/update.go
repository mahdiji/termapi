package sidebar

import (
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func (m *Model) activateItem(item flatItem) {
	switch item.Type {
	case itemTypeAccordion:
		switch item.ZoneID {
		case "accordion_collections":
			m.CollectionsOpen = !m.CollectionsOpen
		case "accordion_workspaces":
			m.WorkspacesOpen = !m.WorkspacesOpen
		case "accordion_pinned":
			m.PinnedOpen = !m.PinnedOpen
		}
	case itemTypeFolder:
		if item.NodePtr != nil {
			item.NodePtr.Open = !item.NodePtr.Open
		}
	}
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		m.IsFocused = true
		if m.State != nil {
			m.State.TabActive = 1
		}
		if m.CreateMode != createNone {
			return m, m.updateCreatePrompt(msg)
		}
		items := m.getFlatItems()
		maxX := maxScrollX(items, m.Widht-2)

		switch msg.String() {
		case "left", "h":
			if m.ScrollX > 0 {
				m.ScrollX--
			}
		case "right", "l":
			if m.ScrollX < maxX {
				m.ScrollX++
			}
		case "shift+left", "H":
			m.ScrollX -= 5
			if m.ScrollX < 0 {
				m.ScrollX = 0
			}
		case "shift+right", "L":
			m.ScrollX += 5
			if m.ScrollX > maxX {
				m.ScrollX = maxX
			}
		case "up", "k":
			if m.CursorIdx > 0 {
				m.CursorIdx--
				m.SelectedZoneID = items[m.CursorIdx].ZoneID
			}
		case "down", "j":
			if m.CursorIdx < len(items)-1 {
				m.CursorIdx++
				m.SelectedZoneID = items[m.CursorIdx].ZoneID
			}
		case "enter", "space", " ":
			for _, item := range items {
				if item.ZoneID == m.SelectedZoneID {
					m.activateItem(item)
					break
				}
			}
		case "esc", "backspace":
			m.selectZone("accordion_collections")
			m.ScrollX = 0
			m.ScrollY = 0
		case "ctrl+l":
			return m, nil // handled by the application-level batch runner
		case "ctrl+a":
			return m, m.beginCreate(createFolder)
		case "ctrl+backspace", "ctrl+delete":
			m.deleteSelected()
		case "p":
			m.togglePinSelected()
		}
		return m, nil

	case tea.MouseWheelMsg:
		m.IsFocused = true
		items := m.getFlatItems()
		maxX := maxScrollX(items, m.Widht-2)
		maxY := len(items) - listHeight(m.Height)
		if maxY < 0 {
			maxY = 0
		}
		shift := msg.Mod&tea.ModShift != 0

		switch msg.Button {
		case tea.MouseWheelUp:
			if shift {
				if m.ScrollX > 0 {
					m.ScrollX--
				}
			} else if m.ScrollY > 0 {
				m.ScrollY--
			}
		case tea.MouseWheelDown:
			if shift {
				if m.ScrollX < maxX {
					m.ScrollX++
				}
			} else if m.ScrollY < maxY {
				m.ScrollY++
			}
		case tea.MouseWheelLeft:
			if m.ScrollX > 0 {
				m.ScrollX--
			}
		case tea.MouseWheelRight:
			if m.ScrollX < maxX {
				m.ScrollX++
			}
		}
		return m, nil

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		m.IsFocused = true
		if m.State != nil {
			m.State.TabActive = 1
		}
		for index, item := range m.getFlatItems() {
			if !zone.Get(item.ZoneID).InBounds(msg) {
				continue
			}
			m.SelectedZoneID = item.ZoneID
			m.CursorIdx = index
			m.activateItem(item)
			break
		}
		return m, nil
	}
	return m, nil
}
