package ui

import (
	"errors"
	"path"
	"strings"

	"api/internal/collectionstore"
)

type saveDestination struct {
	Path           string
	Name           string
	CollectionName string
	Depth          int
}

func (m *AppModel) focusSaveInput() {
	m.SaveName.Blur()
	m.SaveSearch.Blur()
	switch m.SaveFocus {
	case 0:
		m.SaveName.Focus()
	case 1:
		m.SaveSearch.Focus()
	}
}

func (m *AppModel) openSaveDialog() {
	if m.ReqState == nil || m.Sidebar == nil || m.Sidebar.CollectionStore == nil {
		return
	}
	if m.ReqState.RequestFilePath != "" {
		if err := m.saveRequest(path.Dir(m.ReqState.RequestFilePath), m.ReqState.RequestFilePath); err != nil {
			m.SaveError = err.Error()
		} else {
			m.SaveError = ""
		}
		return
	}
	name := strings.TrimSpace(m.ReqState.RequestName)
	if name == "" {
		name = "New request"
	}
	m.SaveName.SetValue(name)
	m.SaveSearch.SetValue("")
	preferred := m.ReqState.CollectionPath
	if preferred == "" {
		preferred = m.Sidebar.SelectedFolderPath()
	}
	entries, err := m.Sidebar.CollectionStore.Load()
	if err != nil {
		m.SaveError = err.Error()
		return
	}
	m.SaveFolders = m.SaveFolders[:0]
	for _, entry := range entries {
		flattenSaveFolders(entry, 0, entry.Name, &m.SaveFolders)
	}
	m.SaveVisible = nil
	m.SaveSelected = 0
	m.SaveScroll = 0
	m.refreshSaveFolders(preferred)
	m.SaveName.SetWidth(max(m.ScreenWidth-24, 20))
	m.SaveSearch.SetWidth(max(m.ScreenWidth-24, 20))
	m.SaveError = ""
	m.SaveFocus = 0
	m.SaveDialog = true
	m.focusSaveInput()
}

func flattenSaveFolders(entry collectionstore.Entry, depth int, collectionName string, out *[]saveDestination) {
	if !entry.IsFolder {
		return
	}
	*out = append(*out, saveDestination{
		Path: entry.RelPath, Name: entry.Name, CollectionName: collectionName, Depth: depth,
	})
	for _, child := range entry.Children {
		flattenSaveFolders(child, depth+1, collectionName, out)
	}
}

func (m *AppModel) refreshSaveFolders(preferred string) {
	if preferred == "" && m.SaveSelected >= 0 && m.SaveSelected < len(m.SaveVisible) {
		preferred = m.SaveVisible[m.SaveSelected].Path
	}
	query := strings.ToLower(strings.TrimSpace(m.SaveSearch.Value()))
	m.SaveVisible = m.SaveVisible[:0]
	for _, folder := range m.SaveFolders {
		if query == "" || strings.Contains(strings.ToLower(folder.Name+" "+folder.Path), query) {
			m.SaveVisible = append(m.SaveVisible, folder)
		}
	}
	m.SaveSelected = 0
	for i, folder := range m.SaveVisible {
		if folder.Path == preferred {
			m.SaveSelected = i
			break
		}
	}
	m.keepSaveSelectionVisible()
}

func (m *AppModel) saveListRows() int {
	return max(m.ScreenHeight-27, 4)
}

func (m *AppModel) keepSaveSelectionVisible() {
	rows := m.saveListRows()
	if m.SaveSelected < m.SaveScroll {
		m.SaveScroll = m.SaveSelected
	}
	if m.SaveSelected >= m.SaveScroll+rows {
		m.SaveScroll = m.SaveSelected - rows + 1
	}
	maxScroll := max(len(m.SaveVisible)-rows, 0)
	m.SaveScroll = min(max(m.SaveScroll, 0), maxScroll)
}

func (m *AppModel) moveSaveSelection(delta int) {
	if len(m.SaveVisible) == 0 {
		return
	}
	m.SaveSelected = min(max(m.SaveSelected+delta, 0), len(m.SaveVisible)-1)
	m.keepSaveSelectionVisible()
}

func (m *AppModel) saveDialogRequest() {
	if m.ReqState == nil || m.Sidebar == nil || m.Sidebar.CollectionStore == nil {
		return
	}
	name := strings.TrimSpace(m.SaveName.Value())
	if name == "" {
		m.SaveError = "Masukkan nama request"
		return
	}
	var folderPath, collectionName string
	if len(m.SaveVisible) > 0 {
		selected := min(max(m.SaveSelected, 0), len(m.SaveVisible)-1)
		destination := m.SaveVisible[selected]
		folderPath, collectionName = destination.Path, destination.CollectionName
	} else {
		// Tetap mendukung pembuatan path baru: ketik path lengkap di pencarian
		// ketika tidak ada folder yang cocok, misalnya Api/Users.
		var err error
		folderPath, collectionName, err = m.ensureSaveFolder(m.SaveSearch.Value())
		if err != nil {
			m.SaveError = err.Error()
			return
		}
	}
	if err := m.saveRequest(folderPath, ""); err != nil {
		m.SaveError = err.Error()
		return
	}
	m.ReqState.RequestName = name
	m.ReqState.CollectionName = collectionName
	m.ReqState.CollectionPath = folderPath
	m.SaveDialog = false
	m.SaveFolders = nil
	m.SaveVisible = nil
	m.SaveError = ""
}

func (m *AppModel) saveRequest(folderPath, existingPath string) error {
	if m.ReqState == nil || m.Sidebar == nil || m.Sidebar.CollectionStore == nil {
		return errors.New("collection store is not available")
	}
	name := strings.TrimSpace(m.ReqState.RequestName)
	if existingPath == "" {
		name = strings.TrimSpace(m.SaveName.Value())
	}
	if name == "" {
		return errors.New("request name cannot be empty")
	}
	request, err := m.ReqState.Snapshot()
	if err != nil {
		return err
	}
	request.Name = name
	entry, err := m.Sidebar.CollectionStore.SaveRequest(folderPath, existingPath, request)
	if err != nil {
		return err
	}
	m.ReqState.RequestName = request.Name
	m.ReqState.RequestFilePath = entry.RelPath
	m.ReqState.CollectionPath = folderPath
	if m.ReqState.CollectionName == "" {
		m.ReqState.CollectionName = strings.Split(folderPath, "/")[0]
	}
	m.Sidebar.ReloadCollections(entry.RelPath)
	return nil
}

func (m *AppModel) ensureSaveFolder(raw string) (string, string, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if value == "" || strings.HasPrefix(value, "/") {
		return "", "", errors.New("masukkan path collection/folder, misalnya Api/Users")
	}
	segments := strings.Split(value, "/")
	parent := ""
	var rootName string
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "." || segment == ".." {
			return "", "", errors.New("path folder tidak valid")
		}
		entries, err := m.Sidebar.CollectionStore.Load()
		if err != nil {
			return "", "", err
		}
		var children []collectionstore.Entry
		if parent == "" {
			children = entries
		} else if found := findEntryPath(entries, parent); found != nil {
			children = found.Children
		} else {
			return "", "", errors.New("folder induk tidak ditemukan")
		}
		var folder *collectionstore.Entry
		for i := range children {
			if children[i].IsFolder && (strings.EqualFold(children[i].Name, segment) || strings.EqualFold(children[i].ID, segment)) {
				folder = &children[i]
				break
			}
		}
		if folder == nil {
			kind := "folder"
			if parent == "" {
				kind = "collection"
			}
			created, createErr := m.Sidebar.CollectionStore.CreateFolder(parent, segment, kind)
			if createErr != nil {
				return "", "", createErr
			}
			folder = &created
		}
		parent = folder.RelPath
		if rootName == "" {
			rootName = folder.Name
		}
	}
	return parent, rootName, nil
}

func findEntryPath(entries []collectionstore.Entry, relPath string) *collectionstore.Entry {
	for i := range entries {
		if entries[i].RelPath == relPath {
			return &entries[i]
		}
		if found := findEntryPath(entries[i].Children, relPath); found != nil {
			return found
		}
	}
	return nil
}
