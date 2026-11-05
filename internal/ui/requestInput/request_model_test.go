package requestinput

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api/internal/collectionstore"
	"api/internal/model"
)

func TestKVEditorRawRoundTrip(t *testing.T) {
	k := NewKVEditor("params", "params", true, true)
	k.SetRows([]collectionstore.KVRow{
		{Key: "page", Value: "1"},
		{Key: "token", Value: "secret", Disabled: true},
	})
	if err := k.ToggleRaw(); err != nil {
		t.Fatal(err)
	}
	if !k.IsRaw() {
		t.Fatal("expected raw mode")
	}
	if !strings.Contains(k.rawInput.Value(), "# token=secret") {
		t.Fatalf("disabled row was not serialized: %q", k.rawInput.Value())
	}
	k.rawInput.SetValue("page=2\nlimit=10\n# debug=1")
	if err := k.ToggleRaw(); err != nil {
		t.Fatal(err)
	}
	rows, err := k.EffectiveRows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Value != "2" || !rows[2].Disabled {
		t.Fatalf("unexpected parsed rows: %+v", rows)
	}
}

func TestKVEditorRejectsInvalidRaw(t *testing.T) {
	k := NewKVEditor("headers", "headers", true, true)
	if err := k.ToggleRaw(); err != nil {
		t.Fatal(err)
	}
	k.rawInput.SetValue("missing-colon")
	if err := k.ToggleRaw(); err == nil {
		t.Fatal("expected invalid raw headers to fail")
	}
	if !k.IsRaw() {
		t.Fatal("editor should stay in raw mode when parsing fails")
	}
}

func TestSnapshotLoadRequestRoundTrip(t *testing.T) {
	m := New(model.NewSharedState())
	m.Method = "POST"
	m.SetValue("https://api.example.com/users")
	m.Params.SetRows([]collectionstore.KVRow{{Key: "page", Value: "2"}})
	m.Headers.SetRows([]collectionstore.KVRow{{Key: "Accept", Value: "application/json"}})
	m.Body.SetSpec(collectionstore.BodySpec{Type: "json", Raw: `{"name":"Ada"}`})
	m.Auth.SetSpec(collectionstore.AuthSpec{Type: "bearer", Token: "{{token}}"})
	m.Cookies.SetRows([]collectionstore.KVRow{{Key: "session", Value: "abc"}})

	req, err := m.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	next := New(model.NewSharedState())
	next.LoadRequest(&req)
	if next.Method != "POST" || next.Input.Value() != "https://api.example.com/users" {
		t.Fatalf("method/url not restored: %s %s", next.Method, next.Input.Value())
	}
	if len(next.Params.Rows) != 1 || next.Params.Rows[0].Value != "2" {
		t.Fatalf("params not restored: %+v", next.Params.Rows)
	}
	if next.Body.Type != "json" || next.Body.Raw.Value() != `{"name":"Ada"}` {
		t.Fatalf("body not restored: %+v", next.Body.Spec())
	}
	if next.Auth.Type != "bearer" || next.Auth.values["token"] != "{{token}}" {
		t.Fatalf("auth not restored: %+v", next.Auth.Spec())
	}
	if len(next.Cookies.Rows) != 1 || next.Cookies.Rows[0].Key != "session" {
		t.Fatalf("cookies not restored: %+v", next.Cookies.Rows)
	}
}

func TestSyncParamsFromURL(t *testing.T) {
	m := New(model.NewSharedState())
	m.SetValue("https://api.example.com/users?page=2&q=ada")
	m.Params.SetRows([]collectionstore.KVRow{{Key: "page", Value: ""}})
	m.syncParamsFromURL()
	if strings.Contains(m.Input.Value(), "?") {
		t.Fatalf("query should move out of the URL bar: %q", m.Input.Value())
	}
	rows, err := m.Params.EffectiveRows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Value != "2" || rows[1].Key != "q" || rows[1].Value != "ada" {
		t.Fatalf("unexpected params: %+v", rows)
	}
}

func TestBuildRequestURLPathVariablesAndQuery(t *testing.T) {
	target, err := buildRequestURL("https://api.example.com/users/:id/items/{item}", []collectionstore.KVRow{
		{Key: "id", Value: "42"},
		{Key: "item", Value: "books"},
		{Key: "page", Value: "2"},
		{Key: "debug", Value: "1", Disabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://api.example.com/users/42/items/books?page=2"
	if target != want {
		t.Fatalf("got %q, want %q", target, want)
	}
}

func TestBuildRequestBodyFormAndMultipart(t *testing.T) {
	reader, contentType, err := buildRequestBody(requestSpec{
		Method: "POST",
		Body: collectionstore.BodySpec{Type: "form", Fields: []collectionstore.KVRow{
			{Key: "name", Value: "Ada"},
			{Key: "tag", Value: "go"},
			{Key: "tag", Value: "api"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(reader)
	if contentType != "application/x-www-form-urlencoded" {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	if string(data) != "name=Ada&tag=go&tag=api" {
		t.Fatalf("unexpected form body: %s", data)
	}

	reader, contentType, err = buildRequestBody(requestSpec{
		Method: "POST",
		Body: collectionstore.BodySpec{Type: "multipart", Fields: []collectionstore.KVRow{
			{Key: "name", Value: "Ada"},
			{Key: "avatar", Value: "/tmp/does-not-exist", Kind: "file"},
		}},
	})
	if err == nil {
		t.Fatal("expected missing multipart file error")
	}
	if reader != nil || contentType != "" {
		t.Fatal("expected empty body on multipart error")
	}
}

func TestPerformRequestAppliesAuthCookiesAndJSON(t *testing.T) {
	var gotAuth, gotCookie, gotBody, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		gotMethod = r.Method
		data, _ := io.ReadAll(r.Body)
		gotBody = string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	response := performRequest(requestSpec{
		Method:      "POST",
		URL:         server.URL + "/users",
		GlobalScope: server.URL,
		Params:      []collectionstore.KVRow{{Key: "page", Value: "2"}},
		Headers:     []collectionstore.KVRow{{Key: "Accept", Value: "application/json"}},
		Body:        collectionstore.BodySpec{Type: "json", Raw: `{"name":"Ada"}`},
		Auth:        collectionstore.AuthSpec{Type: "bearer", Token: "abc123"},
		Cookies:     []collectionstore.KVRow{{Key: "session", Value: "xyz"}},
	}).(ResponseMsg)
	if response.Err != "" {
		t.Fatal(response.Err)
	}
	if response.Code != 200 {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if gotMethod != "POST" {
		t.Fatalf("unexpected method: %s", gotMethod)
	}
	if gotAuth != "Bearer abc123" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if gotCookie != "session=xyz" {
		t.Fatalf("unexpected cookie header: %q", gotCookie)
	}
	if gotBody != `{"name":"Ada"}` {
		t.Fatalf("unexpected body: %q", gotBody)
	}
	if !bytes.Contains([]byte(response.RawBody), []byte(`"ok":true`)) {
		t.Fatalf("unexpected response: %q", response.RawBody)
	}
}
