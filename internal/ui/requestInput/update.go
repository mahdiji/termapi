package requestinput

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	// MURNI BUBBLE TEA V2: Menangkap hanya event klik kiri aktif
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}

		if m.State != nil {
			m.State.TabActive = 0
		}

		// JIKA KLIK SEDANG DIKUNCI, ABAIKAN EVENT DUPLIKAT YANG MASUK
		if m.isClickLocked {
			return m, nil
		}
		m.isClickLocked = true // Kunci klik saat ini
		// Simpan sel yang sedang diedit agar perpindahan fokus/tab tidak
		// meninggalkan state edit yang bocor ke tabel lain.
		m.commitActiveEdit()
		for i := 0; i < requestTabCount; i++ {
			if zone.Get(fmt.Sprintf("request_tab_%d", i)).InBounds(msg) {
				if m.FocusArea == 1 {
					m.syncParamsFromURL()
				}
				m.Tab = i
				m.FocusArea = 3
				return m, m.focusActiveTab()
			}
		}

		// Body dan Auth memakai pill tipe yang bisa diklik.
		if m.Tab == 2 {
			for _, t := range bodyTypes {
				if zone.Get("body_type_" + t).InBounds(msg) {
					m.Body.SetType(t)
					m.FocusArea = 3
					return m, m.focusActiveTab()
				}
			}
		}
		if m.Tab == 3 {
			for _, t := range authTypes {
				if zone.Get("auth_type_" + t).InBounds(msg) {
					m.Auth.SetType(t)
					m.FocusArea = 3
					return m, m.focusActiveTab()
				}
			}
		}

		if zone.Get("request_editor").InBounds(msg) {
			if m.FocusArea == 1 {
				m.syncParamsFromURL()
			}
			m.FocusArea = 3
			m.Input.Blur()
			if kv := m.activeKV(); kv != nil && kv.HandleClick(msg) {
				return m, nil
			}
			return m, m.focusActiveTab()
		}

		if zone.Get("method_left_arrow").InBounds(msg) {
			if m.FocusArea == 1 {
				m.syncParamsFromURL()
			}
			idx := (m.getCurrentMethodIndex() - 1 + len(m.Methods)) % len(m.Methods)
			m.Method = m.Methods[idx]
			m.FocusArea = 0
			m.updateFocusState()
			m.SetWidth(m.lastWidth)
			return m, nil
		}

		if zone.Get("method_right_arrow").InBounds(msg) {
			if m.FocusArea == 1 {
				m.syncParamsFromURL()
			}
			idx := (m.getCurrentMethodIndex() + 1) % len(m.Methods)
			m.Method = m.Methods[idx]
			m.FocusArea = 0
			m.updateFocusState()
			m.SetWidth(m.lastWidth)
			return m, nil
		}

		if zone.Get("method_name").InBounds(msg) || zone.Get("method_selector").InBounds(msg) {
			if m.FocusArea == 1 {
				m.syncParamsFromURL()
			}
			m.FocusArea = 0
			m.updateFocusState()
			return m, nil
		}

		if zone.Get("send_button").InBounds(msg) {
			if m.FocusArea == 1 {
				m.syncParamsFromURL()
			}
			m.FocusArea = 2
			m.updateFocusState()
			return m, m.sendRequest()
		}

		if zone.Get("url_input").InBounds(msg) {
			m.FocusArea = 1
			m.updateFocusState()
			m.Input, cmd = m.Input.Update(msg)
			return m, cmd
		}

	// Tangkap event rilis mouse bawaan v2 untuk membuka kembali kunci klik
	case tea.MouseReleaseMsg:
		m.isClickLocked = false
		return m, nil

	case tea.KeyMsg:
		m.isClickLocked = false // Reset status kunci saat keyboard ditekan

		key := msg.String()
		if key == "ctrl+t" {
			if m.FocusArea == 3 {
				m.commitActiveEdit()
			}
			if m.FocusArea == 1 {
				m.syncParamsFromURL()
			}
			m.Tab = (m.Tab + 1) % requestTabCount
			m.FocusArea = 3
			return m, m.focusActiveTab()
		}
		if key == "ctrl+s" {
			return m, func() tea.Msg { return SaveRequestedMsg{} }
		}

		if m.FocusArea == 3 {
			if key == "ctrl+r" {
				if kv := m.activeKV(); kv != nil && kv.RawAllowed {
					m.commitActiveEdit()
					_ = kv.ToggleRaw()
					if kv.IsRaw() {
						return m, kv.Focus()
					}
				}
				return m, nil
			}
			return m, m.updateActiveTab(msg)
		}

		switch key {
		case "tab", "\t":
			if m.FocusArea == 1 {
				m.syncParamsFromURL()
			}
			m.FocusArea = (m.FocusArea + 1) % 3
			m.updateFocusState()
			return m, nil
		case "shift+tab":
			if m.FocusArea == 1 {
				m.syncParamsFromURL()
			}
			m.FocusArea = (m.FocusArea - 1 + 3) % 3
			m.updateFocusState()
			return m, nil
		}

		if m.FocusArea == 0 {
			idx := m.getCurrentMethodIndex()
			switch key {
			case "right", "down", "space", " ":
				idx = (idx + 1) % len(m.Methods)
				m.Method = m.Methods[idx]
				m.SetWidth(m.lastWidth)
				return m, nil
			case "left", "up":
				idx = (idx - 1 + len(m.Methods)) % len(m.Methods)
				m.Method = m.Methods[idx]
				m.SetWidth(m.lastWidth)
				return m, nil
			case "enter":
				m.FocusArea = 1
				m.updateFocusState()
				return m, nil
			}
		} else if m.FocusArea == 1 {
			if key == "enter" {
				return m, m.sendRequest()
			}
			m.Input, cmd = m.Input.Update(msg)
			return m, cmd
		} else if m.FocusArea == 2 {
			if key == "enter" {
				return m, m.sendRequest()
			}
		}
	}

	if m.FocusArea == 1 {
		m.Input, cmd = m.Input.Update(msg)
	} else if m.FocusArea == 3 {
		cmd = m.updateActiveTab(msg)
	}
	return m, cmd
}
