package sidebar

import (
	"os"
	"path/filepath"
	"strings"

	"api/internal/collectionstore"
	"api/internal/config"
	"api/internal/model"

	"charm.land/bubbles/v2/textinput"
)

func New(state *model.SharedState) *Model {
	w := 38
	h := 45

	if state != nil {
		if state.Widht > 0 {
			calculatedW := state.Widht / 3
			if calculatedW < 24 {
				w = 24
			} else {
				w = calculatedW
			}
		}
		if state.Height > 0 {
			h = state.Height - 2
			if h < 10 {
				h = 10
			}
		}
	}

	collectionStore := defaultCollectionStore()
	projectRoot := filepath.Dir(collectionStore.Root)
	pinned := config.LoadPinned(projectRoot)
	entries, loadErr := collectionStore.Load()
	nodes := treeNodesFromEntries(entries)
	if len(nodes) == 0 {
		nodes = []TreeNode{
			{
				Name:     "collections",
				IsFolder: true,
				Open:     true,
				ZoneID:   "folder_default",
			},
		}
	}
	if len(nodes) > 0 {
		nodes[0].Open = true
	}

	// workspacesNodes := []TreeNode{
	// 	{
	// 		Name:     "Default Workspace",
	// 		IsFolder: true,
	// 		Open:     true,
	// 		ZoneID:   "folder_default_ws",
	// 		Children: []TreeNode{
	// 			{Name: "production.env", IsFolder: false, ZoneID: "file_prod_ws"},
	// 			{Name: "staging.env", IsFolder: false, ZoneID: "file_staging_ws"},
	// 		},
	// 	},
	// }

	createInput := textinput.New()
	createInput.Prompt = ""
	createInput.Placeholder = "Folder or request name"

	return &Model{
		State:           state,
		Widht:           w,
		Height:          h,
		Nodes:           nodes,
		WorkspacesNodes: nil,
		CollectionsOpen: true,
		WorkspacesOpen:  false,
		Pinned:          pinned,
		PinnedOpen:      true,
		ProjectRoot:     projectRoot,
		SelectedZoneID:  "accordion_collections",
		CursorIdx:       0,
		ScrollY:         0,
		ScrollX:         0,
		CollectionStore: collectionStore,
		FolderCounter:   len(nodes),
		StorageError:    errorMessage(loadErr),
		CreateInput:     createInput,
	}
}

func defaultCollectionStore() *collectionstore.Store {
	if root := strings.TrimSpace(os.Getenv("TERMAPI_COLLECTIONS_DIR")); root != "" {
		return collectionstore.New(root)
	}
	workingDir, err := os.Getwd()
	if err == nil {
		for dir := workingDir; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return collectionstore.New(filepath.Join(dir, "collections"))
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return collectionstore.New("collections")
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// SetSize menyesuaikan lebar dan tinggi panel dengan area konten utama.
// contentHeight adalah tinggi yang tersedia setelah navbar dan footer,
// sehingga View() menghasilkan tepat contentHeight baris.
func (m *Model) SetSize(windowWidth, contentHeight int) {
	if m == nil {
		return
	}

	w := windowWidth / 3
	if w < 24 {
		w = 24
	}
	if w > 60 {
		w = 60
	}

	h := contentHeight
	if h < 10 {
		h = 10
	}

	m.Widht = w
	m.Height = h
}
