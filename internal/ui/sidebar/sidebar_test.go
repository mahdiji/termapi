package sidebar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"api/internal/collectionstore"
	"api/internal/config"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func TestNewSidebar(t *testing.T) {
	m := New(nil)
	if m == nil {
		t.Fatalf("Expected sidebar model, got nil")
	}
	if len(m.Nodes) == 0 {
		t.Errorf("Expected initial nodes in sidebar")
	}
}

func TestGetFlatItemsAndHighlight(t *testing.T) {
	m := New(nil)
	items := m.getFlatItems()
	if len(items) == 0 {
		t.Fatalf("Expected flat items, got 0")
	}

	foundFolder := false
	for _, item := range items {
		if item.Type == itemTypeFolder || item.Type == itemTypeAccordion {
			foundFolder = true
			break
		}
	}
	if !foundFolder {
		t.Errorf("Expected folder or accordion item in flat items")
	}
}

func TestHorizontalScrollBounds(t *testing.T) {
	m := New(nil)
	m.Widht = 30

	for i := 0; i < 100; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Text: "l"})
	}

	items := m.getFlatItems()
	innerW := m.Widht - 2
	expectedMaxX := maxScrollX(items, innerW)

	if m.ScrollX > expectedMaxX {
		t.Errorf("ScrollX (%d) exceeded maxScrollX (%d)", m.ScrollX, expectedMaxX)
	}
	if m.ScrollX < 0 {
		t.Errorf("ScrollX (%d) cannot be negative", m.ScrollX)
	}
}

func TestAddFolderAction(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "collections")
	os.MkdirAll(dir, 0755)
	store := collectionstore.New(dir)
	store.CreateFolder("", "col1", "collection")
	m := New(nil)
	m.CollectionStore = store
	entries, _ := store.Load()
	m.Nodes = treeNodesFromEntries(entries)
	if len(m.Nodes) > 0 {
		m.Nodes[0].Open = true
		m.SelectedZoneID = m.Nodes[0].ZoneID
	}
	initialNodeCount := 0
	if len(m.Nodes) > 0 {
		initialNodeCount = len(m.Nodes[0].Children)
	}

	m.beginCreate(createRequest)
	m.CreateInput.SetValue("testreq")
	m.saveCreate()

	entries, _ = store.Load()
	m.Nodes = treeNodesFromEntries(entries)

	if len(m.Nodes) == 0 || len(m.Nodes[0].Children) <= initialNodeCount {
		t.Errorf("Expected new request to be added to node")
	}
}

func TestFolderCreationTargetChoice(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "collections")
	os.MkdirAll(dir, 0755)
	store := collectionstore.New(dir)
	store.CreateFolder("", "col1", "collection")
	store.CreateFolder("col1", "users", "folder")

	m := New(nil)
	m.CollectionStore = store
	entries, _ := store.Load()
	m.Nodes = treeNodesFromEntries(entries)
	if len(m.Nodes) > 0 {
		m.Nodes[0].Open = true
	}
	// Select "users" folder
	for _, item := range m.getFlatItems() {
		if item.NodePtr != nil && item.NodePtr.Name == "users" {
			m.SelectedZoneID = item.ZoneID
			break
		}
	}

	// Begin create folder while inside "users"
	m.beginCreate(createFolder)
	if !m.CreateHasTargetChoice {
		t.Errorf("Expected CreateHasTargetChoice to be true when creating folder inside a subfolder")
	}

	// Test toggling target to root collection
	m.Update(tea.KeyPressMsg{Text: "right"})
	if !m.CreateTargetIsRoot {
		t.Errorf("Expected CreateTargetIsRoot to be true after toggling right")
	}

	m.CreateInput.SetValue("new-root-col")
	m.saveCreate()

	entries, _ = store.Load()
	if len(entries) != 2 {
		t.Errorf("Expected 2 root collections, got %d", len(entries))
	}
}

func TestSidebarSpaceTogglesFolder(t *testing.T) {
	m := New(nil)
	m.Nodes = []TreeNode{{Name: "collections", IsFolder: true, Open: true, ZoneID: "folder_test"}}
	m.SelectedZoneID = "folder_test"

	m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})

	if m.Nodes[0].Open {
		t.Fatal("space should collapse the selected folder")
	}
}

func TestSidebarViewLines(t *testing.T) {
	zone.NewGlobal()
	m := New(nil)
	for _, w := range []int{15, 20, 24, 25, 30, 40} {
		for _, h := range []int{8, 10, 15, 20, 30} {
			m.Widht = w
			m.Height = h
			for scrollX := 0; scrollX <= 15; scrollX++ {
				m.ScrollX = scrollX
				view := m.View()
				lines := strings.Split(view, "\n")
				if len(lines) != h {
					t.Errorf("w=%d h=%d scrollX=%d line count=%d expected=%d", w, h, scrollX, len(lines), h)
				}
				for lineIdx, line := range lines {
					lw := lipgloss.Width(line)
					if lw != w {
						t.Errorf("w=%d h=%d scrollX=%d line %d width=%d expected=%d: %q", w, h, scrollX, lineIdx, lw, w, line)
					}
				}
			}
		}
	}
}

func TestSidebarNoShortcuts(t *testing.T) {
	zone.NewGlobal()
	m := New(nil)
	m.Widht = 24
	m.Height = 20

	view := m.View()
	if strings.Contains(view, "Ctrl+Backspace") || strings.Contains(view, "New request") {
		t.Fatalf("Expected sidebar view to not contain shortcuts like Ctrl+Backspace or New request")
	}
}

func TestPinTogglePersists(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "collections")
	store := collectionstore.New(dir)
	folder, err := store.CreateFolder("", "api", "collection")
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.SaveRequest(folder.RelPath, "", collectionstore.Request{
		Name: "Ping", Method: "GET", URL: "https://example.com/ping",
	})
	if err != nil {
		t.Fatal(err)
	}

	m := New(nil)
	m.CollectionStore = store
	m.ProjectRoot = root
	m.Pinned = nil
	entries, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	m.Nodes = treeNodesFromEntries(entries)
	m.Nodes[0].Open = true
	m.SelectPath(entry.RelPath, false)

	m.togglePinSelected()
	if len(m.Pinned) != 1 || m.Pinned[0] != entry.RelPath {
		t.Fatalf("expected pinned %q, got %v", entry.RelPath, m.Pinned)
	}
	if got := config.LoadPinned(root); len(got) != 1 || got[0] != entry.RelPath {
		t.Fatalf("pinned not persisted: %v", got)
	}

	foundSection := false
	for _, item := range m.getFlatItems() {
		if item.ZoneID == "accordion_pinned" {
			foundSection = true
		}
	}
	if !foundSection {
		t.Fatal("pinned section missing from flat items")
	}

	m.SelectZone(pinnedZoneID(entry.RelPath))
	if got := m.SelectedRequestFilePath(); got != entry.RelPath {
		t.Fatalf("pinned row should resolve to %q, got %q", entry.RelPath, got)
	}

	m.togglePinSelected()
	if len(m.Pinned) != 0 {
		t.Fatalf("expected unpinned, got %v", m.Pinned)
	}
	if got := config.LoadPinned(root); len(got) != 0 {
		t.Fatalf("unpin not persisted: %v", got)
	}
}
