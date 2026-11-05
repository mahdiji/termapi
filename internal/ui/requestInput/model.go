package requestinput

import (
	"context"

	"api/internal/model"
	"api/internal/ui/styles"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	State           *model.SharedState
	Input           textinput.Model
	Method          string
	Focused         bool
	Methods         []string
	FocusArea       int // 0: Method, 1: URL, 2: Send, 3: request editor
	Tab             int
	Params          KVEditor
	Headers         KVEditor
	Body            BodyEditor
	Auth            AuthEditor
	Cookies         KVEditor
	CollectionName  string
	CollectionPath  string
	RequestFilePath string
	RequestName     string
	EditorHeight    int
	lastWidth       int
	isClickLocked   bool
	cancel          context.CancelFunc
}

func New(state *model.SharedState) *Model {
	ti := textinput.New()
	ti.Placeholder = "Enter a URL..."
	ti.Prompt = ""
	params := NewKVEditor("params", "params", true, true)
	headers := NewKVEditor("headers", "headers", true, true)
	cookies := NewKVEditor("cookies", "params", false, false)
	body := NewBodyEditor()
	auth := NewAuthEditor()

	m := &Model{
		State:        state,
		Input:        ti,
		Method:       "GET",
		Focused:      true,
		Methods:      []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"},
		FocusArea:    1, // 1 untuk fokus ke URL Input di awal
		Params:       params,
		Headers:      headers,
		Body:         body,
		Auth:         auth,
		Cookies:      cookies,
		EditorHeight: 8,
	}

	// Gunakan fallback default yang sangat aman (misal 50) agar tidak jebol sebelum di-resize
	targetWidth := 50
	if state != nil && state.Widht > 0 {
		targetWidth = state.Widht
	}
	m.SetWidth(targetWidth)
	m.Input.Focus()

	return m
}

func (m *Model) SetWidth(w int) {
	if m == nil {
		return
	}
	m.lastWidth = w

	// Hitung lebar dari komponen visual dan frame Lipgloss agar kotak request
	// tepat sama lebar dengan editor serta panel RESPONSE di bawahnya.
	// Lebar harus mengikuti komponen yang benar-benar dirender di View().
	methodName := m.Method
	if methodName == "" {
		methodName = "GET"
	}
	methodWidth := lipgloss.Width(methodName) + 2 // padding horizontal pill
	methodWidth += 4                              // ‹ + spasi + spasi + ›
	sepWidth := lipgloss.Width(" │ ")
	buttonWidth := lipgloss.Width(styles.IconSend+" Send") + 2 // padding tombol
	fixedWidth := methodWidth +
		2*sepWidth +
		buttonWidth +
		styles.BaseBoxStyle.GetHorizontalFrameSize()
	// Bubbles merender satu sel ekstra untuk virtual cursor di View().
	targetWidth := max(w-fixedWidth-1, 1)
	m.Input.SetWidth(targetWidth)
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) getCurrentMethodIndex() int {
	for i, meth := range m.Methods {
		if meth == m.Method {
			return i
		}
	}
	return 0
}
