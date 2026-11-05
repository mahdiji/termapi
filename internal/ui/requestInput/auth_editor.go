package requestinput

import (
	"strings"

	"api/internal/collectionstore"
	"api/internal/ui/styles"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

var authTypes = []string{"none", "bearer", "basic", "api_key"}

func authTypeLabel(t string) string {
	switch t {
	case "none":
		return "None"
	case "bearer":
		return "Bearer"
	case "basic":
		return "Basic"
	case "api_key":
		return "API Key"
	}
	return t
}

type authField struct {
	key   string
	label string
}

func authFieldsFor(t string) []authField {
	switch t {
	case "bearer":
		return []authField{{"token", "Token"}}
	case "basic":
		return []authField{{"username", "Username"}, {"password", "Password"}}
	case "api_key":
		return []authField{{"key", "Key"}, {"value", "Value"}, {"add_to", "Add To"}}
	}
	return nil
}

// AuthEditor is a compact form for the selected authentication scheme.
type AuthEditor struct {
	Type    string
	values  map[string]string
	cursor  int
	Editing bool

	input  textinput.Model
	width  int
	height int
	orig   string
}

func NewAuthEditor() AuthEditor {
	in := textinput.New()
	in.Prompt = ""
	a := AuthEditor{
		Type:   "none",
		values: make(map[string]string),
		input:  in,
	}
	a.SetSize(70, 7)
	return a
}

func (a *AuthEditor) SetSize(width, height int) {
	a.width = max(width, 20)
	a.height = max(height, 3)
	a.input.SetWidth(max(a.width-16, 10))
}

func (a *AuthEditor) Focus() tea.Cmd { return nil }

func (a *AuthEditor) Blur() {
	a.Editing = false
	a.input.Blur()
}

func (a *AuthEditor) fields() []authField { return authFieldsFor(a.Type) }

func (a *AuthEditor) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	if a.Editing {
		switch key.String() {
		case "esc":
			a.input.SetValue(a.orig)
			a.commit()
			return nil
		case "enter":
			a.commit()
			return nil
		case "tab":
			a.commit()
			a.move(1)
			return nil
		case "shift+tab":
			a.commit()
			a.move(-1)
			return nil
		}
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(key)
		return cmd
	}

	fields := a.fields()
	switch key.String() {
	case "t":
		a.cycleType()
	case "up", "k":
		a.move(-1)
	case "down", "j":
		a.move(1)
	case "enter", "e":
		if len(fields) > 0 {
			a.startEditing()
		}
	case "tab":
		a.move(1)
	case "shift+tab":
		a.move(-1)
	case "left", "right", "space", " ":
		if a.cursor >= 0 && a.cursor < len(fields) && fields[a.cursor].key == "add_to" {
			if a.values["add_to"] == "query" {
				a.values["add_to"] = "header"
			} else {
				a.values["add_to"] = "query"
			}
		}
	}
	_ = fields
	return nil
}

func (a *AuthEditor) cycleType() {
	a.commitIfEditing()
	idx := 0
	for i, t := range authTypes {
		if t == a.Type {
			idx = i
			break
		}
	}
	a.Type = authTypes[(idx+1)%len(authTypes)]
	a.cursor = 0
	a.Editing = false
	a.input.Blur()
	if a.values["add_to"] == "" {
		a.values["add_to"] = "header"
	}
}

func (a *AuthEditor) SetType(t string) {
	a.commitIfEditing()
	for _, known := range authTypes {
		if known == t {
			a.Type = t
			a.cursor = 0
			a.Editing = false
			a.input.Blur()
			return
		}
	}
	a.Type = "none"
}

func (a *AuthEditor) move(delta int) {
	fields := a.fields()
	if len(fields) == 0 {
		return
	}
	a.cursor = (a.cursor + delta + len(fields)) % len(fields)
	a.Editing = false
	a.input.Blur()
}

func (a *AuthEditor) startEditing() {
	fields := a.fields()
	if a.cursor < 0 || a.cursor >= len(fields) {
		return
	}
	field := fields[a.cursor]
	if field.key == "add_to" {
		return
	}
	a.orig = a.values[field.key]
	a.input.SetValue(a.orig)
	a.input.CursorEnd()
	a.input.Focus()
	a.Editing = true
}

func (a *AuthEditor) commitIfEditing() {
	if a.Editing {
		a.commit()
	}
}

func (a *AuthEditor) commit() {
	fields := a.fields()
	if a.cursor >= 0 && a.cursor < len(fields) {
		field := fields[a.cursor]
		a.values[field.key] = strings.TrimSpace(a.input.Value())
	}
	a.Editing = false
	a.input.Blur()
}

func (a *AuthEditor) View() string {
	var pills []string
	for _, t := range authTypes {
		label := " " + authTypeLabel(t) + " "
		style := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
		if t == a.Type {
			style = lipgloss.NewStyle().
				Foreground(styles.ColBackground).
				Background(styles.ColAccent).
				Bold(true)
		}
		pills = append(pills, zone.Mark("auth_type_"+t, style.Render(label)))
	}
	typeLine := strings.Join(pills, " ")
	if lipgloss.Width(typeLine) < a.width-24 {
		typeLine += "  " + styles.Dim().Render("t ganti tipe")
	}

	lines := []string{typeLine}
	fields := a.fields()
	if len(fields) == 0 {
		lines = append(lines, styles.Dim().Render("  Tidak ada autentikasi."))
	}
	for i, field := range fields {
		label := fitCell(field.label, 12)
		value := a.values[field.key]
		if field.key == "add_to" {
			if value == "" {
				value = "header"
			}
			cell := fitCell(value, max(a.width-18, 10))
			if i == a.cursor {
				cell = styles.Selected().Render(cell)
			}
			lines = append(lines, "  "+styles.Dim().Render(label)+" "+cell)
			continue
		}
		if field.key == "password" && !(i == a.cursor && a.Editing) && value != "" {
			value = strings.Repeat("•", len([]rune(value)))
		}
		var cell string
		if i == a.cursor && a.Editing {
			cell = fitCell(a.input.View(), max(a.width-18, 10))
		} else {
			cell = fitCell(value, max(a.width-18, 10))
			if i == a.cursor {
				cell = styles.Selected().Render(cell)
			}
		}
		lines = append(lines, "  "+styles.Dim().Render(label)+" "+cell)
	}
	if len(fields) > 0 && !a.Editing {
		lines = append(lines, styles.Dim().Render("  ↑/↓ pilih · Enter edit · Tab field berikutnya"))
	}
	for len(lines) < a.height {
		lines = append(lines, "")
	}
	if len(lines) > a.height {
		lines = lines[:a.height]
	}
	return strings.Join(lines, "\n")
}

func (a *AuthEditor) Spec() collectionstore.AuthSpec {
	return collectionstore.AuthSpec{
		Type:     a.Type,
		Token:    a.values["token"],
		Username: a.values["username"],
		Password: a.values["password"],
		Key:      a.values["key"],
		Value:    a.values["value"],
		AddTo:    a.values["add_to"],
	}
}

func (a *AuthEditor) SetSpec(spec collectionstore.AuthSpec) {
	a.values = map[string]string{
		"token":    spec.Token,
		"username": spec.Username,
		"password": spec.Password,
		"key":      spec.Key,
		"value":    spec.Value,
		"add_to":   spec.AddTo,
	}
	if a.values["add_to"] == "" {
		a.values["add_to"] = "header"
	}
	a.SetType(spec.Type)
}
