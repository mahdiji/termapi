package requestinput

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func assertFrame(t *testing.T, p *ResponsePanel, wantWidth, wantHeight int) {
	t.Helper()
	lines := strings.Split(p.View(), "\n")
	if len(lines) != wantHeight {
		t.Fatalf("rendered %d lines, want %d", len(lines), wantHeight)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != wantWidth {
			t.Fatalf("line %d rendered %d cells, want %d: %q", i, got, wantWidth, line)
		}
	}
}

func TestResponseFrameStaysAlignedWithLongContent(t *testing.T) {
	zone.NewGlobal()

	longLine := strings.Repeat("var x = \"https://example.com/long/path/that/keeps/going.css\"; ", 8)
	body := "line1\n" + longLine + "\nline3\n" + longLine
	html := "<html>\r\n<body>\r\n\t<script>\r\n\t\tif (x) { y(); }\r\n\t</script>\r\n</body>\r\n</html>"

	tests := []struct {
		name     string
		response ResponseMsg
		tab      int
		raw      bool
	}{
		{
			name:     "pretty long ascii",
			response: ResponseMsg{Status: "200 OK", Code: 200, Body: body, RawBody: body},
		},
		{
			name:     "raw crlf tabs",
			response: ResponseMsg{Status: "200 OK", Code: 200, Body: html, RawBody: html},
			raw:      true,
		},
		{
			name:     "raw ansi escapes",
			response: ResponseMsg{Status: "200 OK", Code: 200, Body: "a\x1b[31mred\x1b[0mb", RawBody: "a\x1b[31mred\x1b[0mb"},
			raw:      true,
		},
		{
			name: "headers with control chars",
			response: ResponseMsg{
				Status:  "200 OK",
				Code:    200,
				Body:    "{}",
				RawBody: "{}",
				Headers: []string{"Content-Type: text/html\r", "X-Long: " + longLine},
			},
			tab: 2,
		},
		{
			name: "cookies with tabs",
			response: ResponseMsg{
				Status:  "200 OK",
				Code:    200,
				Body:    "{}",
				RawBody: "{}",
				Cookies: []string{"session=abc\tdef", "other=1"},
			},
			tab: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewResponsePanel()
			p.SetSize(80, 20)
			p.SetResponse(tt.response)
			p.Tab = tt.tab
			p.RawMode = tt.raw
			assertFrame(t, p, 80, 20)
		})
	}
}
