package requestinput

import (
	"net/url"
	"strings"

	"api/internal/collectionstore"
)

// syncParamsFromURL moves the query string from the URL bar into the Params
// table once. Keeping a single source avoids duplicate query parameters when
// a request is sent.
func (m *Model) syncParamsFromURL() {
	if m == nil || m.Params.IsRaw() {
		return
	}
	raw := strings.TrimSpace(m.Input.Value())
	if raw == "" || !strings.Contains(raw, "?") {
		return
	}
	if !strings.Contains(raw, "://") {
		if strings.HasPrefix(raw, "localhost") || strings.HasPrefix(raw, "127.0.0.1") || strings.HasPrefix(raw, "[::1]") {
			raw = "http://" + raw
		} else {
			raw = "https://" + raw
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery == "" {
		return
	}
	rows, err := m.Params.EffectiveRows()
	if err != nil {
		return
	}
	for _, pair := range strings.Split(u.RawQuery, "&") {
		if pair == "" {
			continue
		}
		key, value, _ := strings.Cut(pair, "=")
		key, _ = url.QueryUnescape(key)
		value, _ = url.QueryUnescape(value)
		if key == "" {
			continue
		}
		merged := false
		for i := range rows {
			if rows[i].Key == key && rows[i].Value == "" {
				rows[i].Value = value
				merged = true
				break
			}
		}
		if !merged {
			rows = append(rows, collectionstore.KVRow{Key: key, Value: value})
		}
	}
	u.RawQuery = ""
	m.Input.SetValue(u.String())
	m.Params.SetRows(rows)
}
