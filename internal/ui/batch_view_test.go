package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"api/internal/model"
	requestinput "api/internal/ui/requestInput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func batchFixture(t *testing.T, width, height int) AppModel {
	t.Helper()
	zone.NewGlobal()
	app := NewAppModel(model.NewSharedState())
	updated, _ := app.Update(tea.WindowSizeMsg{Width: width, Height: height})
	app = updated.(AppModel)
	app.BatchMode = true
	app.BatchFolder = "collections/auth/folder-06"
	app.BatchExpected = 9
	app.BatchSelected = 3
	app.BatchChain = true
	app.BatchRetries = 2
	names := []string{"Login admin", "Refresh token", "Get profile", "Update avatar", "Logout", "List users", "Create user", "Delete user", "Health check"}
	codes := []int{200, 201, 401, 0, 500, 204, 200, 0, 200}
	durs := []time.Duration{120 * time.Millisecond, 80 * time.Millisecond, 45 * time.Millisecond, 0, 320 * time.Millisecond, 60 * time.Millisecond, 90 * time.Millisecond, 0, 15 * time.Millisecond}
	methods := []string{"POST", "POST", "GET", "PUT", "DELETE", "GET", "POST", "DELETE", "GET"}
	app.BatchResults = make([]requestinput.ResponseMsg, len(names))
	app.BatchNames = names
	app.BatchSpecs = make([]requestinput.BatchSpec, len(names))
	for i := range names {
		app.BatchSpecs[i].Request.Method = methods[i]
		app.BatchResults[i] = requestinput.ResponseMsg{
			RequestName: names[i], Method: methods[i],
			URL:  "https://api.example.com/v1/" + strings.ReplaceAll(names[i], " ", "-"),
			Code: codes[i], Duration: durs[i], Attempts: 1,
		}
	}
	return app
}

func TestBatchViewFillsScreenWidth(t *testing.T) {
	sizes := []struct{ w, h int }{{105, 32}, {120, 40}, {150, 45}, {190, 60}}
	var dump strings.Builder
	for _, size := range sizes {
		app := batchFixture(t, size.w, size.h)
		panel := app.batchView()
		for i, line := range strings.Split(panel, "\n") {
			if got := lipgloss.Width(line); got != size.w {
				t.Fatalf("%dx%d baris %d: lebar %d, harus %d", size.w, size.h, i, got, size.w)
			}
		}
		footer := app.batchFooterView()
		if got := lipgloss.Width(footer); got != size.w {
			t.Fatalf("%dx%d footer: lebar %d, harus %d", size.w, size.h, got, size.w)
		}
		if got := lipgloss.Height(footer); got != 1 {
			t.Fatalf("%dx%d footer: tinggi %d, harus 1", size.w, size.h, got)
		}
		fmt.Fprintf(&dump, "\n=== %dx%d ===\n%s\n%s\n", size.w, size.h, panel, footer)
	}
	os.WriteFile("/tmp/batchview.txt", []byte(dump.String()), 0o644)
}

func TestBatchChooserFitsScreen(t *testing.T) {
	for _, size := range []struct{ w, h int }{{105, 32}, {150, 45}} {
		app := batchFixture(t, size.w, size.h)
		app.BatchChoosing = true
		view := app.batchChooserView()
		if got := lipgloss.Width(view); got != size.w {
			t.Fatalf("%dx%d chooser: lebar %d, harus %d", size.w, size.h, got, size.w)
		}
		if got := lipgloss.Height(view); got != size.h {
			t.Fatalf("%dx%d chooser: tinggi %d, harus %d", size.w, size.h, got, size.h)
		}
		os.WriteFile(fmt.Sprintf("/tmp/chooser_%d.txt", size.w), []byte(view), 0o644)
	}
}

func TestBatchResultsFillsScreen(t *testing.T) {
	for _, size := range []struct{ w, h int }{{105, 32}, {120, 40}, {160, 50}} {
		app := batchFixture(t, size.w, size.h)
		view := app.viewBatchResults()
		lines := strings.Split(view.Content, "\n")
		if len(lines) != size.h {
			t.Fatalf("%dx%d: tinggi view %d, harus %d", size.w, size.h, len(lines), size.h)
		}
		for i, line := range lines {
			if got := lipgloss.Width(line); got != size.w {
				t.Fatalf("%dx%d baris %d: lebar %d, harus %d", size.w, size.h, i, got, size.w)
			}
		}
	}
}
