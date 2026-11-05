package historyscreen

import (
	"strings"
	"testing"
	"time"

	"api/internal/historydb"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func TestHistoryScreenScrollingAndSearch(t *testing.T) {
	m := &Model{}
	m.SetSize(80, 24)

	runs := []historydb.Run{
		{RequestName: "Req 1", Method: "GET", URL: "http://example.com/1", StatusCode: 200, ResponseTime: 10, ResponseBody: "line1\nline2\nline3", ExecutedAt: time.Now()},
		{RequestName: "Req 2", Method: "POST", URL: "http://example.com/2", StatusCode: 201, ResponseTime: 15, ResponseBody: "body2", ExecutedAt: time.Now()},
	}

	m.SetRuns(runs)
	if len(m.Runs) != 2 {
		t.Fatalf("Expected 2 runs, got %d", len(m.Runs))
	}

	// Test cursor navigation
	m.Update(tea.KeyPressMsg{Text: "j"})
	if m.Cursor != 1 {
		t.Errorf("Expected cursor at 1, got %d", m.Cursor)
	}

	m.Update(tea.KeyPressMsg{Text: "k"})
	if m.Cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", m.Cursor)
	}

	// Test Home / End
	m.Update(tea.KeyPressMsg{Text: "end"})
	if m.Cursor != 1 {
		t.Errorf("Expected cursor at 1 (End), got %d", m.Cursor)
	}

	m.Update(tea.KeyPressMsg{Text: "home"})
	if m.Cursor != 0 {
		t.Errorf("Expected cursor at 0 (Home), got %d", m.Cursor)
	}

	// Test Search
	m.StartSearch()
	if !m.Search.Focused() {
		t.Errorf("Expected search input to be focused")
	}

	m.SetQuery("POST")
	if len(m.Runs) != 1 || m.Runs[0].Method != "POST" {
		t.Errorf("Expected search filter to match 1 run (POST), got %d runs", len(m.Runs))
	}

	m.StopSearch()
	if m.Search.Focused() {
		t.Errorf("Expected search input to be blurred")
	}
}

func TestHistoryScreenViewWidth(t *testing.T) {
	zone.NewGlobal()
	m := &Model{}
	height := 30
	m.SetSize(100, height)

	runs := []historydb.Run{
		{RequestName: "Req 1", Method: "GET", URL: "http://example.com/1", StatusCode: 200, ResponseTime: 10, ResponseBody: "line1\nline2\nline3", ExecutedAt: time.Now()},
		{RequestName: "Req 2", Method: "POST", URL: "http://example.com/2", StatusCode: 201, ResponseTime: 15, ResponseBody: "body2", ExecutedAt: time.Now()},
	}
	m.SetRuns(runs)

	for _, width := range []int{60, 80, 100, 120} {
		m.SetSize(width, height)
		output := m.View()
		lines := strings.Split(output, "\n")
		for i, line := range lines {
			w := lipgloss.Width(line)
			if w != width {
				t.Errorf("Width %d: line %d width = %d, expected %d: %q", width, i, w, width, line)
			}
		}
	}
}

func TestSelectedRunFollowsCursor(t *testing.T) {
	m := &Model{}
	m.SetSize(100, 30)
	m.SetRuns([]historydb.Run{
		{RequestName: "First", Method: "GET"},
		{RequestName: "Second", Method: "POST"},
	})

	run, ok := m.SelectedRun()
	if !ok || run.RequestName != "First" {
		t.Fatalf("unexpected first selection: %+v", run)
	}

	m.Update(tea.KeyPressMsg{Text: "j"})

	run, ok = m.SelectedRun()
	if !ok || run.RequestName != "Second" {
		t.Fatalf("unexpected selection after j: %+v", run)
	}
}

func TestFormatResponseTime(t *testing.T) {
	cases := []struct {
		ms   int64
		want string
	}{
		{0, "0ms"},
		{999, "999ms"},
		{1000, "1s"},
		{1234, "1.23s"},
		{9999, "10s"},
		{12345, "12.3s"},
		{60000, "1m"},
		{65000, "1m 5s"},
		{3600000, "1h"},
		{3660000, "1h 1m"},
	}
	for _, tc := range cases {
		if got := formatResponseTime(tc.ms); got != tc.want {
			t.Errorf("formatResponseTime(%d) = %q, want %q", tc.ms, got, tc.want)
		}
	}
}
