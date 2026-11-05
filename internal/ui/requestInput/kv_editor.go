package requestinput

import (
	"fmt"
	"strings"

	"api/internal/collectionstore"
	"api/internal/ui/styles"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

// KVEditor is a small editable key/value table used by the request tabs.
// It has a navigation mode and an edit mode so single-letter shortcuts do not
// conflict with typing.
type KVEditor struct {
	Rows            []collectionstore.KVRow
	Row             int
	Col             int
	Editing         bool
	ShowKind        bool
	ShowDescription bool
	RawAllowed      bool
	RawKind         string // params|headers
	Zone            string
	Suggestions     []string
	MaskSensitive   bool
	suggestionIndex int

	input    textinput.Model
	rawInput textarea.Model
	raw      bool
	rawErr   string
	width    int
	height   int
	original string
}

func NewKVEditor(zonePrefix, rawKind string, showDescription, rawAllowed bool) KVEditor {
	in := textinput.New()
	in.Prompt = ""
	raw := textarea.New()
	raw.Prompt = "  "
	raw.ShowLineNumbers = false
	raw.CharLimit = 0
	k := KVEditor{
		Rows:            []collectionstore.KVRow{{}},
		ShowDescription: showDescription,
		RawAllowed:      rawAllowed,
		RawKind:         rawKind,
		Zone:            zonePrefix,
		input:           in,
		rawInput:        raw,
	}
	if rawKind == "headers" {
		k.MaskSensitive = true
		k.Suggestions = []string{
			"Accept", "Accept-Encoding", "Accept-Language", "Authorization",
			"Cache-Control", "Content-Type", "Cookie", "Origin", "Referer",
			"User-Agent", "X-Api-Key", "X-Request-Id",
		}
	}
	k.SetSize(70, 7)
	return k
}

func (k *KVEditor) SetRows(rows []collectionstore.KVRow) {
	k.Rows = append([]collectionstore.KVRow(nil), rows...)
	if len(k.Rows) == 0 {
		k.Rows = []collectionstore.KVRow{{}}
	}
	k.raw = false
	k.rawErr = ""
	k.Row, k.Col, k.Editing = 0, 0, false
	k.input.Blur()
	k.rawInput.Blur()
}

func (k *KVEditor) SetSize(width, height int) {
	k.width = max(width, 20)
	k.height = max(height, 3)
	k.rawInput.SetWidth(max(k.width-4, 10))
	k.rawInput.SetHeight(max(k.height-1, 2))
	k.syncInputWidth()
}

func (k *KVEditor) Focus() tea.Cmd {
	if k.raw {
		return k.rawInput.Focus()
	}
	return nil
}

func (k *KVEditor) Blur() {
	k.Editing = false
	k.input.Blur()
	k.rawInput.Blur()
}

func (k *KVEditor) IsRaw() bool { return k.raw }

func (k *KVEditor) Error() string { return k.rawErr }

func (k *KVEditor) Count() int {
	rows, err := k.EffectiveRows()
	if err != nil {
		return 0
	}
	n := 0
	for _, row := range rows {
		if strings.TrimSpace(row.Key) != "" || strings.TrimSpace(row.Value) != "" {
			n++
		}
	}
	return n
}

// EffectiveRows returns the rows currently represented by the editor,
// including rows typed in raw mode.
func (k *KVEditor) EffectiveRows() ([]collectionstore.KVRow, error) {
	if !k.raw {
		return append([]collectionstore.KVRow(nil), k.Rows...), nil
	}
	return parseKVRaw(k.RawKind, k.rawInput.Value())
}

func (k *KVEditor) ToggleRaw() error {
	if !k.RawAllowed {
		return nil
	}
	if k.Editing {
		k.commit()
	}
	if !k.raw {
		k.raw = true
		k.Editing = false
		k.input.Blur()
		k.rawInput.SetValue(k.serializeRaw())
		k.rawInput.Focus()
		k.rawErr = ""
		return nil
	}
	rows, err := parseKVRaw(k.RawKind, k.rawInput.Value())
	if err != nil {
		k.rawErr = err.Error()
		return err
	}
	k.Rows = rows
	if len(k.Rows) == 0 {
		k.Rows = []collectionstore.KVRow{{}}
	}
	k.Row = min(k.Row, len(k.Rows)-1)
	k.Col, k.Editing = 0, false
	k.raw = false
	k.rawErr = ""
	k.rawInput.Blur()
	return nil
}

func (k *KVEditor) serializeRaw() string {
	lines := make([]string, 0, len(k.Rows))
	for _, row := range k.Rows {
		if strings.TrimSpace(row.Key) == "" && strings.TrimSpace(row.Value) == "" {
			continue
		}
		prefix := ""
		if row.Disabled {
			prefix = "# "
		}
		if k.RawKind == "headers" {
			lines = append(lines, prefix+row.Key+": "+row.Value)
		} else {
			lines = append(lines, prefix+row.Key+"="+row.Value)
		}
	}
	return strings.Join(lines, "\n")
}

func parseKVRaw(kind, raw string) ([]collectionstore.KVRow, error) {
	var rows []collectionstore.KVRow
	for i, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		disabled := false
		if strings.HasPrefix(line, "#") {
			disabled = true
			line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
		if line == "" {
			continue
		}
		var key, value string
		var ok bool
		if kind == "headers" {
			key, value, ok = strings.Cut(line, ":")
		} else {
			key, value, ok = strings.Cut(line, "=")
			if !ok {
				key, value, ok = strings.Cut(line, ":")
			}
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || key == "" {
			return nil, fmt.Errorf("baris %d tidak valid; gunakan %s", i+1, rawFormatHint(kind))
		}
		rows = append(rows, collectionstore.KVRow{Key: key, Value: value, Disabled: disabled})
	}
	return rows, nil
}

func rawFormatHint(kind string) string {
	if kind == "headers" {
		return "Nama: nilai"
	}
	return "key=value"
}

func (k *KVEditor) Update(msg tea.Msg) tea.Cmd {
	if k.raw {
		if key, ok := msg.(tea.KeyMsg); ok {
			if key.String() == "ctrl+r" {
				_ = k.ToggleRaw()
				return nil
			}
		}
		var cmd tea.Cmd
		k.rawInput, cmd = k.rawInput.Update(msg)
		return cmd
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	if k.Editing {
		switch key.String() {
		case "esc":
			k.input.SetValue(k.original)
			k.commit()
			return nil
		case "enter":
			k.commit()
			return nil
		case "tab":
			k.commit()
			k.moveCell(1)
			k.startEditing()
			return nil
		case "shift+tab":
			k.commit()
			k.moveCell(-1)
			k.startEditing()
			return nil
		}
		var cmd tea.Cmd
		k.input, cmd = k.input.Update(key)
		return cmd
	}

	switch key.String() {
	case "ctrl+r":
		_ = k.ToggleRaw()
		return nil
	case "up", "k":
		k.moveRow(-1)
	case "down", "j":
		k.moveRow(1)
	case "left", "h":
		k.moveCell(-1)
	case "right", "l":
		k.moveCell(1)
	case "enter", "e":
		k.startEditing()
	case "a":
		k.addRow()
	case "d":
		k.deleteRow()
	case "space", " ":
		if k.Row >= 0 && k.Row < len(k.Rows) {
			k.Rows[k.Row].Disabled = !k.Rows[k.Row].Disabled
		}
	case "t":
		if k.ShowKind && k.Row >= 0 && k.Row < len(k.Rows) {
			if k.Rows[k.Row].Kind == "file" {
				k.Rows[k.Row].Kind = "text"
			} else {
				k.Rows[k.Row].Kind = "file"
			}
		}
	case "s":
		if len(k.Suggestions) > 0 && k.Row >= 0 && k.Row < len(k.Rows) && strings.TrimSpace(k.Rows[k.Row].Key) == "" {
			k.Rows[k.Row].Key = k.Suggestions[k.suggestionIndex%len(k.Suggestions)]
			k.suggestionIndex++
		}
	}
	return nil
}

func (k *KVEditor) cellZone(row, col int) string {
	return fmt.Sprintf("%s_cell_%d_%d", k.Zone, row, col)
}

// HandleClick langsung membuka edit pada sel yang diklik, tanpa perlu Enter.
func (k *KVEditor) HandleClick(msg tea.MouseClickMsg) bool {
	if k.raw {
		return false
	}
	for i := range k.Rows {
		for col := 0; col < k.columns(); col++ {
			if zone.Get(k.cellZone(i, col)).InBounds(msg) {
				if i == k.Row && col == k.Col && k.Editing {
					return true
				}
				k.commitIfEditing()
				k.Row, k.Col = i, col
				k.startEditing()
				return true
			}
		}
	}
	for i := range k.Rows {
		if zone.Get(fmt.Sprintf("%s_row_%d", k.Zone, i)).InBounds(msg) {
			k.commitIfEditing()
			k.Row = i
			k.Col = 0
			k.Editing = false
			k.input.Blur()
			return true
		}
	}
	return false
}

func (k *KVEditor) moveRow(delta int) {
	if len(k.Rows) == 0 {
		return
	}
	k.Row = (k.Row + delta + len(k.Rows)) % len(k.Rows)
	k.Editing = false
	k.input.Blur()
}

func (k *KVEditor) columns() int {
	if k.ShowKind {
		return 4
	}
	cols := 3
	if !k.ShowDescription {
		cols = 2
	}
	return cols
}

func (k *KVEditor) moveCell(delta int) {
	cols := k.columns()
	k.Col = (k.Col + delta + cols) % cols
	k.Editing = false
	k.input.Blur()
	k.syncInputWidth()
}

func (k *KVEditor) addRow() {
	insert := k.Row + 1
	if insert <= 0 || insert > len(k.Rows) {
		insert = len(k.Rows)
	}
	k.Rows = append(k.Rows, collectionstore.KVRow{})
	copy(k.Rows[insert+1:], k.Rows[insert:])
	k.Rows[insert] = collectionstore.KVRow{}
	k.Row = insert
	k.Col = 0
}

func (k *KVEditor) deleteRow() {
	if len(k.Rows) <= 1 {
		k.Rows = []collectionstore.KVRow{{}}
		k.Row = 0
		return
	}
	k.Rows = append(k.Rows[:k.Row], k.Rows[k.Row+1:]...)
	if k.Row >= len(k.Rows) {
		k.Row = len(k.Rows) - 1
	}
}

func (k *KVEditor) startEditing() {
	if k.Row < 0 || k.Row >= len(k.Rows) {
		return
	}
	k.original = k.cellValue(k.Row, k.Col)
	k.input.SetValue(k.original)
	k.input.CursorEnd()
	k.syncInputWidth()
	k.input.Focus()
	k.Editing = true
}

func (k *KVEditor) commitIfEditing() {
	if k.Editing {
		k.commit()
	}
}

func (k *KVEditor) commit() {
	if k.Row >= 0 && k.Row < len(k.Rows) {
		k.setCellValue(k.Row, k.Col, strings.TrimSpace(k.input.Value()))
	}
	k.Editing = false
	k.input.Blur()
}

func (k *KVEditor) cellValue(row, col int) string {
	if row < 0 || row >= len(k.Rows) {
		return ""
	}
	switch col {
	case 0:
		return k.Rows[row].Key
	case 1:
		return k.Rows[row].Value
	case 2:
		return k.Rows[row].Description
	case 3:
		return k.Rows[row].Kind
	}
	return ""
}

func (k *KVEditor) setCellValue(row, col int, value string) {
	if row < 0 || row >= len(k.Rows) {
		return
	}
	switch col {
	case 0:
		k.Rows[row].Key = value
	case 1:
		k.Rows[row].Value = value
	case 2:
		k.Rows[row].Description = value
	case 3:
		if strings.EqualFold(value, "file") {
			k.Rows[row].Kind = "file"
		} else {
			k.Rows[row].Kind = "text"
		}
	}
}

func (k *KVEditor) syncInputWidth() {
	if k.Col < 0 {
		k.Col = 0
	}
	keyW, valW, descW, kindW := k.columnWidths()
	switch k.Col {
	case 0:
		k.input.SetWidth(max(keyW-1, 1))
	case 1:
		k.input.SetWidth(max(valW-1, 1))
	case 2:
		k.input.SetWidth(max(descW-1, 1))
	case 3:
		k.input.SetWidth(max(kindW-1, 1))
	}
}

func (k *KVEditor) columnWidths() (int, int, int, int) {
	kindW := 0
	if k.ShowKind {
		kindW = 6
	}
	avail := max(k.width-2-4-kindW-(k.columns()-1), 24)
	keyW := max(8, avail*28/100)
	valW := max(10, avail*40/100)
	descW := 0
	if k.ShowDescription {
		descW = max(0, avail-keyW-valW)
	} else {
		valW = max(10, avail-keyW)
	}
	return keyW, valW, descW, kindW
}

func fitCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = ansi.Truncate(s, width, "…")
	if pad := width - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func (k *KVEditor) View() string {
	if k.raw {
		view := k.rawInput.View()
		lines := strings.Split(view, "\n")
		if k.rawErr != "" {
			errLine := styles.Dim().Render(ansi.Truncate("! "+k.rawErr, max(k.width-2, 10), "…"))
			if len(lines) > 0 {
				lines[len(lines)-1] = errLine
			} else {
				lines = append(lines, errLine)
			}
		}
		return padLines(lines, k.height)
	}

	keyW, valW, descW, kindW := k.columnWidths()
	// Prefix header sama dengan lebar marker "› " + checkbox "[x] " pada baris.
	header := strings.Repeat(" ", 6) + fitCell("KEY", keyW) + " " + fitCell("VALUE", valW)
	if k.ShowDescription {
		header += " " + fitCell("DESC", descW)
	}
	if k.ShowKind {
		header += " " + fitCell("TYPE", kindW)
	}
	lines := []string{styles.Dim().Render(header)}

	visibleHeight := max(k.height-1, 1)
	start := 0
	if k.Row >= visibleHeight {
		start = k.Row - visibleHeight + 1
	}
	end := min(start+visibleHeight, len(k.Rows))
	for i := start; i < end; i++ {
		lines = append(lines, k.rowView(i, keyW, valW, descW, kindW))
	}
	for len(lines) < k.height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:min(len(lines), k.height)], "\n")
}

func (k *KVEditor) rowView(i, keyW, valW, descW, kindW int) string {
	row := k.Rows[i]
	check := "[x] "
	if row.Disabled {
		check = "[ ] "
		check = styles.Dim().Render(check)
	}
	cells := []string{
		k.cellView(row.Key, i, 0, keyW),
		k.cellView(row.Value, i, 1, valW),
	}
	if k.ShowDescription {
		cells = append(cells, k.cellView(row.Description, i, 2, descW))
	}
	if k.ShowKind {
		kind := row.Kind
		if kind == "" {
			kind = "text"
		}
		cells = append(cells, k.cellView(kind, i, 3, kindW))
	}
	for col := range cells {
		if row.Disabled {
			cells[col] = styles.Dim().Render(ansi.Strip(cells[col]))
		}
		cells[col] = zone.Mark(k.cellZone(i, col), cells[col])
	}
	// Marker seleksi harus tetap jelas meski baris disabled.
	marker := "  "
	if i == k.Row {
		marker = styles.Bright().Render("› ")
	}
	line := marker + check + strings.Join(cells, " ")
	return zone.Mark(fmt.Sprintf("%s_row_%d", k.Zone, i), line)
}

func (k *KVEditor) cellView(value string, row, col, width int) string {
	if width <= 0 {
		return ""
	}
	if k.Editing && row == k.Row && col == k.Col {
		return fitCell(k.input.View(), width)
	}
	if col == 1 && k.MaskSensitive && row >= 0 && row < len(k.Rows) && sensitiveHeaderName(k.Rows[row].Key) {
		value = strings.Repeat("•", len([]rune(value)))
	}
	cell := fitCell(value, width)
	if row == k.Row && col == k.Col {
		return styles.Selected().Render(cell)
	}
	return cell
}

func sensitiveHeaderName(name string) bool {
	name = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "-", ""))
	return strings.Contains(name, "auth") || strings.Contains(name, "cookie") ||
		strings.Contains(name, "token") || strings.Contains(name, "secret") ||
		strings.Contains(name, "password") || strings.Contains(name, "credential") ||
		strings.Contains(name, "session") || strings.Contains(name, "apikey") || strings.HasSuffix(name, "key")
}

func padLines(lines []string, height int) string {
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
