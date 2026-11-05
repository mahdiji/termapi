package ui

import (
	"fmt"
	"strings"

	"api/internal/historydb"
	"api/internal/ui/diffscreen"
	tea "charm.land/bubbletea/v2"
)

func (m AppModel) updateDiffMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Saat panduan/pemilih baseline terbuka, Esc/q hanya menutup lapisan itu.
		if m.DiffScreen != nil && (m.DiffScreen.ShowHelp || m.DiffScreen.PickerOpen) {
			if msg.String() == "esc" || msg.String() == "q" {
				_, cmd := m.DiffScreen.Update(msg)
				return m, cmd
			}
		}
		switch msg.String() {
		case "esc", "q":
			m.DiffMode = false
			if m.DiffScreen != nil {
				m.DiffScreen.ShowHelp = false
				m.DiffScreen.PickerOpen = false
			}
			m.recalculateLayout()
			return m, nil
		}
	case tea.WindowSizeMsg:
		if m.DiffScreen != nil {
			m.DiffScreen.SetSize(msg.Width, msg.Height)
		}
	}

	if m.DiffScreen != nil {
		_, cmd := m.DiffScreen.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *AppModel) openDiffViewer() {
	if m == nil {
		return
	}
	orig := ""
	mod := ""
	var runs []historydb.Run
	matchedRequest := false

	method, reqURL := "", ""
	if m.ReqState != nil {
		method = m.ReqState.Method
		reqURL = strings.TrimSpace(m.ReqState.ResolvedURL())
	}

	// Prioritas: history dengan method + URL yang sama dengan request aktif,
	// supaya baseline tidak tertimpa request lain yang dijalankan setelahnya.
	if m.HistoryStore != nil {
		if method != "" && reqURL != "" {
			if history, err := m.HistoryStore.ListForRequest(method, reqURL, 50); err == nil && len(history) > 0 {
				runs = history
				matchedRequest = true
			}
		}
		if len(runs) == 0 {
			if history, err := m.HistoryStore.List(50); err == nil {
				runs = history
			}
		}
	}

	if len(runs) >= 2 {
		mod = runs[0].ResponseBody  // Run terbaru (Actual Response / Current Live Response)
		orig = runs[1].ResponseBody // Run sebelumnya (Expected Response / Baseline)
	} else if len(runs) == 1 {
		mod = runs[0].ResponseBody
	}

	// Fallback jika riwayat belum cukup
	if mod == "" && m.Response != nil {
		mod = m.Response.Body
	}

	if mod == "" {
		mod = "{\n  \"message\": \"Belum ada response terbaru. Jalankan request terlebih dahulu (Ctrl+Enter)\"\n}"
	}
	if orig == "" {
		orig = "{\n  \"message\": \"Belum ada response baseline (sebelumnya)\"\n}"
	}

	screen := diffscreen.NewModel(orig, mod)
	screen.Options = diffBaselineOptions(runs)
	switch {
	case len(runs) >= 2:
		screen.SelectedA = 1
		screen.LabelA = diffRunLabel("A", runs[1])
		screen.LabelB = diffRunLabel("B", runs[0])
	case len(runs) == 1:
		screen.LabelB = diffRunLabel("B", runs[0])
	}

	switch {
	case matchedRequest:
		screen.SourceNote = fmt.Sprintf("Sumber: %d run history untuk %s %s", len(runs), method, reqURL)
	case len(runs) > 0:
		screen.SourceNote = "Sumber: run terakhir global (belum ada history yang cocok dengan request aktif)"
	default:
		screen.SourceNote = "Sumber: belum ada history; memakai response aktif"
	}

	m.DiffScreen = screen
	if m.State != nil {
		m.DiffScreen.SetSize(m.State.Widht, m.State.Height)
	}
	m.DiffMode = true
}

func diffBaselineOptions(runs []historydb.Run) []diffscreen.BaselineOption {
	options := make([]diffscreen.BaselineOption, 0, len(runs))
	for _, run := range runs {
		options = append(options, diffscreen.BaselineOption{
			Label: diffOptionLabel(run),
			Body:  run.ResponseBody,
		})
	}
	return options
}

func diffOptionLabel(run historydb.Run) string {
	method := run.Method
	if method == "" {
		method = "?"
	}
	status := "—"
	if run.StatusCode > 0 {
		status = fmt.Sprint(run.StatusCode)
	}
	executed := "—"
	if !run.ExecutedAt.IsZero() {
		executed = run.ExecutedAt.Local().Format("2006-01-02 15:04:05")
	}
	target := run.RequestName
	if target == "" {
		target = run.URL
	}
	return fmt.Sprintf("%s %s · %s · %s", method, status, executed, target)
}

func diffRunLabel(tag string, run historydb.Run) string {
	method := run.Method
	if method == "" {
		method = "?"
	}
	status := "—"
	if run.StatusCode > 0 {
		status = fmt.Sprint(run.StatusCode)
	}
	executed := "—"
	if !run.ExecutedAt.IsZero() {
		executed = run.ExecutedAt.Local().Format("15:04:05")
	}
	return fmt.Sprintf("%s · %s %s · %s", tag, method, status, executed)
}
