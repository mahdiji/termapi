package envscreen

import (
	"sort"

	"api/internal/envstore"

	"charm.land/bubbles/v2/textinput"
)

// Model is the fullscreen environment-variables manager.
type Model struct {
	Width  int
	Height int

	Store     *envstore.Store
	Envs      []string          // list of env names
	ActiveEnv string            // currently active env name
	Current   *envstore.EnvFile // env shown in right panel

	// Left panel
	EnvCursor int

	// Right panel
	VarKeys   []string // sorted keys of Current.Variables
	VarCursor int

	FocusPane int // 0 = left (env list), 1 = right (var list)

	// Input state — adding new env
	Adding    bool
	InputName textinput.Model

	// Input state — adding / editing var
	AddingVar  bool
	EditingVar bool
	InputKey   textinput.Model
	InputVal   textinput.Model
	InputFocus int // 0 = key field, 1 = val field

	Err     string
	Message string
}

// New creates a new env-screen model.
func New(store *envstore.Store, activeEnv string) *Model {
	iname := textinput.New()
	iname.Prompt = ""
	iname.Placeholder = "environment name"

	ikey := textinput.New()
	ikey.Prompt = ""
	ikey.Placeholder = "KEY"

	ival := textinput.New()
	ival.Prompt = ""
	ival.Placeholder = "value"

	return &Model{
		Store:     store,
		ActiveEnv: activeEnv,
		InputName: iname,
		InputKey:  ikey,
		InputVal:  ival,
	}
}

// SetSize updates screen dimensions.
func (m *Model) SetSize(w, h int) {
	m.Width = w
	m.Height = h
}

// Load reloads the environment list from disk and refreshes the right panel.
func (m *Model) Load() {
	if m.Store == nil {
		return
	}
	names, err := m.Store.List()
	if err != nil {
		m.Err = err.Error()
		return
	}
	m.Envs = names
	if m.EnvCursor >= len(m.Envs) {
		m.EnvCursor = max(len(m.Envs)-1, 0)
	}
	m.refreshRight()
}

// refreshRight loads the env that the left panel cursor points to.
func (m *Model) refreshRight() {
	if len(m.Envs) == 0 || m.Store == nil {
		m.Current = nil
		m.VarKeys = nil
		return
	}
	name := m.Envs[m.EnvCursor]
	ef, err := m.Store.Load(name)
	if err != nil {
		m.Err = err.Error()
		return
	}
	m.Current = ef
	m.rebuildVarKeys()
	if m.VarCursor >= len(m.VarKeys) {
		m.VarCursor = max(len(m.VarKeys)-1, 0)
	}
}

func (m *Model) rebuildVarKeys() {
	if m.Current == nil {
		m.VarKeys = nil
		return
	}
	keys := make([]string, 0, len(m.Current.Variables))
	for k := range m.Current.Variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	m.VarKeys = keys
}

// activeVars returns the vars map of the currently active env.
func (m *Model) activeVars() map[string]string {
	if m.ActiveEnv == "" || m.Store == nil {
		return nil
	}
	ef, err := m.Store.Load(m.ActiveEnv)
	if err != nil {
		return nil
	}
	return ef.Variables
}

// Update handles keyboard events. Returns:
//   - active: false → caller should close the screen.
//   - newActiveEnv: currently active env name.
//   - vars: variables of the active env.
