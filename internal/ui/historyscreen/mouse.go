package historyscreen

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func (m *Model) HandleMouse(msg tea.MouseClickMsg) {
	for i := range m.Runs {
		if zone.Get(fmt.Sprintf("history_run_%d", i)).InBounds(msg) {
			m.Cursor = i
			m.updateSelectedResponse()
			return
		}
	}
	if m.Response != nil {
		m.Response.HandleMouse(msg)
	}
}

func (m *Model) HandleWheel(msg tea.MouseWheelMsg) {
	if m.Response != nil && msg.Y >= m.Height/2 {
		m.Response.HandleWheel(msg)
		return
	}
	if len(m.Runs) == 0 {
		return
	}
	switch msg.Button {
	case tea.MouseWheelUp:
		m.Cursor = max(m.Cursor-3, 0)
		m.updateSelectedResponse()
	case tea.MouseWheelDown:
		m.Cursor = min(m.Cursor+3, len(m.Runs)-1)
		m.updateSelectedResponse()
	}
}

