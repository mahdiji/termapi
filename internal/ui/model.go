package ui

import (
	"api/internal/config"
	"api/internal/envstore"
	"api/internal/historydb"
	"api/internal/model"
	"api/internal/ui/diffscreen"
	"api/internal/ui/envscreen"
	"api/internal/ui/helpscreen"
	"api/internal/ui/historyscreen"
	"api/internal/ui/newscreen"
	requestinput "api/internal/ui/requestInput"
	"api/internal/ui/scopescreen"
	"api/internal/ui/sidebar"
	"api/internal/ui/styles"
	"api/internal/ui/wirescreen"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type AppModel struct {
	State               *model.SharedState
	ReqState            *requestinput.Model
	Sidebar             *sidebar.Model
	Response            *requestinput.ResponsePanel
	HistoryStore        *historydb.Store
	HistoryScreen       *historyscreen.Model
	HistoryMode         bool
	ScopeScreen         *scopescreen.Model
	ScopeMode           bool
	EnvStore            *envstore.Store
	EnvScreen           *envscreen.Model
	EnvMode             bool
	DiffScreen          *diffscreen.Model
	DiffMode            bool
	WireScreen          *wirescreen.Model
	WireMode            bool
	HelpScreen          *helpscreen.Model
	HelpMode            bool
	NewItemScreen       *newscreen.Model
	NewItemMode         bool
	BatchChoosing       bool
	BatchMode           bool
	BatchFolder         string
	BatchDepth          int
	BatchMaxDepth       int
	BatchExpected       int
	BatchResults        []requestinput.ResponseMsg
	BatchNames          []string
	BatchSpecs          []requestinput.BatchSpec
	BatchVariables      map[string]string
	BatchNext           int
	BatchChain          bool
	BatchRetries        int
	BatchSelected       int
	BatchError          string
	SaveDialog          bool
	SaveName            textinput.Model
	SaveSearch          textinput.Model
	SaveFolders         []saveDestination
	SaveVisible         []saveDestination
	SaveFocus           int
	SaveSelected        int
	SaveScroll          int
	SaveError           string
	SearchMode          bool
	SearchInput         textinput.Model
	SearchHistory       bool
	SearchSelectedIndex int
	ResponseRatio       float64
	ScreenWidth         int
	ScreenHeight        int
	// Layout dimensions computed once per resize; single source of truth.
	ContentWidth   int
	MainHeight     int
	RequestHeight  int
	ResponseHeight int
}

func NewAppModel(state *model.SharedState) AppModel {
	if state != nil && state.GlobalScope == "" {
		if scope, err := config.LoadGlobalScope(""); err == nil {
			state.GlobalScope = scope
		}
	}
	nameInput := textinput.New()
	nameInput.Prompt = ""
	nameInput.Placeholder = "Request name"
	saveSearch := textinput.New()
	saveSearch.Prompt = ""
	saveSearch.Placeholder = "Cari nama atau path folder..."
	saveSearch.CharLimit = 128
	saveInputStyles := textinput.Styles{
		Focused: textinput.StyleState{
			Text:        styles.Text(),
			Placeholder: styles.Dim(),
			Suggestion:  styles.Muted(),
			Prompt:      lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true),
		},
		Blurred: textinput.StyleState{
			Text:        styles.Text(),
			Placeholder: styles.Dim(),
			Suggestion:  styles.Muted(),
			Prompt:      lipgloss.NewStyle().Foreground(styles.ColTextMuted).Bold(true),
		},
		Cursor: textinput.CursorStyle{
			Color: styles.ColAccent,
			Shape: tea.CursorBar,
			Blink: true,
		},
	}
	nameInput.SetStyles(saveInputStyles)
	saveSearch.SetStyles(saveInputStyles)
	searchInput := textinput.New()
	searchInput.Prompt = "Search files: "
	searchInput.Placeholder = "name, path, method, URL..."
	searchInput.SetStyles(textinput.Styles{
		Focused: textinput.StyleState{
			Text:        styles.Text(),
			Placeholder: styles.Dim(),
			Suggestion:  styles.Muted(),
			Prompt:      lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true),
		},
		Blurred: textinput.StyleState{
			Text:        styles.Text(),
			Placeholder: styles.Dim(),
			Suggestion:  styles.Muted(),
			Prompt:      lipgloss.NewStyle().Foreground(styles.ColTextMuted).Bold(true),
		},
		Cursor: textinput.CursorStyle{
			Color: styles.ColAccent,
			Shape: tea.CursorBar,
			Blink: true,
		},
	})
	store, err := historydb.OpenProject()
	screen := &historyscreen.Model{}
	if err != nil {
		screen.Err = err.Error()
	}
	scopeScreen := &scopescreen.Model{}
	envStore := envstore.New(".")
	activeEnv := ""
	if names, err := envStore.List(); err == nil && len(names) > 0 {
		activeEnv = names[0]
		if ef, err := envStore.Load(activeEnv); err == nil {
			state.ActiveEnv = activeEnv
			state.EnvVars = ef.Variables
		}
	}
	envScreen := envscreen.New(envStore, activeEnv)
	helpScreen := helpscreen.NewModel()
	newItemScreen := newscreen.New()
	return AppModel{
		State:         state,
		ReqState:      requestinput.New(state),
		Sidebar:       sidebar.New(state),
		Response:      requestinput.NewResponsePanel(),
		HistoryStore:  store,
		HistoryScreen: screen,
		ScopeScreen:   scopeScreen,
		EnvStore:      envStore,
		EnvScreen:     envScreen,
		DiffScreen:    diffscreen.NewModel("", ""),
		HelpScreen:    helpScreen,
		NewItemScreen: newItemScreen,
		SaveName:      nameInput,
		SaveSearch:    saveSearch,
		SearchInput:   searchInput,
		ResponseRatio: config.LoadResponseRatio(""),
	}
}
