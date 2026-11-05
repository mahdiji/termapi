package scopescreen

import (
	"api/internal/config"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbletea/v2"
)

type Model struct {
	Width   int
	Height  int
	Origins []string
	Cursor  int
	Input   textinput.Model
	Adding  bool
	Err     string
	Message string
}

func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
}

func (m *Model) Load() {
	scopeStr, err := config.LoadGlobalScope("")
	if err != nil {
		m.Err = err.Error()
		return
	}
	m.Origins = nil
	for _, line := range strings.Split(scopeStr, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			m.Origins = append(m.Origins, line)
		}
	}
	if m.Cursor >= len(m.Origins) {
		m.Cursor = max(len(m.Origins)-1, 0)
	}
}

func (m *Model) Save() error {
	scopeStr := strings.Join(m.Origins, "\n")
	if err := config.SaveGlobalScope("", scopeStr); err != nil {
		m.Err = err.Error()
		return err
	}
	m.Message = "Global scope saved."
	return nil
}

func (m *Model) StartAdd() tea.Cmd {
	m.Adding = true
	m.Input = textinput.New()
	m.Input.Prompt = "New origin: "
	m.Input.Placeholder = "e.g. http://localhost:8080"
	m.Input.Focus()
	return textinput.Blink
}

func (m *Model) StopAdd() {
	m.Adding = false
	m.Input.Blur()
}

func (m *Model) Update(msg tea.KeyMsg) (bool, string) {
	if m.Adding {
		if msg.String() == "esc" {
			m.StopAdd()
			return true, strings.Join(m.Origins, "\n")
		}
		if msg.String() == "enter" {
			val := strings.TrimSpace(m.Input.Value())
			if val != "" {
				m.Origins = append(m.Origins, val)
				m.Save()
				m.Message = "Origin added: " + val
			}
			m.StopAdd()
			return true, strings.Join(m.Origins, "\n")
		}
		var cmd tea.Cmd
		m.Input, cmd = m.Input.Update(msg)
		_ = cmd
		return true, strings.Join(m.Origins, "\n")
	}

	switch msg.String() {
	case "esc":
		return false, strings.Join(m.Origins, "\n")
	case "up", "k":
		if len(m.Origins) > 0 {
			m.Cursor = max(m.Cursor-1, 0)
		}
		return true, strings.Join(m.Origins, "\n")
	case "down", "j":
		if len(m.Origins) > 0 {
			m.Cursor = min(m.Cursor+1, len(m.Origins)-1)
		}
		return true, strings.Join(m.Origins, "\n")
	case "a", "n":
		m.StartAdd()
		return true, strings.Join(m.Origins, "\n")
	case "d", "x", "backspace":
		if len(m.Origins) > 0 && m.Cursor < len(m.Origins) {
			removed := m.Origins[m.Cursor]
			m.Origins = append(m.Origins[:m.Cursor], m.Origins[m.Cursor+1:]...)
			if m.Cursor >= len(m.Origins) {
				m.Cursor = max(len(m.Origins)-1, 0)
			}
			m.Save()
			m.Message = "Origin removed: " + removed
		}
		return true, strings.Join(m.Origins, "\n")
	}
	return true, strings.Join(m.Origins, "\n")
}
