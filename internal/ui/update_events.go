package ui

import (
	"net/url"
	"time"

	"api/internal/historydb"
	"api/internal/ui/nav"
	requestinput "api/internal/ui/requestInput"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// updateEvents menangani event utama saat tidak ada mode overlay aktif:
// response HTTP, resize, mouse, dan shortcut keyboard global.
func (m AppModel) updateEvents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case batchResponseMsg:
		if msg.index >= 0 && msg.index < len(m.BatchResults) {
			m.BatchResults[msg.index] = msg.response
			if msg.index == m.BatchSelected && m.Response != nil {
				m.Response.SetResponse(msg.response)
			}
		}
		if m.HistoryStore != nil {
			r := msg.response
			target := ""
			if parsed, err := url.Parse(r.URL); err == nil {
				target = parsed.Hostname()
			}
			body := r.RawBody
			if body == "" && r.Err != "" {
				body = r.Err
			}
			if err := m.HistoryStore.Save(historydb.Run{RequestName: r.RequestName, TargetDomain: target, URL: r.URL, Method: r.Method, StatusCode: r.Code, ResponseTime: r.Duration.Milliseconds(), ResponseBody: body, ExecutedAt: time.Now()}, r.CollectionName, r.CollectionPath); err != nil && m.HistoryScreen != nil {
				m.HistoryScreen.Err = err.Error()
			}
		}
		if m.BatchChain {
			m.captureChainedVariables(msg.response)
			return m, m.nextChainedRequest()
		}
		return m, nil

	case spinner.TickMsg:
		if m.Response != nil && m.Response.Loading {
			return m, m.Response.UpdateSpinner(msg)
		}
		return m, nil

	case requestinput.SaveRequestedMsg:
		m.openSaveDialog()
		return m, nil

	case requestinput.RequestStartedMsg:
		if m.Response != nil {
			m.Response.SetLoading()
			return m, m.Response.SpinnerTick()
		}
		return m, nil

	case requestinput.ResponseMsg:
		m.captureChainedVariables(msg)
		if m.Response != nil {
			m.Response.SetResponse(msg)
		}
		if m.HistoryStore != nil {
			target := ""
			if parsed, err := url.Parse(msg.URL); err == nil {
				target = parsed.Hostname()
			}
			body := msg.RawBody
			if body == "" && msg.Err != "" {
				body = msg.Err
			}
			err := m.HistoryStore.Save(historydb.Run{
				RequestName: msg.RequestName, TargetDomain: target, URL: msg.URL,
				Method: msg.Method, StatusCode: msg.Code,
				ResponseTime: msg.Duration.Milliseconds(), ResponseBody: body, ExecutedAt: time.Now(),
			}, msg.CollectionName, msg.CollectionPath)
			if err != nil && m.HistoryScreen != nil {
				m.HistoryScreen.Err = err.Error()
			}
			if m.HistoryMode {
				m.reloadHistory()
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)

	case tea.MouseMotionMsg:
		if m.Response != nil && m.Response.IsDraggingSelection() {
			m.Response.HandleMotion(msg)
			return m, nil
		}
		return m, nil

	case tea.MouseReleaseMsg:
		if m.Response != nil {
			m.Response.HandleRelease()
		}
		if m.ReqState != nil {
			_, reqCmd := m.ReqState.Update(msg)
			return m, reqCmd
		}
		return m, nil

	case requestinput.NoticeExpiredMsg:
		if m.Response != nil {
			m.Response.ClearNotice(msg.ID)
		}
		return m, nil

	case tea.MouseWheelMsg:
		if m.HelpMode {
			if m.HelpScreen != nil {
				if msg.Button == tea.MouseWheelUp {
					m.HelpScreen.ScrollBy(-3)
				} else if msg.Button == tea.MouseWheelDown {
					m.HelpScreen.ScrollBy(3)
				}
			}
			return m, nil
		}
		navHeight := lipgloss.Height(nav.NavbarContent())
		if m.Sidebar != nil && msg.Y >= navHeight && msg.X < m.Sidebar.Widht {
			_, cmd := m.Sidebar.Update(msg)
			return m, cmd
		}
		if m.Response != nil && m.Response.HandlesWheel(msg) {
			m.Response.HandleWheel(msg)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleGlobalKeys(msg)
	}

	if m.Sidebar != nil {
		_, sidebarCmd := m.Sidebar.Update(msg)
		cmd = tea.Batch(cmd, sidebarCmd)
	}

	if m.ReqState != nil {
		_, reqCmd := m.ReqState.Update(msg)
		cmd = tea.Batch(cmd, reqCmd)
	}

	return m, cmd
}
