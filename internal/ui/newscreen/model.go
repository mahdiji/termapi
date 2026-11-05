package newscreen

import (
	"errors"
	"strings"

	"api/internal/collectionstore"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// Kind is the type of item created by the screen.
type Kind int

const (
	KindRequest Kind = iota
	KindFolder
)

// Focus identifies which control owns keyboard input.
type Focus int

const (
	FocusName Focus = iota
	FocusMethod
	FocusSearch
	FocusList
)

// Action is returned by Update to tell the parent what happened.
type Action int

const (
	ActionNone Action = iota
	ActionClose
	ActionSubmit
)

// Destination is a folder that can receive the new item.
type Destination struct {
	Path   string // collection-relative path; empty means the collections root
	Name   string
	Depth  int
	IsRoot bool
}

// Model is the fullscreen "new request / new folder" composer.
type Model struct {
	Width  int
	Height int

	Store *collectionstore.Store

	Kind     Kind
	Method   string
	Methods  []string
	Focus    Focus
	Name     textinput.Model
	Search   textinput.Model
	Selected int
	Scroll   int
	Err      string

	folders   []Destination
	visible   []Destination
	preferred string
	entries   []collectionstore.Entry
}

// New returns an empty composer. Call Open before showing it.
func New() *Model {
	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "Name"

	search := textinput.New()
	search.Prompt = ""
	search.Placeholder = "Search folders..."
	search.CharLimit = 128

	return &Model{
		Method:  "GET",
		Methods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"},
		Name:    name,
		Search:  search,
	}
}

// SetStore attaches the collection store used to list destinations and persist.
func (m *Model) SetStore(store *collectionstore.Store) {
	if m == nil {
		return
	}
	m.Store = store
}

// Open resets the form for a new item of the given kind.
func (m *Model) Open(kind Kind, preferredPath string) {
	if m == nil {
		return
	}
	m.Kind = kind
	m.preferred = strings.TrimSpace(preferredPath)
	m.Method = "GET"
	m.Name.SetValue("")
	m.Search.SetValue("")
	m.Err = ""
	m.Focus = FocusName
	m.reloadFolders()
	m.visible = nil
	m.Selected = -1
	m.rebuild(true)
	m.Name.Focus()
	m.Search.Blur()
}

// SetSize updates the available screen area.
func (m *Model) SetSize(width, height int) {
	if m == nil {
		return
	}
	m.Width = width
	m.Height = height
	inputWidth := m.contentWidth() - 12
	if inputWidth < 12 {
		inputWidth = 12
	}
	m.Name.SetWidth(inputWidth)
	m.Search.SetWidth(max(inputWidth-4, 8))
}

// Update handles one key event and reports the resulting action.
func (m *Model) Update(msg tea.KeyMsg) Action {
	if m == nil {
		return ActionNone
	}
	key := msg.String()

	switch key {
	case "esc", "ctrl+c":
		return ActionClose
	case "ctrl+t":
		if m.Kind == KindRequest {
			m.Kind = KindFolder
		} else {
			m.Kind = KindRequest
		}
		m.Err = ""
		m.rebuild(false)
		m.focusForKind()
		return ActionNone
	case "ctrl+f":
		m.setFocus(FocusSearch)
		return ActionNone
	case "/":
		if m.Focus == FocusList || m.Focus == FocusMethod {
			m.setFocus(FocusSearch)
			return ActionNone
		}
	}

	switch m.Focus {
	case FocusName:
		switch key {
		case "tab":
			m.cycleFocus(1)
		case "shift+tab":
			m.cycleFocus(-1)
		case "enter":
			return ActionSubmit
		default:
			m.Name, _ = m.Name.Update(msg)
			m.Err = ""
		}
	case FocusMethod:
		switch key {
		case "tab":
			m.cycleFocus(1)
		case "shift+tab":
			m.cycleFocus(-1)
		case "left", "up", "h":
			m.cycleMethod(-1)
		case "right", "down", "l", "space", " ":
			m.cycleMethod(1)
		case "enter":
			return ActionSubmit
		}
	case FocusSearch:
		switch key {
		case "tab":
			m.cycleFocus(1)
		case "shift+tab":
			m.cycleFocus(-1)
		case "enter":
			m.setFocus(FocusList)
		case "esc":
			return ActionClose
		default:
			m.Search, _ = m.Search.Update(msg)
			m.rebuild(false)
			m.Err = ""
		}
	case FocusList:
		switch key {
		case "tab":
			m.cycleFocus(1)
		case "shift+tab":
			m.cycleFocus(-1)
		case "up", "k":
			m.move(-1)
		case "down", "j":
			m.move(1)
		case "pgup":
			m.move(-m.listRows())
		case "pgdown":
			m.move(m.listRows())
		case "home":
			m.moveTo(0)
		case "end":
			m.moveTo(len(m.visible) - 1)
		case "enter":
			return ActionSubmit
		case "m":
			m.cycleMethod(1)
		}
	}
	return ActionNone
}

// Submit creates the selected item and returns its entry.
func (m *Model) Submit() (collectionstore.Entry, error) {
	if m == nil || m.Store == nil {
		return collectionstore.Entry{}, errors.New("collection store is not available")
	}
	name := strings.TrimSpace(m.Name.Value())
	if name == "" {
		return collectionstore.Entry{}, errors.New("nama tidak boleh kosong")
	}

	dest, ok := m.selectedDestination()
	if m.Kind == KindRequest {
		if !ok {
			return collectionstore.Entry{}, errors.New("belum ada folder tujuan — buat folder dulu")
		}
		return m.Store.CreateRequestWithMethod(dest.Path, name, m.Method)
	}

	parent := ""
	kind := "collection"
	if ok && !dest.IsRoot {
		parent = dest.Path
		kind = "folder"
	}
	return m.Store.CreateFolder(parent, name, kind)
}

// SelectedDestinationPath returns the currently highlighted folder path.
func (m *Model) SelectedDestinationPath() string {
	dest, ok := m.selectedDestination()
	if !ok {
		return ""
	}
	return dest.Path
}

// SelectedDestinationName returns a human readable label for the header.
func (m *Model) SelectedDestinationName() string {
	dest, ok := m.selectedDestination()
	if !ok {
		return "(none)"
	}
	if dest.IsRoot {
		return "collections/"
	}
	if dest.Path != "" {
		return dest.Path
	}
	return dest.Name
}

func (m *Model) reloadFolders() {
	m.entries = nil
	if m.Store == nil {
		return
	}
	entries, err := m.Store.Load()
	if err != nil {
		m.Err = err.Error()
		return
	}
	m.entries = entries
}

func (m *Model) rebuild(usePreferred bool) {
	previous, hadPrevious := m.selectedDestination()
	query := strings.ToLower(strings.TrimSpace(m.Search.Value()))

	m.folders = m.folders[:0]
	if m.Kind == KindFolder {
		m.folders = append(m.folders, Destination{Name: "collections/", IsRoot: true})
	}
	baseDepth := 0
	if m.Kind == KindFolder {
		baseDepth = 1
	}
	m.flatten(m.entries, baseDepth, &m.folders)

	m.visible = m.visible[:0]
	for _, dest := range m.folders {
		if query == "" || strings.Contains(strings.ToLower(dest.Name+" "+dest.Path), query) {
			m.visible = append(m.visible, dest)
		}
	}

	target := m.preferred
	if hadPrevious && !usePreferred {
		target = previous.Path
	}

	m.Selected = 0
	found := false
	for i, dest := range m.visible {
		if dest.Path == target {
			m.Selected = i
			found = true
			break
		}
	}
	if !found && m.Kind == KindRequest {
		for i, dest := range m.visible {
			if !dest.IsRoot {
				m.Selected = i
				break
			}
		}
	}
	m.clampScroll()
}

func (m *Model) flatten(entries []collectionstore.Entry, depth int, out *[]Destination) {
	for _, entry := range entries {
		if !entry.IsFolder {
			continue
		}
		*out = append(*out, Destination{Path: entry.RelPath, Name: entry.Name, Depth: depth})
		m.flatten(entry.Children, depth+1, out)
	}
}

func (m *Model) selectedDestination() (Destination, bool) {
	if m.Selected < 0 || m.Selected >= len(m.visible) {
		return Destination{}, false
	}
	return m.visible[m.Selected], true
}

func (m *Model) move(delta int) {
	if len(m.visible) == 0 {
		return
	}
	m.moveTo(m.Selected + delta)
}

func (m *Model) moveTo(index int) {
	if len(m.visible) == 0 {
		return
	}
	m.Selected = clamp(index, 0, len(m.visible)-1)
	m.clampScroll()
}

func (m *Model) clampScroll() {
	rows := m.listRows()
	if rows <= 0 {
		m.Scroll = 0
		return
	}
	if m.Selected < m.Scroll {
		m.Scroll = m.Selected
	}
	if m.Selected >= m.Scroll+rows {
		m.Scroll = m.Selected - rows + 1
	}
	maxScroll := max(len(m.visible)-rows, 0)
	m.Scroll = clamp(m.Scroll, 0, maxScroll)
}

func (m *Model) listRows() int {
	rows := m.Height - 21
	return clamp(rows, 3, 12)
}

func (m *Model) contentWidth() int {
	return clamp(m.Width-8, 52, 76)
}

func (m *Model) cycleMethod(delta int) {
	index := 0
	for i, method := range m.Methods {
		if method == m.Method {
			index = i
			break
		}
	}
	index = (index + delta + len(m.Methods)) % len(m.Methods)
	m.Method = m.Methods[index]
}

func (m *Model) cycleFocus(dir int) {
	order := m.focusOrder()
	index := 0
	for i, focus := range order {
		if focus == m.Focus {
			index = i
			break
		}
	}
	index = (index + dir + len(order)) % len(order)
	m.setFocus(order[index])
}

func (m *Model) focusOrder() []Focus {
	order := []Focus{FocusName}
	if m.Kind == KindRequest {
		order = append(order, FocusMethod)
	}
	return append(order, FocusSearch, FocusList)
}

func (m *Model) focusForKind() {
	valid := false
	for _, focus := range m.focusOrder() {
		if focus == m.Focus {
			valid = true
			break
		}
	}
	if !valid {
		m.setFocus(FocusName)
	}
}

func (m *Model) setFocus(focus Focus) {
	m.Focus = focus
	m.Name.Blur()
	m.Search.Blur()
	switch focus {
	case FocusName:
		m.Name.Focus()
	case FocusSearch:
		m.Search.Focus()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
