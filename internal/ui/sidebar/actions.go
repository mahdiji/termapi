package sidebar

import (
	"path/filepath"
	"sort"
	"strings"

	"api/internal/collectionstore"
	"api/internal/config"
)

func removeNode(nodes *[]TreeNode, storePath string) bool {
	for i := range *nodes {
		if (*nodes)[i].StorePath == storePath {
			*nodes = append((*nodes)[:i], (*nodes)[i+1:]...)
			return true
		}
		if removeNode(&(*nodes)[i].Children, storePath) {
			return true
		}
	}
	return false
}

func (m *Model) selectZone(zoneID string) {
	m.SelectedZoneID = zoneID
	for i, item := range m.getFlatItems() {
		if item.ZoneID == zoneID {
			m.CursorIdx = i
			return
		}
	}
}

func (m *Model) SelectZone(zoneID string) {
	if m == nil {
		return
	}
	m.selectZone(zoneID)
}

func (m *Model) GetSearchSuggestions(query string) []SearchSuggestion {
	if m == nil {
		return nil
	}

	q := strings.ToLower(strings.TrimSpace(query))
	type match struct {
		suggestion SearchSuggestion
		score      int
	}
	var matches []match

	var walk func(nodes []TreeNode, collection, parent string)
	walk = func(nodes []TreeNode, collection, parent string) {
		for i := range nodes {
			node := &nodes[i]
			if node.IsFolder {
				coll := collection
				folderParent := parent
				if coll == "" {
					coll = node.Name
					folderParent = ""
				}
				s := SearchSuggestion{
					Text:       node.Name,
					ZoneID:     node.ZoneID,
					Path:       node.StorePath,
					Name:       node.Name,
					Folder:     folderParent,
					Collection: coll,
					Children:   len(node.Children),
					IsFolder:   true,
				}
				if rank, ok := searchRank(s, q); ok {
					matches = append(matches, match{suggestion: s, score: rank*2 + 1})
				}
				walk(node.Children, coll, node.StorePath)
				continue
			}

			label := requestRoute(node.RequestURL)
			if label == "" {
				label = node.RequestName
			}
			if label == "" {
				label = node.Name
			}
			s := SearchSuggestion{
				Text:       label,
				ZoneID:     node.ZoneID,
				Path:       node.StorePath,
				Name:       node.RequestName,
				Method:     node.Method,
				URL:        node.RequestURL,
				Folder:     parent,
				Collection: collection,
			}
			if s.Name == "" {
				s.Name = label
			}
			if rank, ok := searchRank(s, q); ok {
				matches = append(matches, match{suggestion: s, score: rank * 2})
			}
		}
	}
	walk(m.Nodes, "", "")

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score < matches[j].score
		}
		if matches[i].suggestion.Path != matches[j].suggestion.Path {
			return matches[i].suggestion.Path < matches[j].suggestion.Path
		}
		return matches[i].suggestion.Name < matches[j].suggestion.Name
	})

	results := make([]SearchSuggestion, 0, len(matches))
	for _, match := range matches {
		results = append(results, match.suggestion)
	}
	return results
}

// searchRank memberi prioritas hasil: exact match, prefix, lalu contains.
func searchRank(s SearchSuggestion, query string) (int, bool) {
	if query == "" {
		return 3, true
	}
	fields := []string{s.Name, s.Method, s.Path, s.URL, s.Folder, s.Collection, s.Text}
	best := -1
	for _, field := range fields {
		value := strings.ToLower(strings.TrimSpace(field))
		if value == "" {
			continue
		}
		rank := 3
		switch {
		case value == query:
			rank = 0
		case strings.HasPrefix(value, query):
			rank = 1
		case strings.Contains(value, query):
			rank = 2
		default:
			continue
		}
		if best == -1 || rank < best {
			best = rank
		}
	}
	if best == -1 {
		return 0, false
	}
	return best, true
}

// RevealPath membuka folder induk agar path hasil pencarian terlihat di sidebar.
func (m *Model) RevealPath(relPath string) {
	if m == nil || strings.TrimSpace(relPath) == "" {
		return
	}
	var reveal func(nodes []TreeNode) bool
	reveal = func(nodes []TreeNode) bool {
		for i := range nodes {
			node := &nodes[i]
			if node.StorePath == relPath {
				return true
			}
			if strings.HasPrefix(relPath, node.StorePath+"/") {
				if node.IsFolder {
					node.Open = true
				}
				if reveal(node.Children) {
					return true
				}
			}
		}
		return false
	}
	if reveal(m.Nodes) {
		m.CollectionsOpen = true
	}
}

// DeleteSelectedRequest removes only the selected request file and returns its path.
func (m *Model) DeleteSelectedRequest() string {
	if m == nil {
		return ""
	}
	path := m.SelectedRequestFilePath()
	if path == "" {
		m.StorageError = "Pilih file request terlebih dahulu (Ctrl+D hanya menghapus file)"
		return ""
	}
	m.deleteSelected()
	if m.StorageError != "" {
		return ""
	}
	return path
}

// SelectedRequestFilePath returns the selected request's collection-relative path.
func (m *Model) SelectedRequestFilePath() string {
	if m == nil {
		return ""
	}
	for _, item := range m.getFlatItems() {
		if item.ZoneID == m.SelectedZoneID && item.Type == itemTypeFile && item.NodePtr != nil {
			return item.NodePtr.StorePath
		}
	}
	return ""
}

func (m *Model) deleteSelected() {
	var selected *TreeNode
	for _, item := range m.getFlatItems() {
		if item.ZoneID == m.SelectedZoneID {
			selected = item.NodePtr
			break
		}
	}
	if selected == nil || selected.StorePath == "" || (!selected.IsFolder && selected.Method == "") {
		m.StorageError = "Select a collection folder or request file to delete"
		return
	}
	if m.CollectionStore == nil {
		m.CollectionStore = defaultCollectionStore()
	}
	if err := m.CollectionStore.Delete(selected.StorePath, selected.IsFolder); err != nil {
		m.StorageError = err.Error()
		return
	}
	removeNode(&m.Nodes, selected.StorePath)
	if len(m.Pinned) > 0 {
		before := len(m.Pinned)
		m.Pinned = removePinnedPaths(m.Pinned, selected.StorePath)
		if len(m.Pinned) != before {
			_ = config.SavePinned(m.ProjectRoot, m.Pinned)
		}
	}
	m.StorageError = ""
	m.CollectionsOpen = true
	m.selectZone("accordion_collections")
}

// SelectedRequestEntry loads the complete saved request selected in the sidebar.
func (m *Model) SelectedRequestEntry() (collectionstore.Entry, string, string, bool) {
	if m == nil || m.CollectionStore == nil {
		return collectionstore.Entry{}, "", "", false
	}
	var selectedPath string
	for _, item := range m.getFlatItems() {
		if item.ZoneID == m.SelectedZoneID && item.Type == itemTypeFile && item.NodePtr != nil {
			selectedPath = item.NodePtr.StorePath
			break
		}
	}
	if selectedPath == "" {
		return collectionstore.Entry{}, "", "", false
	}
	entries, err := m.CollectionStore.Load()
	if err != nil {
		m.StorageError = err.Error()
		return collectionstore.Entry{}, "", "", false
	}
	var selected collectionstore.Entry
	var rootName, rootPath string
	var walk func([]collectionstore.Entry, string, string)
	walk = func(items []collectionstore.Entry, currentRootName, currentRootPath string) {
		for _, entry := range items {
			if entry.RelPath == selectedPath {
				selected, rootName, rootPath = entry, currentRootName, currentRootPath
				return
			}
			name, folder := currentRootName, currentRootPath
			if currentRootPath == "" {
				name, folder = entry.Name, entry.RelPath
			}
			walk(entry.Children, name, folder)
			if selected.Request != nil {
				return
			}
		}
	}
	walk(entries, "", "")
	return selected, rootName, rootPath, selected.Request != nil
}

// ReloadCollections refreshes the sidebar after saving a request or creating folders.
func (m *Model) ReloadCollections(selectedPath string) {
	if m == nil || m.CollectionStore == nil {
		return
	}
	entries, err := m.CollectionStore.Load()
	if err != nil {
		m.StorageError = err.Error()
		return
	}
	m.Nodes = treeNodesFromEntries(entries)
	var openAll func([]TreeNode)
	openAll = func(nodes []TreeNode) {
		for i := range nodes {
			if nodes[i].IsFolder {
				nodes[i].Open = true
			}
			openAll(nodes[i].Children)
		}
	}
	openAll(m.Nodes)
	m.CollectionsOpen = true
	if selectedPath != "" {
		m.SelectedZoneID = "file_" + strings.ReplaceAll(filepath.ToSlash(selectedPath), "/", "__")
	}
	items := m.getFlatItems()
	for i, item := range items {
		if item.ZoneID == m.SelectedZoneID {
			m.CursorIdx = i
			break
		}
	}
	m.StorageError = ""
}

// SelectPath selects a request or folder by its collection-relative path.
func (m *Model) SelectPath(relPath string, isFolder bool) {
	if m == nil {
		return
	}
	prefix := "file_"
	if isFolder {
		prefix = "folder_"
	}
	m.selectZone(prefix + strings.ReplaceAll(filepath.ToSlash(relPath), "/", "__"))
}

func (m *Model) SelectedRequestDetails() (method, requestURL, collectionName, collectionPath, requestName string, ok bool) {
	for _, root := range m.Nodes {
		selected := findTreeNode(root, m.SelectedZoneID)
		if selected != nil && selected.Method != "" {
			return selected.Method, selected.RequestURL, root.Name, root.StorePath, selected.RequestName, true
		}
	}
	return "", "", "", "", "", false
}

// SelectedFolderPath returns the collection-relative path of the selected folder.
func (m *Model) SelectedFolderPath() string {
	if m == nil {
		return ""
	}
	for _, item := range m.getFlatItems() {
		if item.ZoneID == m.SelectedZoneID && item.Type == itemTypeFolder && item.NodePtr != nil {
			return item.NodePtr.StorePath
		}
	}
	return ""
}

// selectedRequestNode mengembalikan node request yang sedang dipilih.
func (m *Model) selectedRequestNode() *TreeNode {
	for _, item := range m.getFlatItems() {
		if item.ZoneID == m.SelectedZoneID && item.Type == itemTypeFile && item.NodePtr != nil {
			return item.NodePtr
		}
	}
	return nil
}

// togglePinSelected mem-pin atau melepas pin request yang dipilih, lalu
// menyimpannya ke .termapi/global.yaml.
func (m *Model) togglePinSelected() {
	if m == nil {
		return
	}
	node := m.selectedRequestNode()
	if node == nil || node.StorePath == "" {
		m.StorageError = "Pilih request untuk pin/unpin"
		return
	}
	path := node.StorePath
	pinned := !m.isPinned(path)
	if pinned {
		m.Pinned = append(m.Pinned, path)
		m.PinnedOpen = true
	} else {
		m.Pinned = removePinnedPaths(m.Pinned, path)
	}
	if err := config.SavePinned(m.ProjectRoot, m.Pinned); err != nil {
		m.StorageError = err.Error()
		return
	}
	m.StorageError = ""
	if pinned {
		m.selectZone(pinnedZoneID(path))
		return
	}
	m.selectZone("file_" + strings.ReplaceAll(filepath.ToSlash(path), "/", "__"))
}

// removePinnedPaths membuang path yang sama dengan prefix, atau berada di
// dalam folder prefix saat folder dihapus.
func removePinnedPaths(paths []string, prefix string) []string {
	out := paths[:0]
	for _, p := range paths {
		if p == prefix || strings.HasPrefix(p, strings.TrimSuffix(prefix, "/")+"/") {
			continue
		}
		out = append(out, p)
	}
	return out
}
