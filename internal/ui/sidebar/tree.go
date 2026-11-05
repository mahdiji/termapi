package sidebar

import (
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"api/internal/collectionstore"
	"api/internal/ui/styles"
	"charm.land/lipgloss/v2"
)

func treeNodesFromEntries(entries []collectionstore.Entry) []TreeNode {
	nodes := make([]TreeNode, 0, len(entries))
	for _, entry := range entries {
		nodes = append(nodes, treeNodeFromEntry(entry))
	}
	return nodes
}

func treeNodeFromEntry(entry collectionstore.Entry) TreeNode {
	zonePrefix := "folder_"
	node := TreeNode{
		Name:      entry.Name,
		IsFolder:  entry.IsFolder,
		ZoneID:    zonePrefix + strings.ReplaceAll(filepath.ToSlash(entry.RelPath), "/", "__"),
		StorePath: entry.RelPath,
	}
	if entry.Request != nil {
		node.RequestName = entry.Request.Name
		node.Method = strings.ToUpper(entry.Request.Method)
		node.RequestURL = entry.Request.URL
		route := requestRoute(entry.Request.URL)
		if entry.Request.URL == "" {
			route = entry.Request.Name
		}
		node.Name = node.Method + " " + route
		node.ZoneID = "file_" + strings.ReplaceAll(filepath.ToSlash(entry.RelPath), "/", "__")
	}
	node.Children = treeNodesFromEntries(entry.Children)
	return node
}

func requestRoute(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Path != "" {
		route := parsed.EscapedPath()
		if parsed.RawQuery != "" {
			route += "?" + parsed.RawQuery
		}
		return route
	}
	if rawURL == "" {
		return "/"
	}
	return rawURL
}

func findTreeNode(node TreeNode, zoneID string) *TreeNode {
	if node.ZoneID == zoneID {
		return &node
	}
	for _, child := range node.Children {
		if found := findTreeNode(child, zoneID); found != nil {
			return found
		}
	}
	return nil
}

func getStyledItem(fgHex string, isSelected bool) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(fgHex))
	if isSelected {
		return style.Bold(true)
	}
	return style
}

func fileMethod(name string) string {
	method, _, hasPath := strings.Cut(name, " ")
	if !hasPath {
		return ""
	}

	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		return method
	default:
		return ""
	}
}

// appendTreeItems membangun item tree dengan indentasi 2 spasi per level,
// chevron untuk folder, dan method badge 3-huruf untuk request — tanpa garis
// ASCII. Method dipotong ke 3 huruf agar kolom path rata seperti Bruno/Insomnia.
func shortMethod(m string) string {
	switch m {
	case "GET":
		return "GET"
	case "POST":
		return "POST"
	case "PUT":
		return "PUT"
	case "DELETE":
		return "DEL"
	case "PATCH":
		return "PTCH"
	case "HEAD":
		return "HEAD"
	case "OPTIONS":
		return "OPT"
	default:
		if len(m) > 4 {
			return m[:4]
		}
		return m
	}
}

func requestCount(nodes []TreeNode) int {
	count := 0
	for _, node := range nodes {
		if node.IsFolder {
			count += requestCount(node.Children)
		} else {
			count++
		}
	}
	return count
}

// requestFlatItem menyusun satu baris request. pinned menambahkan penanda
// bintang di depan badge method.
func requestFlatItem(node *TreeNode, indent string, pinned, selected bool) flatItem {
	method := fileMethod(node.Name)
	prefix := indent + "  "
	if pinned {
		prefix = indent + "★ "
	}
	displayName := indent + "  " + styles.IconFile + " " + node.Name
	if method != "" {
		_, path, _ := strings.Cut(node.Name, " ")
		displayName = prefix + shortMethod(method)
		if node.RequestName != "" {
			displayName += "  " + node.RequestName
		}
		if path != "" && path != node.RequestName {
			displayName += "  ·  " + path
		}
	}
	item := flatItem{
		ZoneID:  node.ZoneID,
		Text:    displayName,
		Style:   lipgloss.NewStyle().Foreground(styles.ColTextMuted).Bold(selected),
		Type:    itemTypeFile,
		NodePtr: node,
	}
	if method != "" {
		item.AccentText = shortMethod(method)
		item.AccentStyle = lipgloss.NewStyle().
			Foreground(styles.MethodColor(method)).
			Bold(true)
	}
	return item
}

// pinnedZoneID memberi zone unik untuk baris di section Pinned agar tidak
// bertabrakan dengan baris yang sama di tree Collections.
func pinnedZoneID(path string) string {
	return "pin_" + strings.ReplaceAll(filepath.ToSlash(path), "/", "__")
}

func (m *Model) isPinned(path string) bool {
	for _, pinned := range m.Pinned {
		if pinned == path {
			return true
		}
	}
	return false
}

func findNodeByPath(nodes []TreeNode, path string) *TreeNode {
	for i := range nodes {
		if !nodes[i].IsFolder && nodes[i].StorePath == path {
			return &nodes[i]
		}
		if found := findNodeByPath(nodes[i].Children, path); found != nil {
			return found
		}
	}
	return nil
}

// pinnedNodes mengembalikan node request yang dipin sesuai urutan pin.
// Path yang filenya sudah tidak ada akan dilewati.
func (m *Model) pinnedNodes() []*TreeNode {
	nodes := make([]*TreeNode, 0, len(m.Pinned))
	for _, path := range m.Pinned {
		if node := findNodeByPath(m.Nodes, path); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (m *Model) appendTreeItems(items *[]flatItem, nodes []TreeNode, depth int) {
	for i := range nodes {
		node := &nodes[i]
		indent := strings.Repeat("  ", depth)

		if node.IsFolder {
			chevron := styles.IconChevronRight
			folderIcon := styles.IconFolderClosed
			if node.Open {
				chevron = styles.IconChevronDown
				folderIcon = styles.IconFolderOpen
			}
			label := indent + chevron + " " + folderIcon + " " + node.Name
			if count := requestCount(node.Children); count > 0 {
				label += "  ·  " + strconv.Itoa(count)
			}
			*items = append(*items, flatItem{
				ZoneID:  node.ZoneID,
				Text:    label,
				Style:   lipgloss.NewStyle().Foreground(styles.ColTextBright).Bold(true),
				Type:    itemTypeFolder,
				NodePtr: node,
			})

			if !node.Open {
				continue
			}
			m.appendTreeItems(items, node.Children, depth+1)
			continue
		}

		*items = append(*items, requestFlatItem(node, indent, m.isPinned(node.StorePath), node.ZoneID == m.SelectedZoneID))
	}
}

func (m *Model) getFlatItems() []flatItem {
	items := make([]flatItem, 0)

	accordionStyle := func(zoneID string) lipgloss.Style {
		if m.SelectedZoneID == zoneID {
			return lipgloss.NewStyle().
				Foreground(styles.ColAccent).
				Bold(true)
		}
		return lipgloss.NewStyle().Foreground(styles.ColTextBright).Bold(true)
	}

	if pinned := m.pinnedNodes(); len(pinned) > 0 {
		chevron := styles.IconChevronRight
		if m.PinnedOpen {
			chevron = styles.IconChevronDown
		}
		items = append(items, flatItem{
			ZoneID: "accordion_pinned",
			Text:   chevron + " Pinned  ·  " + strconv.Itoa(len(pinned)),
			Style:  accordionStyle("accordion_pinned"),
			Type:   itemTypeAccordion,
		})
		if m.PinnedOpen {
			for _, node := range pinned {
				zoneID := pinnedZoneID(node.StorePath)
				item := requestFlatItem(node, "  ", true, zoneID == m.SelectedZoneID)
				item.ZoneID = zoneID
				items = append(items, item)
			}
		}
	}

	collectionsChevron := styles.IconChevronRight
	if m.CollectionsOpen {
		collectionsChevron = styles.IconChevronDown
	}
	collectionTitle := collectionsChevron + " Collections"
	if requests := requestCount(m.Nodes); requests > 0 {
		collectionTitle += "  ·  " + strconv.Itoa(requests) + " req"
	}
	items = append(items, flatItem{
		ZoneID: "accordion_collections",
		Text:   collectionTitle,
		Style:  accordionStyle("accordion_collections"),
		Type:   itemTypeAccordion,
	})
	if m.CollectionsOpen {
		m.appendTreeItems(&items, m.Nodes, 0)
		if m.StorageError != "" {
			items = append(items, flatItem{
				ZoneID: "collections_storage_notice",
				Text:   "  ! Collection storage error",
				Style:  lipgloss.NewStyle().Foreground(styles.ColError),
				Type:   itemTypeNotice,
			})
		}
	}

	if query := strings.ToLower(strings.TrimSpace(m.Filter)); query != "" {
		filtered := items[:0]
		var startsWith []flatItem
		var contains []flatItem
		for _, item := range items {
			if item.Type == itemTypeAccordion {
				filtered = append(filtered, item)
				continue
			}
			cleanText := strings.ToLower(strings.TrimSpace(item.Text))
			// Hilangkan ikon/chevron dari teks agar matching bersih.
			for _, strip := range []string{
				styles.IconChevronRight, styles.IconChevronDown,
				styles.IconFolderClosed, styles.IconFolderOpen, styles.IconFile,
			} {
				cleanText = strings.ReplaceAll(cleanText, strip, "")
			}
			cleanText = strings.TrimSpace(cleanText)
			if strings.HasPrefix(cleanText, query) {
				startsWith = append(startsWith, item)
			} else if strings.Contains(cleanText, query) || strings.Contains(strings.ToLower(item.Text), query) {
				contains = append(contains, item)
			}
		}
		filtered = append(filtered, startsWith...)
		filtered = append(filtered, contains...)
		items = filtered
	}
	return items
}

// SetFilter applies the global search query to sidebar items.
func (m *Model) SetFilter(query string) {
	m.Filter = query
	m.CursorIdx = 0
	m.ScrollY = 0
	items := m.getFlatItems()
	if len(items) > 0 {
		m.SelectedZoneID = items[0].ZoneID
	}
}
