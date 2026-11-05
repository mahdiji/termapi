package envscreen

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m *Model) Update(msg tea.KeyMsg) (active bool, newActiveEnv string, vars map[string]string) {
	ret := func() (bool, string, map[string]string) {
		return true, m.ActiveEnv, m.activeVars()
	}
	close := func() (bool, string, map[string]string) {
		return false, m.ActiveEnv, m.activeVars()
	}

	// ── adding a new env ─────────────────────────────────────────────────
	if m.Adding {
		switch msg.String() {
		case "esc":
			m.Adding = false
			m.InputName.Blur()
			return ret()
		case "enter":
			name := strings.TrimSpace(m.InputName.Value())
			if name != "" {
				if _, err := m.Store.Create(name); err != nil {
					m.Err = err.Error()
				} else {
					m.Err = ""
					m.Message = "Created: " + name
					m.Load()
					// Move cursor to the new env
					for i, n := range m.Envs {
						if n == name {
							m.EnvCursor = i
							break
						}
					}
					m.refreshRight()
				}
			}
			m.Adding = false
			m.InputName.Blur()
			return ret()
		default:
			m.InputName, _ = m.InputName.Update(msg)
			return ret()
		}
	}

	// ── adding / editing a variable ──────────────────────────────────────
	if m.AddingVar || m.EditingVar {
		switch msg.String() {
		case "esc":
			m.AddingVar = false
			m.EditingVar = false
			m.InputKey.Blur()
			m.InputVal.Blur()
			return ret()
		case "tab":
			m.InputFocus = 1 - m.InputFocus
			if m.InputFocus == 0 {
				m.InputKey.Focus()
				m.InputVal.Blur()
			} else {
				m.InputVal.Focus()
				m.InputKey.Blur()
			}
			return ret()
		case "enter":
			key := strings.TrimSpace(m.InputKey.Value())
			val := m.InputVal.Value()
			if key != "" && m.Current != nil {
				if m.Current.Variables == nil {
					m.Current.Variables = make(map[string]string)
				}
				m.Current.Variables[key] = val
				if err := m.Store.Save(m.Current); err != nil {
					m.Err = err.Error()
				} else {
					m.Err = ""
					m.Message = "Saved variable: " + key
					m.rebuildVarKeys()
					// Position cursor on edited/added key
					for i, k := range m.VarKeys {
						if k == key {
							m.VarCursor = i
							break
						}
					}
					// If active env updated, refresh vars
					if m.Current.Name == m.ActiveEnv {
						// vars returned via activeVars() → already up to date since we saved
					}
				}
			}
			m.AddingVar = false
			m.EditingVar = false
			m.InputKey.Blur()
			m.InputVal.Blur()
			return ret()
		default:
			if m.InputFocus == 0 {
				m.InputKey, _ = m.InputKey.Update(msg)
			} else {
				m.InputVal, _ = m.InputVal.Update(msg)
			}
			return ret()
		}
	}

	// ── left panel (env list) ────────────────────────────────────────────
	if m.FocusPane == 0 {
		switch msg.String() {
		case "esc":
			return close()
		case "ctrl+e":
			return close()
		case "tab":
			m.FocusPane = 1
			return ret()
		case "up", "k":
			if m.EnvCursor > 0 {
				m.EnvCursor--
				m.refreshRight()
			}
			return ret()
		case "down", "j":
			if m.EnvCursor < len(m.Envs)-1 {
				m.EnvCursor++
				m.refreshRight()
			}
			return ret()
		case "enter":
			// Set as active env
			if len(m.Envs) > 0 {
				m.ActiveEnv = m.Envs[m.EnvCursor]
				m.Message = "Active env: " + m.ActiveEnv
				m.refreshRight()
			}
			return ret()
		case "a":
			m.Adding = true
			m.InputName = textinput.New()
			m.InputName.Prompt = ""
			m.InputName.Placeholder = "environment name"
			m.InputName.Focus()
			return ret()
		case "d":
			if len(m.Envs) > 0 {
				name := m.Envs[m.EnvCursor]
				if name == m.ActiveEnv {
					m.Err = "Cannot delete the active environment"
				} else {
					if err := m.Store.Delete(name); err != nil {
						m.Err = err.Error()
					} else {
						m.Err = ""
						m.Message = "Deleted: " + name
						m.Load()
					}
				}
			}
			return ret()
		}
		return ret()
	}

	// ── right panel (var list) ───────────────────────────────────────────
	switch msg.String() {
	case "esc":
		return close()
	case "ctrl+e":
		return close()
	case "tab":
		m.FocusPane = 0
		return ret()
	case "up", "k":
		if m.VarCursor > 0 {
			m.VarCursor--
		}
		return ret()
	case "down", "j":
		if m.VarCursor < len(m.VarKeys)-1 {
			m.VarCursor++
		}
		return ret()
	case "a":
		if m.Current != nil {
			m.startAddVar("")
		}
		return ret()
	case "e", "enter":
		if m.Current != nil && len(m.VarKeys) > 0 {
			key := m.VarKeys[m.VarCursor]
			m.startEditVar(key, m.Current.Variables[key])
		}
		return ret()
	case "d":
		if m.Current != nil && len(m.VarKeys) > 0 {
			key := m.VarKeys[m.VarCursor]
			delete(m.Current.Variables, key)
			if err := m.Store.Save(m.Current); err != nil {
				m.Err = err.Error()
			} else {
				m.Err = ""
				m.Message = "Deleted var: " + key
				m.rebuildVarKeys()
				if m.VarCursor >= len(m.VarKeys) {
					m.VarCursor = max(len(m.VarKeys)-1, 0)
				}
			}
		}
		return ret()
	}
	return ret()
}

func (m *Model) startAddVar(key string) {
	m.AddingVar = true
	m.EditingVar = false
	m.InputFocus = 0
	m.InputKey = textinput.New()
	m.InputKey.Prompt = ""
	m.InputKey.Placeholder = "KEY"
	m.InputKey.SetValue(key)
	m.InputVal = textinput.New()
	m.InputVal.Prompt = ""
	m.InputVal.Placeholder = "value"
	m.InputKey.Focus()
}

func (m *Model) startEditVar(key, val string) {
	m.EditingVar = true
	m.AddingVar = false
	m.InputFocus = 1
	m.InputKey = textinput.New()
	m.InputKey.Prompt = ""
	m.InputKey.Placeholder = "KEY"
	m.InputKey.SetValue(key)
	m.InputVal = textinput.New()
	m.InputVal.Prompt = ""
	m.InputVal.Placeholder = "value"
	m.InputVal.SetValue(val)
	m.InputVal.Focus()
}

// ── View ──────────────────────────────────────────────────────────────────────
