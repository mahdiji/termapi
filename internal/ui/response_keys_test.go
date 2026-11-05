package ui

import (
	"testing"

	"api/internal/model"
	requestinput "api/internal/ui/requestInput"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func newResponseTestApp(t *testing.T) AppModel {
	t.Helper()
	app := NewAppModel(model.NewSharedState())
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(AppModel)
	app.Response.SetResponse(requestinput.ResponseMsg{
		Status:  "200 OK",
		Code:    200,
		Body:    "{\n  \"ok\": true\n}",
		RawBody: "{\"ok\":true}",
	})
	app.Response.Focus()
	return app
}

func TestResponseTabShortcuts(t *testing.T) {
	app := newResponseTestApp(t)

	app = pressKey(t, app, tea.KeyPressMsg{Code: tea.KeyTab})
	if app.Response.Tab != 1 {
		t.Fatalf("tab = %d, want 1", app.Response.Tab)
	}
	app = pressKey(t, app, tea.KeyPressMsg{Code: tea.KeyTab})
	if app.Response.Tab != 2 {
		t.Fatalf("tab = %d, want 2", app.Response.Tab)
	}
	app = pressKey(t, app, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if app.Response.Tab != 1 {
		t.Fatalf("shift+tab = %d, want 1", app.Response.Tab)
	}
}

func TestCtrlYStartsCopyNotice(t *testing.T) {
	app := newResponseTestApp(t)
	app = pressKey(t, app, tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	if app.Response.Notice == "" {
		t.Fatal("expected a copy notice after Ctrl+Y")
	}
}

func TestEscReleasesResponseFocus(t *testing.T) {
	app := newResponseTestApp(t)
	app = pressKey(t, app, tea.KeyPressMsg{Code: tea.KeyEsc})
	if app.Response.Focused {
		t.Fatal("esc should leave response focus")
	}
}

func TestResponseMouseDragSelectsFromApp(t *testing.T) {
	zone.NewGlobal()
	app := NewAppModel(model.NewSharedState())
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(AppModel)
	app.Response.SetResponse(requestinput.ResponseMsg{
		Status:  "200 OK",
		Code:    200,
		Body:    "alpha\nbeta\ngamma\ndelta",
		RawBody: "alpha\nbeta\ngamma\ndelta",
	})
	_ = app.View()

	click := waitZoneClick(t, "response_body")
	click.Y++
	updated, _ = app.Update(click)
	app = updated.(AppModel)
	updated, _ = app.Update(tea.MouseMotionMsg{X: click.X, Y: click.Y + 1, Button: tea.MouseLeft})
	app = updated.(AppModel)

	text, lines, ok := app.Response.CopySelection()
	if !ok || lines != 2 || text != "beta\ngamma" {
		t.Fatalf("app selection = %q (%d lines, ok=%v)", text, lines, ok)
	}

	updated, _ = app.Update(tea.MouseReleaseMsg{X: click.X, Y: click.Y + 1, Button: tea.MouseLeft})
	app = updated.(AppModel)
	if !app.Response.SelActive {
		t.Fatal("selection should stay active after release")
	}
}
