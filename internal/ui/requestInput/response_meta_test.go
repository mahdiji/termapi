package requestinput

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPerformCapturesRedirectsAndTimings(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	response := performRequest(requestSpec{Method: "GET", URL: server.URL + "/start"}).(ResponseMsg)
	if response.Err != "" {
		t.Fatal(response.Err)
	}
	if len(response.Redirects) != 1 || !strings.HasSuffix(response.Redirects[0], "/final") {
		t.Fatalf("redirects = %v, want one entry ending in /final", response.Redirects)
	}
	if response.Timings.Total <= 0 {
		t.Fatalf("total timing = %v, want > 0", response.Timings.Total)
	}
	if response.Timings.FirstByte <= 0 {
		t.Fatalf("first byte timing = %v, want > 0", response.Timings.FirstByte)
	}
	if response.TLS != nil {
		t.Fatalf("TLS info should be nil for plain HTTP, got %+v", response.TLS)
	}
}

func TestTLSInfoFromStateHandlesNil(t *testing.T) {
	if got := tlsInfoFromState(nil); got != nil {
		t.Fatalf("tlsInfoFromState(nil) = %+v, want nil", got)
	}
}
