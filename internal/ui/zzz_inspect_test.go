package ui

import (
	"api/internal/model"
	zone "github.com/lrstanley/bubblezone/v2"
	"regexp"
	"strings"
	"testing"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestInspect(t *testing.T) {
	zone.NewGlobal()
	st := model.NewSharedState()
	m := NewAppModel(st)
	m.ScreenWidth, m.ScreenHeight = 140, 40
	m.State.Widht, m.State.Height = 140, 40
	m.recalculateLayout()
	out := m.viewMain().Content
	plain := ansiRe.ReplaceAllString(out, "")
	for i, l := range strings.Split(plain, "\n") {
		t.Logf("%2d|%s|", i, l)
	}
}
