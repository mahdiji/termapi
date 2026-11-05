package requestinput

import (
	"testing"

	"api/internal/collectionstore"
	"api/internal/model"

	tea "charm.land/bubbletea/v2"
)

func spaceKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
}

func TestKVEditorSpaceTogglesRow(t *testing.T) {
	k := NewKVEditor("params", "params", false, false)
	k.SetRows([]collectionstore.KVRow{{Key: "page", Value: "1"}})

	k.Update(spaceKey())

	if !k.Rows[0].Disabled {
		t.Fatal("space should disable the selected row")
	}
	k.Update(spaceKey())
	if k.Rows[0].Disabled {
		t.Fatal("space should enable the selected row again")
	}
}

func TestRequestInputSpaceCyclesMethod(t *testing.T) {
	m := New(model.NewSharedState())
	m.FocusArea = 0
	if m.Method != "GET" {
		t.Fatalf("unexpected initial method: %s", m.Method)
	}

	m.Update(spaceKey())

	if m.Method != "POST" {
		t.Fatalf("space should cycle the method, got %s", m.Method)
	}
}

func TestAuthEditorSpaceTogglesAddTo(t *testing.T) {
	a := NewAuthEditor()
	a.SetType("api_key")
	a.values["add_to"] = "query"
	a.cursor = 2

	a.Update(spaceKey())

	if a.values["add_to"] != "header" {
		t.Fatalf("space should toggle add_to, got %q", a.values["add_to"])
	}
}
