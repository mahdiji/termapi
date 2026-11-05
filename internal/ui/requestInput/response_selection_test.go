package requestinput

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func waitZone(t *testing.T, id string) *zone.ZoneInfo {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		if z := zone.Get(id); !z.IsZero() {
			return z
		}
		if time.Now().After(deadline) {
			t.Fatalf("zone %q not registered", id)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func newSelectablePanel(t *testing.T) *ResponsePanel {
	t.Helper()
	zone.NewGlobal()
	p := NewResponsePanel()
	p.SetSize(90, 18)
	p.SetResponse(ResponseMsg{
		Status:  "200 OK",
		Code:    200,
		Body:    "alpha\nbeta\ngamma\ndelta",
		RawBody: "alpha\nbeta\ngamma\ndelta",
	})
	_ = zone.Scan(p.View())
	return p
}

func TestMouseDragSelectsLinesAndCopy(t *testing.T) {
	p := newSelectablePanel(t)
	z := waitZone(t, "response_body")

	start := tea.MouseClickMsg{X: z.StartX + 2, Y: z.StartY + 1, Button: tea.MouseLeft}
	p.HandleMouse(start)
	p.HandleMotion(tea.MouseMotionMsg{X: z.StartX + 2, Y: z.StartY + 2, Button: tea.MouseLeft})

	text, lines, ok := p.CopySelection()
	if !ok {
		t.Fatal("expected an active selection")
	}
	if text != "beta\ngamma" || lines != 2 {
		t.Fatalf("selected text = %q (%d lines), want beta/gamma (2 lines)", text, lines)
	}

	p.HandleRelease()
	if !p.SelActive {
		t.Fatal("selection should remain active after release")
	}
}

func TestSingleClickDoesNotKeepSelection(t *testing.T) {
	p := newSelectablePanel(t)
	z := waitZone(t, "response_body")

	p.HandleMouse(tea.MouseClickMsg{X: z.StartX + 2, Y: z.StartY, Button: tea.MouseLeft})
	p.HandleRelease()

	if p.SelActive {
		t.Fatal("single click should not create a persistent selection")
	}
}

func TestCopyAllReturnsActiveTabText(t *testing.T) {
	p := newSelectablePanel(t)
	p.Tab = 2
	p.Headers = []string{"Accept: application/json", "X-Test: 1"}
	_ = zone.Scan(p.View())

	text, lines := p.CopyAll()
	if lines != 2 || text != "Accept: application/json\nX-Test: 1" {
		t.Fatalf("copy all = %q (%d lines)", text, lines)
	}
}
