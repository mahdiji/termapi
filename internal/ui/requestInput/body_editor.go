package requestinput

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"api/internal/collectionstore"
	"api/internal/ui/styles"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

var bodyTypes = []string{"none", "json", "xml", "text", "form", "multipart", "binary", "graphql"}

func bodyTypeLabel(t string) string {
	switch t {
	case "none":
		return "None"
	case "json":
		return "JSON"
	case "xml":
		return "XML"
	case "text":
		return "Text"
	case "form":
		return "Form"
	case "multipart":
		return "Multipart"
	case "binary":
		return "Binary"
	case "graphql":
		return "GraphQL"
	}
	return t
}

func isRawBodyType(t string) bool {
	return t == "json" || t == "xml" || t == "text" || t == "graphql"
}

// BodyEditor owns the body type selector and the matching editor surface.
type BodyEditor struct {
	Type   string
	Raw    textarea.Model
	Fields KVEditor
	File   textinput.Model

	width  int
	height int
	err    string
}

func NewBodyEditor() BodyEditor {
	raw := textarea.New()
	raw.Prompt = "  "
	raw.ShowLineNumbers = true
	raw.CharLimit = 0
	file := textinput.New()
	file.Prompt = ""
	file.Placeholder = "Path file, contoh: ./payload.bin"
	b := BodyEditor{
		Type:   "none",
		Raw:    raw,
		Fields: NewKVEditor("body_fields", "params", true, false),
		File:   file,
	}
	b.Fields.ShowKind = true
	b.SetSize(70, 7)
	return b
}

func (b *BodyEditor) SetSize(width, height int) {
	b.width = max(width, 20)
	b.height = max(height, 3)
	b.Raw.SetWidth(max(b.width-4, 10))
	b.Raw.SetHeight(max(b.height-1, 2))
	b.Fields.SetSize(b.width, max(b.height-1, 2))
	b.File.SetWidth(max(b.width-6, 10))
}

func (b *BodyEditor) Focus() tea.Cmd {
	switch {
	case b.Type == "binary":
		return b.File.Focus()
	case b.Type == "form" || b.Type == "multipart":
		return b.Fields.Focus()
	case isRawBodyType(b.Type):
		return b.Raw.Focus()
	}
	return nil
}

func (b *BodyEditor) Blur() {
	b.Raw.Blur()
	b.Fields.Blur()
	b.File.Blur()
}

func (b *BodyEditor) Update(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+b":
			b.cycleType(1)
			return b.Focus()
		case "f6":
			b.formatBody(true)
			return nil
		case "f7":
			b.formatBody(false)
			return nil
		case "f8":
			b.validateBody()
			return nil
		}
	}
	switch b.Type {
	case "binary":
		var cmd tea.Cmd
		b.File, cmd = b.File.Update(msg)
		return cmd
	case "form", "multipart":
		return b.Fields.Update(msg)
	default:
		if isRawBodyType(b.Type) {
			if key, ok := msg.(tea.KeyMsg); ok {
				if key.String() == "tab" {
					b.Raw.InsertString("  ")
					return nil
				}
			}
			var cmd tea.Cmd
			b.Raw, cmd = b.Raw.Update(msg)
			return cmd
		}
	}
	return nil
}

func (b *BodyEditor) cycleType(delta int) {
	b.Fields.commitIfEditing()
	idx := 0
	for i, t := range bodyTypes {
		if t == b.Type {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(bodyTypes)) % len(bodyTypes)
	b.Type = bodyTypes[idx]
	b.err = ""
}

func (b *BodyEditor) SetType(t string) {
	b.Fields.commitIfEditing()
	for _, known := range bodyTypes {
		if known == t {
			b.Type = t
			b.err = ""
			return
		}
	}
	b.Type = "none"
}

func (b *BodyEditor) formatBody(pretty bool) {
	if !isRawBodyType(b.Type) {
		return
	}
	raw := b.Raw.Value()
	if b.Type == "json" || b.Type == "graphql" {
		var out bytes.Buffer
		var err error
		if pretty {
			err = json.Indent(&out, []byte(raw), "", "  ")
		} else {
			err = json.Compact(&out, []byte(raw))
		}
		if err != nil {
			b.err = "JSON tidak valid: " + err.Error()
			return
		}
		b.Raw.SetValue(out.String())
		b.err = ""
		return
	}
	b.validateBody()
}

func (b *BodyEditor) validateBody() {
	if b.Type == "json" || b.Type == "graphql" {
		if json.Valid([]byte(b.Raw.Value())) {
			b.err = ""
		} else {
			b.err = "JSON tidak valid"
		}
		return
	}
	if b.Type == "xml" {
		decoder := xml.NewDecoder(strings.NewReader(b.Raw.Value()))
		for {
			if _, err := decoder.Token(); err != nil {
				if errors.Is(err, io.EOF) {
					b.err = ""
				} else {
					b.err = "XML tidak valid: " + err.Error()
				}
				return
			}
		}
	}
}

func (b *BodyEditor) View() string {
	var pills []string
	for _, t := range bodyTypes {
		label := " " + bodyTypeLabel(t) + " "
		style := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
		if t == b.Type {
			style = lipgloss.NewStyle().
				Foreground(styles.ColBackground).
				Background(styles.ColAccent).
				Bold(true)
		}
		pills = append(pills, zone.Mark("body_type_"+t, style.Render(label)))
	}
	typeLine := strings.Join(pills, " ")
	hint := "Ctrl+B ganti tipe"
	if b.Type == "json" || b.Type == "graphql" {
		hint += " · F6 rapikan · F7 minify"
	}
	if lipgloss.Width(typeLine)+lipgloss.Width(hint)+2 < b.width {
		typeLine += "  " + styles.Dim().Render(hint)
	}

	contentHeight := max(b.height-1, 1)
	var content []string
	switch b.Type {
	case "none":
		content = []string{styles.Dim().Render("  Tidak ada request body.")}
	case "binary":
		content = append(content, "  File: "+b.File.View())
		content = append(content, styles.Dim().Render("  Isi path file yang akan dikirim sebagai body biner."))
	case "form":
		content = strings.Split(b.Fields.View(), "\n")
	case "multipart":
		content = append([]string{styles.Dim().Render("  TYPE: text|file (tekan t untuk ganti)")}, strings.Split(b.Fields.View(), "\n")...)
	case "graphql":
		content = append([]string{styles.Dim().Render("  Kirim payload GraphQL JSON: {\"query\": \"...\", \"variables\": {}}")}, strings.Split(b.Raw.View(), "\n")...)
	default:
		content = strings.Split(b.Raw.View(), "\n")
	}
	if b.err != "" {
		content = append([]string{styles.Dim().Render("! " + b.err)}, content...)
	}
	for len(content) < contentHeight {
		content = append(content, "")
	}
	if len(content) > contentHeight {
		content = content[:contentHeight]
	}
	return strings.Join(append([]string{typeLine}, content...), "\n")
}

func (b *BodyEditor) Spec() collectionstore.BodySpec {
	spec := collectionstore.BodySpec{Type: b.Type}
	switch b.Type {
	case "json", "xml", "text", "graphql":
		spec.Raw = b.Raw.Value()
	case "form", "multipart":
		rows, err := b.Fields.EffectiveRows()
		if err != nil {
			b.err = err.Error()
		}
		spec.Fields = rows
	case "binary":
		spec.File = strings.TrimSpace(b.File.Value())
	}
	return spec
}

func (b *BodyEditor) SetSpec(spec collectionstore.BodySpec) {
	b.SetType(spec.Type)
	b.Raw.SetValue(spec.Raw)
	b.Fields.SetRows(spec.Fields)
	b.File.SetValue(spec.File)
}

func (b *BodyEditor) Error() string { return b.err }

func (b *BodyEditor) commitIfEditing() {
	b.Fields.commitIfEditing()
}
