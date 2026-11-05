package requestinput

import (
	"testing"

	"api/internal/model"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func TestRequestBarWidthMatchesTarget(t *testing.T) {
	zone.NewGlobal()

	widths := []int{50, 60, 80, 105, 120}
	methods := []string{"GET", "POST", "OPTIONS", ""}

	for _, width := range widths {
		for _, method := range methods {
			m := New(model.NewSharedState())
			m.Method = method
			m.SetWidth(width)

			if got := lipgloss.Width(m.View()); got != width {
				t.Errorf("width %d method %q: request bar rendered %d cells", width, method, got)
			}
		}
	}
}
