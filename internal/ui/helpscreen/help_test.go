package helpscreen

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHelpScrollsWhenContentExceedsHeight(t *testing.T) {
	m := NewModel()
	m.SetSize(120, 20)

	top := ansi.Strip(m.View())
	m.ScrollBy(5)
	if m.Scroll != 5 {
		t.Fatalf("scroll = %d, want 5", m.Scroll)
	}
	shifted := ansi.Strip(m.View())
	if top == shifted {
		t.Fatal("expected the rendered help to change after scrolling")
	}

	m.ScrollBottom()
	if m.Scroll == 0 {
		t.Fatal("expected a non-zero bottom scroll position")
	}
	bottom := ansi.Strip(m.View())
	if bottom == shifted {
		t.Fatal("expected different content at the bottom")
	}
}

func TestHelpScrollIsClamped(t *testing.T) {
	m := NewModel()
	m.SetSize(120, 40)

	m.ScrollBy(-10)
	if m.Scroll != 0 {
		t.Fatalf("scroll = %d, want 0", m.Scroll)
	}

	m.ScrollBy(1000)
	atBottom := m.Scroll
	if atBottom == 0 {
		t.Fatal("expected content to scroll")
	}
	m.ScrollBy(10)
	if m.Scroll != atBottom {
		t.Fatalf("scroll moved past bottom: %d -> %d", atBottom, m.Scroll)
	}

	m.ScrollTop()
	if m.Scroll != 0 {
		t.Fatalf("scroll = %d, want 0 after ScrollTop", m.Scroll)
	}
}
