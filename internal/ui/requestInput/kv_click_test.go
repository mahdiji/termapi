package requestinput

import (
	"strings"
	"testing"
	"time"

	"api/internal/collectionstore"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

func zoneClick(t *testing.T, id string) tea.MouseClickMsg {
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

func TestKVEditorClickStartsEditingImmediately(t *testing.T) {
	zone.NewGlobal()
	k := NewKVEditor("click_cells", "params", true, true)
	k.SetRows([]collectionstore.KVRow{{Key: "page", Value: "1", Description: "halaman"}})
	k.SetSize(90, 8)
	_ = zone.Scan(k.View())

	for _, col := range []int{0, 1, 2} {
		k.Blur()
		if !k.HandleClick(zoneClick(t, k.cellZone(0, col))) {
			t.Fatalf("click on column %d not handled", col)
		}
		if !k.Editing || k.Row != 0 || k.Col != col {
			t.Fatalf("column %d: editing=%v row=%d col=%d", col, k.Editing, k.Row, k.Col)
		}
	}
}

func TestKVEditorSelectedRowMarkerNotDuplicated(t *testing.T) {
	zone.NewGlobal()
	k := NewKVEditor("click_marker", "params", true, true)
	k.SetRows([]collectionstore.KVRow{{Key: "page", Value: "1", Description: "halaman"}})
	k.SetSize(90, 8)

	for _, disabled := range []bool{false, true} {
		k.Rows[0].Disabled = disabled
		lines := strings.Split(ansi.Strip(zone.Scan(k.View())), "\n")
		header, row := lines[0], lines[1]
		if !strings.HasPrefix(row, "› ") {
			t.Fatalf("disabled=%v: row marker missing: %q", disabled, row)
		}
		if strings.HasPrefix(row, "› ›") {
			t.Fatalf("disabled=%v: duplicate row marker: %q", disabled, row)
		}
		if ansi.StringWidth(header) != ansi.StringWidth(row) {
			t.Fatalf("disabled=%v: row width %d != header width %d",
				disabled, ansi.StringWidth(row), ansi.StringWidth(header))
		}
	}
}

func TestClickingEmptyDescriptionCellStartsEditing(t *testing.T) {
	zone.NewGlobal()
	m := New(nil)
	m.SetWidth(90)
	m.SetEditorSize(90, 8)
	m.Params.SetRows([]collectionstore.KVRow{{Key: "page", Value: "1"}})
	_ = zone.Scan(m.EditorView())

	next, _ := m.Update(zoneClick(t, m.Params.cellZone(0, 2)))
	if !next.Params.Editing || next.Params.Row != 0 || next.Params.Col != 2 {
		t.Fatalf("description cell not editable: editing=%v row=%d col=%d",
			next.Params.Editing, next.Params.Row, next.Params.Col)
	}
}

func TestTabWhileEditingKeepsEditingNextCell(t *testing.T) {
	k := NewKVEditor("tab_next", "params", true, true)
	k.SetRows([]collectionstore.KVRow{{}})
	k.startEditing()
	k.input.SetValue("page")

	_ = k.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !k.Editing || k.Col != 1 {
		t.Fatalf("tab should move into the value cell: editing=%v col=%d", k.Editing, k.Col)
	}

	k.input.SetValue("1")
	_ = k.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if k.Editing {
		t.Fatal("enter should commit and leave edit mode")
	}
	if k.Rows[0].Key != "page" || k.Rows[0].Value != "1" {
		t.Fatalf("values lost after moving with tab: %+v", k.Rows[0])
	}
}

func TestClickingTabCommitsPendingCellEdit(t *testing.T) {
	zone.NewGlobal()
	m := New(nil)
	m.SetWidth(90)
	m.SetEditorSize(90, 8)
	m.Params.SetRows([]collectionstore.KVRow{{Key: "page", Value: "1"}})
	_ = zone.Scan(m.EditorView())

	m.Params.Col = 1
	m.Params.startEditing()
	m.Params.input.SetValue("2")

	next, _ := m.Update(zoneClick(t, "request_tab_1"))
	if next.Tab != 1 {
		t.Fatalf("tab = %d, want 1", next.Tab)
	}
	if next.Params.Editing {
		t.Fatal("params editor still in edit mode after switching tabs")
	}
	if got := next.Params.Rows[0].Value; got != "2" {
		t.Fatalf("pending params value = %q, want 2", got)
	}
	if next.Headers.Editing {
		t.Fatal("headers editor unexpectedly in edit mode")
	}
}

func TestBlurCommitsPendingCellEdit(t *testing.T) {
	m := New(nil)
	m.Params.SetRows([]collectionstore.KVRow{{Key: "page", Value: "1"}})
	m.Params.Col = 1
	m.Params.startEditing()
	m.Params.input.SetValue("2")

	m.Blur()

	if m.Params.Editing {
		t.Fatal("params editor still in edit mode after blur")
	}
	if got := m.Params.Rows[0].Value; got != "2" {
		t.Fatalf("pending params value = %q, want 2", got)
	}
}

func TestClickingBodyFormCellStartsEditing(t *testing.T) {
	zone.NewGlobal()
	m := New(nil)
	m.SetWidth(90)
	m.SetEditorSize(90, 8)
	m.Tab = 2
	m.Body.SetType("form")
	m.Body.Fields.SetRows([]collectionstore.KVRow{{Key: "name", Value: "Ada"}})
	_ = zone.Scan(m.EditorView())

	next, _ := m.Update(zoneClick(t, m.Body.Fields.cellZone(0, 0)))
	if !next.Body.Fields.Editing || next.Body.Fields.Row != 0 || next.Body.Fields.Col != 0 {
		t.Fatalf("body field not editing: editing=%v row=%d col=%d",
			next.Body.Fields.Editing, next.Body.Fields.Row, next.Body.Fields.Col)
	}
}
