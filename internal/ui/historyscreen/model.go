package historyscreen

import (
	"fmt"
	"strings"
	"time"

	"api/internal/historydb"
	requestinput "api/internal/ui/requestInput"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type Model struct {
	Runs     []historydb.Run
	Cursor   int
	Width    int
	Height   int
	Err      string
	Notice   string
	Search   textinput.Model
	Query    string
	allRuns  []historydb.Run
	Response *requestinput.ResponsePanel
}

// SelectedRun returns the run currently highlighted in the history table.
func (m *Model) SelectedRun() (historydb.Run, bool) {
	if m == nil || m.Cursor < 0 || m.Cursor >= len(m.Runs) {
		return historydb.Run{}, false
	}
	return m.Runs[m.Cursor], true
}

func (m *Model) SetNotice(text string) { m.Notice = text }

func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	if m.Response != nil {
		m.Response.SetSize(max(width-2, 10), max(height/2, 6))
	}
}

func (m *Model) SetRuns(runs []historydb.Run) {
	m.allRuns = runs
	m.applySearch()
	if m.Cursor >= len(runs) {
		m.Cursor = max(len(runs)-1, 0)
	}
	m.updateSelectedResponse()
}

func (m *Model) StartSearch() tea.Cmd {
	m.Search = textinput.New()
	m.Search.Prompt = "Search: "
	m.Search.Placeholder = "request, URL, method, status..."
	m.Search.SetValue(m.Query)
	m.Search.Focus()
	return textinput.Blink
}

func (m *Model) StopSearch() { m.Search.Blur() }

func (m *Model) SetQuery(query string) {
	m.Query = query
	m.applySearch()
}

func (m *Model) applySearch() {
	q := strings.ToLower(strings.TrimSpace(m.Query))
	if q == "" {
		m.Runs = append([]historydb.Run(nil), m.allRuns...)
	} else {
		m.Runs = nil
		for _, run := range m.allRuns {
			text := strings.ToLower(strings.Join([]string{run.RequestName, run.TargetDomain, run.URL, run.Method, fmt.Sprint(run.StatusCode), run.Collection}, " "))
			if strings.Contains(text, q) {
				m.Runs = append(m.Runs, run)
			}
		}
	}
	if m.Cursor >= len(m.Runs) {
		m.Cursor = max(len(m.Runs)-1, 0)
	}
	m.updateSelectedResponse()
}

func (m *Model) updateSelectedResponse() {
	if m.Response == nil {
		m.Response = requestinput.NewResponsePanel()
	}
	if len(m.Runs) > 0 && m.Cursor < len(m.Runs) {
		run := m.Runs[m.Cursor]
		m.Response.SetResponse(requestinput.ResponseMsg{
			Method:   run.Method,
			URL:      run.URL,
			Code:     run.StatusCode,
			Status:   fmt.Sprintf("%d", run.StatusCode),
			RawBody:  run.ResponseBody,
			Body:     run.ResponseBody,
			Duration: time.Duration(run.ResponseTime) * time.Millisecond,
		})
	} else {
		m.Response.SetLoading()
	}
}
