package sidebar

import (
	"fmt"
	"path/filepath"
	"strings"

	"api/internal/collectionstore"
	tea "charm.land/bubbletea/v2"
)

func (m *Model) createParentPath(mode createMode) (string, error) {
	var selected *TreeNode
	for _, item := range m.getFlatItems() {
		if item.ZoneID == m.SelectedZoneID {
			selected = item.NodePtr
			break
		}
	}

	if selected != nil {
		if selected.StorePath == "" {
			return "", fmt.Errorf("this sidebar item is not stored in a collection")
		}
		if selected.IsFolder {
			return selected.StorePath, nil
		}
		return filepath.ToSlash(filepath.Dir(filepath.FromSlash(selected.StorePath))), nil
	}

	if m.SelectedZoneID != "accordion_collections" {
		return "", fmt.Errorf("select a collection or folder first")
	}
	if mode == createRequest {
		if len(m.Nodes) == 0 {
			return "", fmt.Errorf("create a collection before adding a request")
		}
		m.Nodes[0].Open = true
		return m.Nodes[0].StorePath, nil
	}
	return "", nil
}

func (m *Model) beginCreate(mode createMode) tea.Cmd {
	parentPath, err := m.createParentPath(mode)
	if err != nil {
		m.StorageError = err.Error()
		return nil
	}
	m.StorageError = ""
	m.CreateMode = mode
	m.CreateParentPath = parentPath
	m.CreateMethod = "GET"
	m.CreateMethodFocus = false
	m.CreateHasTargetChoice = false
	m.CreateTargetFocus = false
	m.CreateTargetIsRoot = false
	m.CreateSelectedFolderPath = ""
	m.CreateSelectedFolderName = ""

	if mode == createFolder {
		var selected *TreeNode
		for _, item := range m.getFlatItems() {
			if item.ZoneID == m.SelectedZoneID {
				selected = item.NodePtr
				break
			}
		}
		var selectedFolderStorePath string
		var selectedFolderName string
		if selected != nil && selected.StorePath != "" {
			if selected.IsFolder {
				selectedFolderStorePath = selected.StorePath
				selectedFolderName = selected.Name
			} else {
				selectedFolderStorePath = filepath.ToSlash(filepath.Dir(filepath.FromSlash(selected.StorePath)))
				if selectedFolderStorePath != "." && selectedFolderStorePath != "" {
					selectedFolderName = filepath.Base(selectedFolderStorePath)
				}
			}
		}
		if selectedFolderStorePath != "" && selectedFolderStorePath != "." {
			m.CreateHasTargetChoice = true
			m.CreateSelectedFolderPath = selectedFolderStorePath
			m.CreateSelectedFolderName = selectedFolderName
			if m.CreateSelectedFolderName == "" {
				m.CreateSelectedFolderName = filepath.Base(selectedFolderStorePath)
			}
			m.CreateTargetIsRoot = false
			m.CreateTargetFocus = false
		}
	}

	m.CreateInput.SetValue("")
	m.CreateInput.Placeholder = "Folder name"
	if mode == createRequest {
		m.CreateInput.Placeholder = "Request name"
	}
	inputWidth := m.Widht - 14
	if inputWidth < 12 {
		inputWidth = 12
	}
	m.CreateInput.SetWidth(inputWidth)
	return m.CreateInput.Focus()
}

var createRequestMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

func (m *Model) cycleCreateMethod(forward bool) {
	index := 0
	for i, method := range createRequestMethods {
		if method == m.CreateMethod {
			index = i
			break
		}
	}
	if forward {
		index = (index + 1) % len(createRequestMethods)
	} else {
		index = (index - 1 + len(createRequestMethods)) % len(createRequestMethods)
	}
	m.CreateMethod = createRequestMethods[index]
}

func (m *Model) cancelCreate() {
	m.CreateMode = createNone
	m.CreateParentPath = ""
	m.CreateMethodFocus = false
	m.CreateHasTargetChoice = false
	m.CreateTargetFocus = false
	m.CreateTargetIsRoot = false
	m.CreateSelectedFolderPath = ""
	m.CreateSelectedFolderName = ""
	m.CreateInput.Blur()
}

func (m *Model) saveCreate() {
	name := strings.TrimSpace(m.CreateInput.Value())
	if name == "" {
		m.StorageError = "Enter a name before saving"
		return
	}
	if m.CollectionStore == nil {
		m.CollectionStore = defaultCollectionStore()
	}

	var entry collectionstore.Entry
	var err error
	if m.CreateMode == createFolder {
		var parentPath string
		if m.CreateHasTargetChoice && !m.CreateTargetIsRoot {
			parentPath = m.CreateSelectedFolderPath
		} else {
			parentPath = ""
		}
		kind := "folder"
		if parentPath == "" {
			kind = "collection"
		}
		entry, err = m.CollectionStore.CreateFolder(parentPath, name, kind)
	} else if m.CreateMode == createRequest {
		entry, err = m.CollectionStore.CreateRequestWithMethod(m.CreateParentPath, name, m.CreateMethod)
	}
	if err != nil {
		m.StorageError = err.Error()
		return
	}

	newNode := treeNodeFromEntry(entry)
	if m.CreateMode == createFolder {
		newNode.Open = true
	}
	if m.CreateParentPath == "" && newNode.IsFolder && (!m.CreateHasTargetChoice || m.CreateTargetIsRoot) {
		m.Nodes = append(m.Nodes, newNode)
	} else {
		targetParent := m.CreateParentPath
		if m.CreateMode == createFolder && m.CreateHasTargetChoice {
			if m.CreateTargetIsRoot {
				targetParent = ""
			} else {
				targetParent = m.CreateSelectedFolderPath
			}
		}
		if targetParent == "" && newNode.IsFolder {
			m.Nodes = append(m.Nodes, newNode)
		} else if parent := findFolder(&m.Nodes, targetParent); parent != nil {
			parent.Children = append(parent.Children, newNode)
			parent.Open = true
		} else {
			m.StorageError = "The selected collection folder no longer exists"
			return
		}
	}

	m.StorageError = ""
	m.cancelCreate()
	m.selectZone(newNode.ZoneID)
}

func findFolder(nodes *[]TreeNode, storePath string) *TreeNode {
	if nodes == nil {
		return nil
	}
	cleanTarget := filepath.ToSlash(filepath.Clean(storePath))
	for i := range *nodes {
		node := &(*nodes)[i]
		if node.IsFolder && filepath.ToSlash(filepath.Clean(node.StorePath)) == cleanTarget {
			return node
		}
		if parent := findFolder(&node.Children, storePath); parent != nil {
			return parent
		}
	}
	return nil
}

// CreatePromptActive reports whether the sidebar is capturing create-dialog input.
