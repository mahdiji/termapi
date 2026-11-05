package diffscreen

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestFitLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
	}{
		{"plain short", "hello", 10},
		{"plain exact", "hello", 5},
		{"plain long", "hello world", 5},
		{"ansi colored", "\x1b[31mred text\x1b[0m", 15},
		{"ansi colored short", "\x1b[31mred\x1b[0m", 10},
		{"cjk and emoji", "你好 🚀 world", 15},
		{"width 1", "hello", 1},
		{"width 200", "hello", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := FitLine(tt.input, tt.width)
			gotWidth := ansi.StringWidth(res)
			if gotWidth != tt.width {
				t.Errorf("FitLine() width = %d, want %d (result: %q)", gotWidth, tt.width, res)
			}
		})
	}
}

func TestBoxWithTitle(t *testing.T) {
	title := "TEST BOX"
	body := "line 1\n\x1b[32mline 2 green\x1b[0m\n你好 emoji 🚀"

	widths := []int{30, 80, 200}
	heights := []int{10, 20, 30}

	for _, w := range widths {
		for _, h := range heights {
			res := BoxWithTitle(title, body, w, h)
			lines := strings.Split(res, "\n")
			if len(lines) != h {
				t.Errorf("BoxWithTitle height = %d, want %d (width=%d)", len(lines), h, w)
			}
			for idx, line := range lines {
				lw := ansi.StringWidth(line)
				if lw != w {
					t.Errorf("BoxWithTitle line %d width = %d, want %d", idx, lw, w)
				}
			}
		}
	}
}

func TestViewFitsRequestedSize(t *testing.T) {
	m := NewModel(`{"name":"a","items":[1,2,3]}`, `{"name":"b","items":[1,2,3,4]}`)
	sizes := [][2]int{{60, 15}, {100, 24}, {140, 40}}

	for _, size := range sizes {
		m.SetSize(size[0], size[1])
		for _, help := range []bool{false, true} {
			m.ShowHelp = help
			lines := strings.Split(m.View(), "\n")
			if len(lines) != size[1] {
				t.Errorf("size %dx%d help=%v: height = %d, want %d", size[0], size[1], help, len(lines), size[1])
			}
			for idx, line := range lines {
				if lw := ansi.StringWidth(line); lw != size[0] {
					t.Errorf("size %dx%d help=%v: line %d width = %d, want %d", size[0], size[1], help, idx, lw, size[0])
				}
			}
		}
	}
}

func TestBaselinePickerAppliesSelection(t *testing.T) {
	m := NewModel("old", "new")
	m.Options = []BaselineOption{
		{Label: "latest", Body: "new"},
		{Label: "older", Body: "old"},
	}
	m.SelectedA = 1

	m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	if !m.PickerOpen {
		t.Fatal("expected picker to open")
	}
	m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.PickerOpen {
		t.Fatal("expected picker to close after selection")
	}
	if m.OriginalText != "new" || m.SelectedA != 0 {
		t.Fatalf("baseline not applied: text=%q selected=%d", m.OriginalText, m.SelectedA)
	}
	if m.DiffResult == nil || m.DiffResult.StatsAdd != 0 || m.DiffResult.StatsDel != 0 {
		t.Fatalf("expected identical diff, got %+v", m.DiffResult)
	}

	m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.PickerOpen {
		t.Fatal("expected Esc to close picker")
	}
}

func TestViewFitsSizeWithPicker(t *testing.T) {
	m := NewModel("old", "new")
	m.Options = []BaselineOption{
		{Label: "latest", Body: "new"},
		{Label: "older", Body: "old"},
	}
	m.SetSize(100, 24)
	m.PickerOpen = true
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 24 {
		t.Fatalf("height = %d, want 24", len(lines))
	}
	for idx, line := range lines {
		if lw := ansi.StringWidth(line); lw != 100 {
			t.Fatalf("line %d width = %d, want 100", idx, lw)
		}
	}
}
