package envscreen

import (
	"strings"
	"testing"

	"api/internal/envstore"

	"charm.land/lipgloss/v2"
)

func testModel() *Model {
	m := New(nil, "dev")
	m.Envs = []string{"dev", "prod"}
	m.Current = &envstore.EnvFile{
		Name: "dev",
		Variables: map[string]string{
			"BASE_URL": "http://localhost:8080",
			"TOKEN":    "super-secret-value",
		},
	}
	m.rebuildVarKeys()
	return m
}

func TestViewShowsThemeInfoAndVariables(t *testing.T) {
	m := testModel()
	m.SetSize(120, 40)

	out := m.View()
	for _, want := range []string{
		"ENVIRONMENT VARIABLES",
		"ENVIRONMENTS",
		"VARIABLES — dev · 2",
		"USAGE & INFO",
		"{{KEY}}",
		".termapi/envs/",
		"SECURITY",
		"BASE_URL",
		"super-secret-value",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q", want)
		}
	}

	if got := lipgloss.Height(out); got != 40 {
		t.Errorf("view height = %d, want 40", got)
	}
	if got := lipgloss.Width(out); got != 120 {
		t.Errorf("view width = %d, want 120", got)
	}
}

func TestViewKeepsBoundsOnMinimumWindow(t *testing.T) {
	m := testModel()
	m.SetSize(100, 28)

	out := m.View()
	if got := lipgloss.Height(out); got != 28 {
		t.Errorf("view height = %d, want 28", got)
	}
	if got := lipgloss.Width(out); got > 100 {
		t.Errorf("view width = %d, want <= 100", got)
	}
}
