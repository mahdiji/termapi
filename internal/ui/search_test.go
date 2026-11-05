package ui

import (
	"strings"
	"testing"

	"api/internal/ui/sidebar"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

func searchTestApp() AppModel {
	input := textinput.New()
	input.Prompt = "Search files: "
	input.Focus()

	return AppModel{
		Sidebar: &sidebar.Model{
			Nodes: []sidebar.TreeNode{
				{
					Name:      "authentication",
					IsFolder:  true,
					StorePath: "authentication",
					ZoneID:    "folder_authentication",
					Children: []sidebar.TreeNode{
						{
							Name:        "POST /login",
							Method:      "POST",
							RequestName: "Login",
							RequestURL:  "https://api.test/login",
							StorePath:   "authentication/login.yaml",
							ZoneID:      "file_authentication__login.yaml",
						},
					},
				},
				{
					Name:      "users",
					IsFolder:  true,
					StorePath: "users",
					ZoneID:    "folder_users",
					Children: []sidebar.TreeNode{
						{
							Name:        "GET /users",
							Method:      "GET",
							RequestName: "List users",
							RequestURL:  "https://api.test/users",
							StorePath:   "users/list.yaml",
							ZoneID:      "file_users__list.yaml",
						},
					},
				},
			},
			SelectedZoneID: "accordion_collections",
		},
		SearchInput:         input,
		SearchMode:          true,
		SearchSelectedIndex: 0,
		ScreenWidth:         120,
		ScreenHeight:        40,
	}
}

func TestSearchViewShowsFileInfo(t *testing.T) {
	zone.NewGlobal()
	m := searchTestApp()
	m.SearchInput.SetValue("list")

	out := ansi.Strip(m.viewSearch().Content)
	for _, want := range []string{
		"FILE SEARCH",
		"RESULTS · 1",
		"List users",
		"users/list.yaml",
		"REQUEST FILE",
		"SEARCH SCOPE",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("search view missing %q", want)
		}
	}
}

func TestSearchViewEmptyResultShowsHint(t *testing.T) {
	zone.NewGlobal()
	m := searchTestApp()
	m.SearchInput.SetValue("does-not-exist")

	out := ansi.Strip(m.viewSearch().Content)
	for _, want := range []string{"RESULTS · 0", "No files match", "Try a request name"} {
		if !strings.Contains(out, want) {
			t.Errorf("search view missing %q", want)
		}
	}
}

func TestSearchViewTracksWindowResize(t *testing.T) {
	zone.NewGlobal()
	m := searchTestApp()

	for _, size := range []struct {
		width  int
		height int
	}{
		{140, 50},
		{105, 32},
		{180, 60},
	} {
		updated, _ := m.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
		next, ok := updated.(AppModel)
		if !ok {
			t.Fatalf("expected AppModel, got %T", updated)
		}
		if next.ScreenWidth != size.width || next.ScreenHeight != size.height {
			t.Fatalf("%dx%d: screen size = %dx%d", size.width, size.height, next.ScreenWidth, next.ScreenHeight)
		}

		out := ansi.Strip(next.viewSearch().Content)
		lines := strings.Split(out, "\n")
		if len(lines) != size.height {
			t.Fatalf("%dx%d: rendered %d lines", size.width, size.height, len(lines))
		}
		for i, line := range lines {
			if got := ansi.StringWidth(line); got > size.width {
				t.Fatalf("%dx%d: line %d rendered %d cells", size.width, size.height, i, got)
			}
		}
	}
}

func TestSearchEnterRevealsCollapsedFile(t *testing.T) {
	m := searchTestApp()
	m.SearchInput.SetValue("login")

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	next, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel, got %T", updated)
	}
	if next.SearchMode {
		t.Fatal("expected search to close after Enter")
	}
	if !next.Sidebar.CollectionsOpen {
		t.Fatal("expected collections to be expanded")
	}
	if !next.Sidebar.Nodes[0].Open {
		t.Fatal("expected authentication folder to be expanded")
	}
	if got := next.Sidebar.SelectedZoneID; got != "file_authentication__login.yaml" {
		t.Fatalf("selected zone = %q", got)
	}
}
