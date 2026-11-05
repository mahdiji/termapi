package ui

import (
	"strings"
	"testing"

	"api/internal/model"
	requestinput "api/internal/ui/requestInput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func TestSidebarHeightMatchesMainContent(t *testing.T) {
	zone.NewGlobal()
	sizes := []struct {
		width  int
		height int
	}{
		{105, 32},
		{120, 40},
		{160, 55},
	}

	for _, size := range sizes {
		app := NewAppModel(model.NewSharedState())
		updated, _ := app.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
		app = updated.(AppModel)

		if app.Sidebar.Height != app.MainHeight {
			t.Fatalf("%dx%d: sidebar height %d != main height %d",
				size.width, size.height, app.Sidebar.Height, app.MainHeight)
		}

		if got := lipgloss.Height(app.Sidebar.View()); got != app.MainHeight {
			t.Fatalf("%dx%d: rendered sidebar height %d != main height %d",
				size.width, size.height, got, app.MainHeight)
		}

		if got := lipgloss.Height(app.viewMain().Content); got != size.height {
			t.Fatalf("%dx%d: rendered layout height %d != window height %d",
				size.width, size.height, got, size.height)
		}
	}
}

func TestLongResponseKeepsLayoutWidth(t *testing.T) {
	zone.NewGlobal()

	app := NewAppModel(model.NewSharedState())
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(AppModel)

	body := strings.Repeat("var x = \"https://example.com/long/path/that/keeps/going/and/going.css\"; if (a) { b(); }\n", 80)
	app.Response.SetResponse(requestinput.ResponseMsg{
		Method: "GET", URL: "https://detik.com", Status: "200 OK", Code: 200,
		Body: body, RawBody: body, Size: len(body),
	})

	if got := lipgloss.Width(requestinput.RequestInputContent(app.ReqState)); got != app.ContentWidth {
		t.Fatalf("request bar rendered %d cells, want %d", got, app.ContentWidth)
	}

	view := app.viewMain().Content

	right := lipgloss.JoinVertical(lipgloss.Left,
		requestinput.RequestInputContent(app.ReqState),
		app.ReqState.EditorView(),
		app.Response.View(),
	)
	if got := lipgloss.Height(right); got != app.MainHeight {
		t.Fatalf("right column rendered %d lines, want %d", got, app.MainHeight)
	}
	if got := lipgloss.Height(app.Sidebar.View()); got != app.MainHeight {
		t.Fatalf("sidebar rendered %d lines, want %d", got, app.MainHeight)
	}

	lines := strings.Split(view, "\n")
	if len(lines) != app.ScreenHeight {
		t.Fatalf("rendered %d lines, want %d", len(lines), app.ScreenHeight)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got > app.ScreenWidth {
			t.Fatalf("line %d rendered %d cells, exceeds window width %d", i, got, app.ScreenWidth)
		}
	}
}
