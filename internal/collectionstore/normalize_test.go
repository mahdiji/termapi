package collectionstore

import "testing"

func TestNormalizeLegacyRequest(t *testing.T) {
	req := Request{
		Params:  "page=1&limit=20\nsort=name",
		Headers: map[string]string{"X-Trace": "abc", "Authorization": "Bearer token"},
		Body:    `{"name":"api"}`,
	}
	req.Normalize()
	if len(req.ParamsList) != 3 {
		t.Fatalf("expected 3 params, got %d", len(req.ParamsList))
	}
	if req.ParamsList[0].Key != "page" || req.ParamsList[0].Value != "1" {
		t.Fatalf("unexpected first param: %+v", req.ParamsList[0])
	}
	if len(req.HeadersList) != 2 || req.HeadersList[0].Key != "Authorization" {
		t.Fatalf("unexpected headers: %+v", req.HeadersList)
	}
	if req.BodySpec.Type != "json" || req.BodySpec.Raw != req.Body {
		t.Fatalf("unexpected body spec: %+v", req.BodySpec)
	}
	if req.Auth.Type != "none" {
		t.Fatalf("expected auth none, got %q", req.Auth.Type)
	}

	req.Normalize()
	if len(req.ParamsList) != 3 || len(req.HeadersList) != 2 {
		t.Fatal("normalize should be idempotent")
	}
}

func TestSaveLoadStructuredRequest(t *testing.T) {
	store := New(t.TempDir())
	folder, err := store.CreateFolder("", "Api", "collection")
	if err != nil {
		t.Fatal(err)
	}
	req := Request{
		Name:        "Create User",
		Method:      "post",
		URL:         "https://api.example.com/users/:id",
		ParamsList:  []KVRow{{Key: "id", Value: "42"}, {Key: "page", Value: "2"}},
		HeadersList: []KVRow{{Key: "Content-Type", Value: "application/json"}},
		BodySpec:    BodySpec{Type: "json", Raw: `{"name":"Ada"}`},
		Auth:        AuthSpec{Type: "bearer", Token: "{{token}}"},
		Cookies:     []KVRow{{Key: "session", Value: "abc"}},
	}
	entry, err := store.SaveRequest(folder.RelPath, "", req)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Request.Params != "" || entry.Request.Headers != nil || entry.Request.Body != "" {
		t.Fatal("legacy fields should be cleared when saving")
	}

	entries, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(entries[0].Children) != 1 {
		t.Fatalf("unexpected tree: %+v", entries)
	}
	got := entries[0].Children[0].Request
	if got == nil {
		t.Fatal("request was not loaded")
	}
	if len(got.ParamsList) != 2 || got.ParamsList[0].Key != "id" {
		t.Fatalf("unexpected params: %+v", got.ParamsList)
	}
	if len(got.HeadersList) != 1 || got.HeadersList[0].Value != "application/json" {
		t.Fatalf("unexpected headers: %+v", got.HeadersList)
	}
	if got.BodySpec.Type != "json" || got.BodySpec.Raw != `{"name":"Ada"}` {
		t.Fatalf("unexpected body: %+v", got.BodySpec)
	}
	if got.Auth.Type != "bearer" || got.Auth.Token != "{{token}}" {
		t.Fatalf("unexpected auth: %+v", got.Auth)
	}
	if len(got.Cookies) != 1 || got.Cookies[0].Value != "abc" {
		t.Fatalf("unexpected cookies: %+v", got.Cookies)
	}
}
