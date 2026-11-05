package requestinput

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		title string
	}{
		{"dns", `Get "https://nope.invalid": dial tcp: lookup nope.invalid: no such host`, "Host not found"},
		{"refused", "dial tcp 127.0.0.1:9: connect: connection refused", "Connection refused"},
		{"timeout", "context deadline exceeded", "Request timed out"},
		{"tls", "x509: certificate signed by unknown authority", "TLS verification failed"},
		{"scope", "credentials blocked: add the exact request origin to .termapi/global.yaml", "Request blocked by Scope"},
		{"invalid url", "invalid URL: missing host", "Invalid URL"},
		{"header", `invalid header line: "X-Bad: a\nb"`, "Invalid header"},
		{"file", `multipart file "./nope.bin": open ./nope.bin: no such file`, "Cannot read body file"},
		{"redirect", "redirect blocked: destination origin is outside the Scope allowlist", "Redirect blocked"},
		{"unknown", "something odd happened", "Request failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyError(tt.raw).Title; got != tt.title {
				t.Fatalf("title = %q, want %q", got, tt.title)
			}
		})
	}
}

func TestErrorLinesHideRawMessageOutsideRawMode(t *testing.T) {
	p := NewResponsePanel()
	p.SetSize(90, 18)
	p.Err = "dial tcp 127.0.0.1:9: connect: connection refused"

	pretty := ansi.Strip(p.View())
	if strings.Contains(pretty, "dial tcp") {
		t.Fatalf("raw error leaked into pretty mode:\n%s", pretty)
	}
	if !strings.Contains(pretty, "Connection refused") {
		t.Fatalf("friendly title missing:\n%s", pretty)
	}

	p.RawMode = true
	raw := ansi.Strip(p.View())
	if !strings.Contains(raw, "dial tcp") {
		t.Fatalf("raw mode should show the technical error:\n%s", raw)
	}
}
