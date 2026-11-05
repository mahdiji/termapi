package sidebar

import "testing"

func searchTestTree() *Model {
	return &Model{
		Nodes: []TreeNode{
			{
				Name:      "authentication",
				IsFolder:  true,
				StorePath: "authentication",
				ZoneID:    "folder_authentication",
				Children: []TreeNode{
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
				Children: []TreeNode{
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
	}
}

func TestGetSearchSuggestionsFindsFilesInCollapsedFolders(t *testing.T) {
	m := searchTestTree()

	results := m.GetSearchSuggestions("login")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	got := results[0]
	if got.IsFolder {
		t.Fatal("expected a request file")
	}
	if got.Path != "authentication/login.yaml" {
		t.Errorf("path = %q", got.Path)
	}
	if got.Method != "POST" || got.Name != "Login" || got.Collection != "authentication" {
		t.Errorf("unexpected metadata: %+v", got)
	}
}

func TestGetSearchSuggestionsMatchesMetadata(t *testing.T) {
	m := searchTestTree()

	tests := []struct {
		query string
		path  string
	}{
		{"users", "users/list.yaml"},
		{"POST", "authentication/login.yaml"},
		{"api.test/login", "authentication/login.yaml"},
		{"authentication/", "authentication/login.yaml"},
	}
	for _, tc := range tests {
		results := m.GetSearchSuggestions(tc.query)
		found := false
		for _, result := range results {
			if result.Path == tc.path {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("query %q did not return %q: %+v", tc.query, tc.path, results)
		}
	}
}

func TestGetSearchSuggestionsEmptyQueryIncludesFolders(t *testing.T) {
	m := searchTestTree()

	results := m.GetSearchSuggestions("")
	if len(results) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(results))
	}
	folders := 0
	for _, result := range results {
		if result.IsFolder {
			folders++
		}
	}
	if folders != 2 {
		t.Errorf("expected 2 folders, got %d", folders)
	}
}

func TestRevealPathOpensAncestors(t *testing.T) {
	m := searchTestTree()

	m.RevealPath("users/list.yaml")
	if !m.CollectionsOpen {
		t.Fatal("expected collections to be opened")
	}
	if !m.Nodes[1].Open {
		t.Fatal("expected the users folder to be expanded")
	}
	if m.Nodes[0].Open {
		t.Fatal("did not expect unrelated folders to be expanded")
	}
}
