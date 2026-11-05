package ui

import (
	"testing"
	"time"

	"api/internal/collectionstore"
	"api/internal/model"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func waitZoneClick(t *testing.T, id string) tea.MouseClickMsg {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		if z := zone.Get(id); !z.IsZero() {
			return tea.MouseClickMsg{X: z.StartX, Y: z.StartY, Button: tea.MouseLeft}
		}
		if time.Now().After(deadline) {
			t.Fatalf("zone %q not registered", id)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestClickingParamsCellStartsEditingFromApp(t *testing.T) {
	zone.NewGlobal()
	app := NewAppModel(model.NewSharedState())
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(AppModel)
	app.ReqState.Params.SetRows([]collectionstore.KVRow{{Key: "page", Value: "1"}})
	app.ReqState.Tab = 0
	app.ReqState.FocusArea = 1
	_ = app.View()

	app.Update(waitZoneClick(t, "params_cell_0_1"))

	if !app.ReqState.Params.Editing {
		t.Fatal("clicking a params cell did not start editing")
	}
	if app.ReqState.Params.Row != 0 || app.ReqState.Params.Col != 1 {
		t.Fatalf("selected cell = row %d col %d, want row 0 col 1",
			app.ReqState.Params.Row, app.ReqState.Params.Col)
	}
}
