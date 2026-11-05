package ui

import (
	"testing"

	"api/internal/model"
	tea "charm.land/bubbletea/v2"
)

func newHelpTestApp() AppModel {
	return NewAppModel(model.NewSharedState())
}

func pressKey(t *testing.T, app AppModel, key tea.KeyPressMsg) AppModel {
	t.Helper()
	updated, _ := app.Update(key)
	next, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel, got %T", updated)
	}
	return next
}

func TestF1OpensHelpFromURLInput(t *testing.T) {
	app := newHelpTestApp()
	if app.ReqState.FocusArea != 1 {
		t.Fatalf("expected URL focus at startup, got %d", app.ReqState.FocusArea)
	}

	app = pressKey(t, app, tea.KeyPressMsg{Code: tea.KeyF1})
	if !app.HelpMode {
		t.Fatal("expected F1 to open help while URL input is focused")
	}
}

func TestQuestionMarkStillTypesInURLInput(t *testing.T) {
	app := newHelpTestApp()

	app = pressKey(t, app, tea.KeyPressMsg{Text: "?", Code: '?'})
	if app.HelpMode {
		t.Fatal("expected ? to be typed instead of opening help in URL input")
	}
	if got := app.ReqState.Input.Value(); got != "?" {
		t.Fatalf("expected URL input to contain %q, got %q", "?", got)
	}
}

func TestAltQuestionMarkOpensHelpOutsideTextInput(t *testing.T) {
	app := newHelpTestApp()
	app.ReqState.FocusArea = 0

	app = pressKey(t, app, tea.KeyPressMsg{Text: "?", Code: '?', Mod: tea.ModAlt})
	if !app.HelpMode {
		t.Fatal("expected alt+? to open help when focus is not in a text field")
	}
}

func TestAltQuestionMarkOpensHelpWhenSidebarFocused(t *testing.T) {
	app := newHelpTestApp()
	app.State.TabActive = 1

	app = pressKey(t, app, tea.KeyPressMsg{Text: "?", Code: '?', Mod: tea.ModAlt})
	if !app.HelpMode {
		t.Fatal("expected alt+? to open help while sidebar is focused")
	}
}

func TestHelpScrollKeys(t *testing.T) {
	app := newHelpTestApp()
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(AppModel)

	app = pressKey(t, app, tea.KeyPressMsg{Code: tea.KeyF1})
	if !app.HelpMode {
		t.Fatal("expected help to be open")
	}

	app = pressKey(t, app, tea.KeyPressMsg{Text: "j", Code: 'j'})
	if app.HelpScreen.Scroll != 1 {
		t.Fatalf("scroll after j = %d, want 1", app.HelpScreen.Scroll)
	}
	app = pressKey(t, app, tea.KeyPressMsg{Code: tea.KeyPgDown})
	if app.HelpScreen.Scroll <= 1 {
		t.Fatalf("scroll after pgdown = %d, want > 1", app.HelpScreen.Scroll)
	}
	if !app.HelpMode {
		t.Fatal("scrolling must not close help")
	}

	app = pressKey(t, app, tea.KeyPressMsg{Code: tea.KeyHome})
	if app.HelpScreen.Scroll != 0 {
		t.Fatalf("scroll after home = %d, want 0", app.HelpScreen.Scroll)
	}
}

func TestHelpClosesWithF1AndAltQuestionMark(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{
		{Code: tea.KeyF1},
		{Text: "?", Code: '?', Mod: tea.ModAlt},
	} {
		app := newHelpTestApp()
		app = pressKey(t, app, tea.KeyPressMsg{Code: tea.KeyF1})
		if !app.HelpMode {
			t.Fatal("expected help to be open")
		}
		app = pressKey(t, app, key)
		if app.HelpMode {
			t.Fatalf("expected key %q to close help", key.String())
		}
	}
}
