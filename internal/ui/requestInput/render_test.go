package requestinput

import (
	"strings"
	"testing"

	"api/internal/collectionstore"
	"api/internal/model"
	zone "github.com/lrstanley/bubblezone/v2"
)

func TestAllRequestTabsRender(t *testing.T) {
	zone.NewGlobal()
	m := New(model.NewSharedState())
	m.SetWidth(90)
	m.SetEditorSize(90, 8)
	m.Params.SetRows([]collectionstore.KVRow{{Key: "page", Value: "1", Description: "halaman"}})
	m.Headers.SetRows([]collectionstore.KVRow{{Key: "Accept", Value: "application/json"}})
	m.Cookies.SetRows([]collectionstore.KVRow{{Key: "session", Value: "abc"}})
	m.Auth.SetSpec(collectionstore.AuthSpec{Type: "basic", Username: "ada", Password: "secret"})

	bodyTypesToCheck := []string{"none", "json", "form", "multipart", "binary", "graphql"}
	for _, tab := range []int{0, 1, 2, 3, 4} {
		m.Tab = tab
		for _, bodyType := range bodyTypesToCheck {
			m.Body.SetSpec(collectionstore.BodySpec{Type: bodyType, Raw: `{"a":1}`, File: "payload.bin", Fields: []collectionstore.KVRow{{Key: "name", Value: "Ada", Kind: "text"}}})
			view := m.EditorView()
			if strings.TrimSpace(view) == "" {
				t.Fatalf("empty editor view for tab %d body %s", tab, bodyType)
			}
		}
	}
}
