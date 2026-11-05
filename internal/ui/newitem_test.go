package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"api/internal/model"
	"api/internal/ui/newscreen"

	tea "charm.land/bubbletea/v2"
)

func newItemTestApp(t *testing.T) AppModel {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("TERMAPI_COLLECTIONS_DIR", filepath.Join(dir, "collections"))
	app := NewAppModel(model.NewSharedState())
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(AppModel)
	if _, err := app.Sidebar.CollectionStore.CreateFolder("", "auth", "collection"); err != nil {
		t.Fatal(err)
	}
	app.Sidebar.ReloadCollections("")
	return app
}

func ctrlN() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl}
}

func enterKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEnter}
}

func TestCtrlNOpensFromAnyFocus(t *testing.T) {
	for _, focus := range []string{"url", "editor", "sidebar", "filter"} {
		t.Run(focus, func(t *testing.T) {
			app := newItemTestApp(t)
			switch focus {
			case "url":
				app.ReqState.FocusArea = 1
			case "editor":
				app.ReqState.FocusArea = 3
			case "sidebar":
				app.State.TabActive = 1
				app.Sidebar.IsFocused = true
			case "filter":
				app.Response.FilterFocued = true
			}

			updated, _ := app.Update(ctrlN())
			app = updated.(AppModel)
			if !app.NewItemMode {
				t.Fatalf("Ctrl+N did not open the composer from %s focus", focus)
			}
			if app.NewItemScreen.Kind != newscreen.KindRequest {
				t.Fatalf("composer kind = %v, want request", app.NewItemScreen.Kind)
			}
		})
	}
}

func TestCreateRequestFromComposer(t *testing.T) {
	app := newItemTestApp(t)

	updated, _ := app.Update(ctrlN())
	app = updated.(AppModel)
	app.NewItemScreen.Name.SetValue("Ping")

	updated, _ = app.Update(enterKey())
	app = updated.(AppModel)

	if app.NewItemMode {
		t.Fatal("composer should close after creating a request")
	}
	path := filepath.Join(app.Sidebar.CollectionStore.Root, "auth", "ping.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("created request missing: %v", err)
	}
	if app.ReqState.RequestFilePath != "auth/ping.yaml" {
		t.Fatalf("request path = %q", app.ReqState.RequestFilePath)
	}
}

func TestCreateTopLevelFolderFromComposer(t *testing.T) {
	app := newItemTestApp(t)

	updated, _ := app.Update(ctrlN())
	app = updated.(AppModel)
	updated, _ = app.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	app = updated.(AppModel)
	if app.NewItemScreen.Kind != newscreen.KindFolder {
		t.Fatal("Ctrl+T did not switch to folder mode")
	}
	app.NewItemScreen.Selected = 0
	app.NewItemScreen.Name.SetValue("Billing")

	updated, _ = app.Update(enterKey())
	app = updated.(AppModel)

	if app.NewItemMode {
		t.Fatal("composer should close after creating a folder")
	}
	path := filepath.Join(app.Sidebar.CollectionStore.Root, "billing")
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatalf("created folder missing: %v", err)
	}
	if app.Sidebar.SelectedZoneID != "folder_billing" {
		t.Fatalf("selected zone = %q, want folder_billing", app.Sidebar.SelectedZoneID)
	}
}

func TestComposerHandlesResizeAndEscape(t *testing.T) {
	app := newItemTestApp(t)

	updated, _ := app.Update(ctrlN())
	app = updated.(AppModel)
	updated, _ = app.Update(tea.WindowSizeMsg{Width: 140, Height: 50})
	app = updated.(AppModel)
	if app.NewItemScreen.Width != 140 || app.NewItemScreen.Height != 50 {
		t.Fatalf("composer size = %dx%d", app.NewItemScreen.Width, app.NewItemScreen.Height)
	}

	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	app = updated.(AppModel)
	if app.NewItemMode {
		t.Fatal("Esc should close the composer")
	}
}

func TestComposerViewRenders(t *testing.T) {
	app := newItemTestApp(t)
	updated, _ := app.Update(ctrlN())
	app = updated.(AppModel)

	view := app.View().Content
	if !strings.Contains(view, "NEW REQUEST") {
		t.Fatalf("composer view missing title:\n%s", view)
	}
	if !strings.Contains(view, "DESTINATION") || !strings.Contains(view, "auth") {
		t.Fatalf("composer view missing destination picker:\n%s", view)
	}
}
