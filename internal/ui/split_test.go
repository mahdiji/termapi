package ui

import (
	"math"
	"testing"

	"api/internal/config"
	"api/internal/model"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func newSplitTestApp(t *testing.T, width, height int) AppModel {
	t.Helper()
	zone.NewGlobal()
	app := NewAppModel(model.NewSharedState())
	updated, _ := app.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(AppModel)
}

func TestCtrlUpDownResizesResponse(t *testing.T) {
	t.Chdir(t.TempDir())
	app := newSplitTestApp(t, 120, 40)

	beforeRequest := app.RequestHeight
	beforeResponse := app.ResponseHeight

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl})
	app = updated.(AppModel)
	if app.RequestHeight >= beforeRequest {
		t.Fatalf("Ctrl+Up: request height %d did not shrink from %d", app.RequestHeight, beforeRequest)
	}
	if app.ResponseHeight <= beforeResponse {
		t.Fatalf("Ctrl+Up: response height %d did not grow from %d", app.ResponseHeight, beforeResponse)
	}

	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl})
	app = updated.(AppModel)
	if app.RequestHeight <= beforeRequest-2 {
		t.Fatalf("Ctrl+Down: request height %d did not grow back", app.RequestHeight)
	}
	if app.ResponseHeight >= beforeResponse+2 {
		t.Fatalf("Ctrl+Down: response height %d did not shrink back", app.ResponseHeight)
	}
}

func TestSplitClampsBothPanes(t *testing.T) {
	t.Chdir(t.TempDir())
	app := newSplitTestApp(t, 120, 40)
	requestChrome := requestFrameHeight() + 3
	minRequest := requestChrome + minEditorHeight
	maxRequest := app.MainHeight - minPaneHeight

	for i := 0; i < 100; i++ {
		updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl})
		app = updated.(AppModel)
	}
	if app.RequestHeight > maxRequest {
		t.Fatalf("request height %d exceeds max %d", app.RequestHeight, maxRequest)
	}
	if app.ResponseHeight < minPaneHeight {
		t.Fatalf("response height %d below min %d", app.ResponseHeight, minPaneHeight)
	}

	for i := 0; i < 100; i++ {
		updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl})
		app = updated.(AppModel)
	}
	if app.RequestHeight < minRequest {
		t.Fatalf("request height %d below min %d", app.RequestHeight, minRequest)
	}
	if app.ResponseHeight > app.MainHeight-minRequest {
		t.Fatalf("response height %d exceeds max %d", app.ResponseHeight, app.MainHeight-minRequest)
	}
}

func TestResponseRatioSurvivesResizeAndPersists(t *testing.T) {
	t.Chdir(t.TempDir())
	app := newSplitTestApp(t, 120, 40)

	app.ResponseRatio = 0.5
	app.recalculateLayout()
	want := int(math.Round(float64(app.MainHeight) * 0.5))
	if diff := app.ResponseHeight - want; diff < -1 || diff > 1 {
		t.Fatalf("response height %d, want around %d", app.ResponseHeight, want)
	}

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 160, Height: 55})
	app = updated.(AppModel)
	want = int(math.Round(float64(app.MainHeight) * app.ResponseRatio))
	if diff := app.ResponseHeight - want; diff < -1 || diff > 1 {
		t.Fatalf("after resize response height %d, want around %d", app.ResponseHeight, want)
	}

	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl})
	app = updated.(AppModel)
	persisted := config.LoadResponseRatio("")
	if math.Abs(persisted-app.ResponseRatio) > 0.0001 {
		t.Fatalf("persisted ratio %.4f != in-memory %.4f", persisted, app.ResponseRatio)
	}
}
