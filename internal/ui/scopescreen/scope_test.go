package scopescreen

import (
	"strings"
	"testing"
)

func TestParseOrigin(t *testing.T) {
	tests := []struct {
		raw      string
		valid    bool
		scheme   string
		host     string
		port     string
		hostType string
		risk     string
	}{
		{"http://localhost:8080", true, "http", "localhost", "8080", "LOCAL", "MEDIUM"},
		{"https://api.example.com", true, "https", "api.example.com", "443", "PUBLIC", "LOW"},
		{"http://192.168.1.10:3000", true, "http", "192.168.1.10", "3000", "LAN", "MEDIUM"},
		{"http://example.com", true, "http", "example.com", "80", "PUBLIC", "HIGH"},
		{"ftp://example.com", false, "—", "—", "—", "UNKNOWN", "—"},
		{"example.com", false, "—", "—", "—", "UNKNOWN", "—"},
		{"", false, "—", "—", "—", "UNKNOWN", "—"},
	}

	for _, tc := range tests {
		info := parseOrigin(tc.raw)
		if info.Valid != tc.valid {
			t.Errorf("parseOrigin(%q).Valid = %v, want %v", tc.raw, info.Valid, tc.valid)
			continue
		}
		if !tc.valid {
			continue
		}
		if info.Scheme != tc.scheme || info.Host != tc.host || info.Port != tc.port {
			t.Errorf("parseOrigin(%q) = %s://%s:%s, want %s://%s:%s",
				tc.raw, info.Scheme, info.Host, info.Port, tc.scheme, tc.host, tc.port)
		}
		if info.Type != tc.hostType || info.Risk != tc.risk {
			t.Errorf("parseOrigin(%q) type/risk = %s/%s, want %s/%s",
				tc.raw, info.Type, info.Risk, tc.hostType, tc.risk)
		}
	}
}

func TestBuildStats(t *testing.T) {
	stats := buildStats([]string{
		"https://api.example.com",
		"http://localhost:8080",
		"not-an-origin",
	})

	if stats.Total != 3 || stats.HTTPS != 1 || stats.HTTP != 1 ||
		stats.Local != 1 || stats.Public != 1 || stats.Invalid != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestViewRendersScopeDetails(t *testing.T) {
	m := &Model{
		Width:   120,
		Height:  40,
		Origins: []string{"http://localhost:8080", "https://api.example.com"},
		Cursor:  1,
	}

	out := m.View()
	for _, want := range []string{
		"GLOBAL SCOPE",
		"ALLOWED ORIGINS · 2",
		"ORIGIN DETAIL",
		"SECURITY & USAGE",
		"api.example.com",
		"SUMMARY",
		"HTTPS",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q", want)
		}
	}
}

func TestViewEmptyState(t *testing.T) {
	m := &Model{Width: 120, Height: 40}

	out := m.View()
	for _, want := range []string{"No allowed origins yet.", "Press 'a' to add an origin."} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q", want)
		}
	}
}
