package sidebar

import (
	"api/internal/collectionstore"
	"api/internal/model"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

type itemType string

const (
	itemTypeAccordion itemType = "accordion"
	itemTypeFolder    itemType = "folder"
	itemTypeFile      itemType = "file"
	itemTypeNotice    itemType = "notice"
)

type flatItem struct {
	ZoneID      string
	Text        string
	Style       lipgloss.Style
	AccentText  string
	AccentStyle lipgloss.Style
	Type        itemType
	NodePtr     *TreeNode
}

type TreeNode struct {
	Name        string
	IsFolder    bool
	Open        bool
	ZoneID      string
	StorePath   string
	RequestName string
	Method      string
	RequestURL  string
	Children    []TreeNode
}

type createMode int

const (
	createNone createMode = iota
	createFolder
	createRequest
)

// SearchSuggestion adalah satu entry yang bisa dicari: request file atau folder.
type SearchSuggestion struct {
	Text       string // label tampilan utama
	ZoneID     string
	Path       string // path relatif collection, mis. auth/login.yaml
	Name       string // nama request/folder
	Method     string
	URL        string
	Folder     string // path folder induk
	Collection string // nama collection root
	Children   int    // jumlah isi untuk folder
	IsFolder   bool
}

type sidebarShortcut struct {
	key    string
	action string
}

var sidebarShortcuts = []sidebarShortcut{
	{key: "Ctrl+N", action: "New item"},
	{key: "Ctrl+A", action: "New folder"},
	{key: "Ctrl+F", action: "File search"},
	{key: "Ctrl+D", action: "Delete file"},
	{key: "Ctrl+Backspace", action: "Delete selected"},
}

type Model struct {
	State                    *model.SharedState
	Widht                    int
	Height                   int
	Nodes                    []TreeNode
	WorkspacesNodes          []TreeNode
	CollectionsOpen          bool
	WorkspacesOpen           bool
	Pinned                   []string
	PinnedOpen               bool
	ProjectRoot              string
	SelectedZoneID           string
	CursorIdx                int
	FolderCounter            int
	ScrollY                  int
	ScrollX                  int
	IsFocused                bool
	CollectionStore          *collectionstore.Store
	StorageError             string
	CreateMode               createMode
	CreateInput              textinput.Model
	CreateParentPath         string
	CreateMethod             string
	CreateMethodFocus        bool
	CreateHasTargetChoice    bool
	CreateSelectedFolderPath string
	CreateSelectedFolderName string
	CreateTargetIsRoot       bool
	CreateTargetFocus        bool
	Filter                   string
}
