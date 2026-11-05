package newscreen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"api/internal/collectionstore"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "ctrl+t", "ctrl+f", "tab", "shift+tab", "enter", "esc", "up", "down", "left", "right":
		switch s {
		case "ctrl+t":
			return tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl}
		case "ctrl+f":
			return tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}
		case "tab":
			return tea.KeyPressMsg{Code: tea.KeyTab}
		case "shift+tab":
			return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
		case "enter":
			return tea.KeyPressMsg{Code: tea.KeyEnter}
		case "esc":
			return tea.KeyPressMsg{Code: tea.KeyEsc}
		case "up":
			return tea.KeyPressMsg{Code: tea.KeyUp}
		case "down":
			return tea.KeyPressMsg{Code: tea.KeyDown}
		case "left":
			return tea.KeyPressMsg{Code: tea.KeyLeft}
		case "right":
			return tea.KeyPressMsg{Code: tea.KeyRight}
		}
	}
	return tea.KeyPressMsg{Text: s, Code: rune(s[0])}
}

func newStore(t *testing.T) *collectionstore.Store {
	t.Helper()
	return collectionstore.New(t.TempDir())
}

func TestDestinationsFollowKind(t *testing.T) {
	store := newStore(t)
	if _, err := store.CreateFolder("", "auth", "collection"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateFolder("auth", "nested", "folder"); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetStore(store)
	m.SetSize(120, 40)
	m.Open(KindRequest, "")

	if len(m.visible) != 2 {
		t.Fatalf("request destinations = %d, want 2 folders (root excluded)", len(m.visible))
	}
	if m.visible[0].IsRoot {
		t.Fatal("request mode must not offer the collections root")
	}

	m.Update(key("ctrl+t"))
	if m.Kind != KindFolder {
		t.Fatal("Ctrl+T did not switch to folder mode")
	}
	if len(m.visible) != 3 || !m.visible[0].IsRoot {
		t.Fatalf("folder destinations = %d, want root + 2 folders", len(m.visible))
	}
}

func TestSearchFiltersDestinations(t *testing.T) {
	store := newStore(t)
	for _, name := range []string{"Auth", "Users", "Billing"} {
		if _, err := store.CreateFolder("", name, "collection"); err != nil {
			t.Fatal(err)
		}
	}

	m := New()
	m.SetStore(store)
	m.SetSize(120, 40)
	m.Open(KindRequest, "")
	m.setFocus(FocusSearch)

	for _, r := range "use" {
		m.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	if len(m.visible) != 1 || m.visible[0].Name != "Users" {
		t.Fatalf("filtered destinations = %+v, want only Users", m.visible)
	}

	if cmd := m.Search.Value(); cmd != "use" {
		t.Fatalf("search value = %q", cmd)
	}
}

func TestSubmitCreatesRequestAndFolder(t *testing.T) {
	store := newStore(t)
	if _, err := store.CreateFolder("", "auth", "collection"); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetStore(store)
	m.SetSize(120, 40)
	m.Open(KindRequest, "auth")
	m.Name.SetValue("Login")
	entry, err := m.Submit()
	if err != nil {
		t.Fatalf("submit request: %v", err)
	}
	if filepath.ToSlash(entry.RelPath) != "auth/login.yaml" {
		t.Fatalf("request path = %q", entry.RelPath)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "auth", "login.yaml")); err != nil {
		t.Fatalf("request file missing: %v", err)
	}

	m.Open(KindFolder, "")
	m.moveTo(0)
	m.Name.SetValue("New Collection")
	entry, err = m.Submit()
	if err != nil {
		t.Fatalf("submit folder: %v", err)
	}
	if !entry.IsFolder || entry.RelPath != "new-collection" {
		t.Fatalf("folder entry = %+v", entry)
	}
}

func TestSubmitRequestWithoutFoldersFails(t *testing.T) {
	m := New()
	m.SetStore(newStore(t))
	m.SetSize(120, 40)
	m.Open(KindRequest, "")
	m.Name.SetValue("Ping")
	if _, err := m.Submit(); err == nil {
		t.Fatal("expected error when no destination folder exists")
	}
}

func TestViewFitsScreen(t *testing.T) {
	store := newStore(t)
	if _, err := store.CreateFolder("", "Authentication With A Very Long Collection Name", "collection"); err != nil {
		t.Fatal(err)
	}
	m := New()
	m.SetStore(store)
	m.SetSize(120, 40)
	m.Open(KindRequest, "")
	m.Err = strings.Repeat("x", 200)

	lines := strings.Split(m.View(), "\n")
	if len(lines) != 40 {
		t.Fatalf("view height = %d, want 40", len(lines))
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w > 120 {
			t.Fatalf("line %d width = %d, exceeds 120", i, w)
		}
	}
}

func TestSpaceCyclesMethod(t *testing.T) {
	m := New()
	m.Focus = FocusMethod
	if m.Method != "GET" {
		t.Fatalf("unexpected initial method: %s", m.Method)
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})

	if m.Method != "POST" {
		t.Fatalf("space should cycle the method, got %s", m.Method)
	}
}
