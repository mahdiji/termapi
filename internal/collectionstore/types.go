package collectionstore

import (
	"encoding/json"
	"sort"
	"strings"
)

const folderFile = "_folder.yaml"

// KVRow is one editable key/value entry used by params, headers, form fields,
// cookies and multipart bodies. Disabled is stored instead of Enabled so rows
// without the field default to enabled.
type KVRow struct {
	Key         string `yaml:"key,omitempty"`
	Value       string `yaml:"value,omitempty"`
	Description string `yaml:"description,omitempty"`
	Kind        string `yaml:"kind,omitempty"` // text|file (multipart only)
	Disabled    bool   `yaml:"disabled,omitempty"`
}

func (r KVRow) Enabled() bool { return !r.Disabled }

// BodySpec describes the request body independent of its textual
// representation. Raw is used by json/xml/text/graphql bodies, Fields by
// form/multipart bodies, and File by binary bodies.
type BodySpec struct {
	Type        string  `yaml:"type,omitempty"`
	Raw         string  `yaml:"raw,omitempty"`
	Fields      []KVRow `yaml:"fields,omitempty"`
	File        string  `yaml:"file,omitempty"`
	ContentType string  `yaml:"content_type,omitempty"`
}

// AuthSpec describes how credentials are applied to a request.
type AuthSpec struct {
	Type     string `yaml:"type,omitempty"` // none|bearer|basic|api_key
	Token    string `yaml:"token,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Key      string `yaml:"key,omitempty"`
	Value    string `yaml:"value,omitempty"`
	AddTo    string `yaml:"add_to,omitempty"` // header|query
}

type Request struct {
	ID      string            `yaml:"id"`
	Name    string            `yaml:"name"`
	Method  string            `yaml:"method"`
	URL     string            `yaml:"url"`
	Params  string            `yaml:"params,omitempty"`  // legacy
	Headers map[string]string `yaml:"headers,omitempty"` // legacy
	Body    string            `yaml:"body,omitempty"`    // legacy

	ParamsList  []KVRow  `yaml:"params_list,omitempty"`
	HeadersList []KVRow  `yaml:"headers_list,omitempty"`
	BodySpec    BodySpec `yaml:"body_spec,omitempty"`
	Auth        AuthSpec `yaml:"auth,omitempty"`
	Cookies     []KVRow  `yaml:"cookies,omitempty"`
}

// Normalize converts legacy YAML fields into the structured representation.
// It is intentionally idempotent so it can be called after every load/save.
func (r *Request) Normalize() {
	if len(r.ParamsList) == 0 && strings.TrimSpace(r.Params) != "" {
		for _, line := range strings.FieldsFunc(r.Params, func(c rune) bool { return c == '\n' || c == '&' }) {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				key, value, _ = strings.Cut(line, ":")
			}
			r.ParamsList = append(r.ParamsList, KVRow{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
		}
	}
	if len(r.HeadersList) == 0 && len(r.Headers) > 0 {
		keys := make([]string, 0, len(r.Headers))
		for key := range r.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			r.HeadersList = append(r.HeadersList, KVRow{Key: key, Value: r.Headers[key]})
		}
	}
	if r.BodySpec.Type == "" {
		switch {
		case strings.TrimSpace(r.Body) == "":
			r.BodySpec.Type = "none"
		case json.Valid([]byte(r.Body)):
			r.BodySpec.Type = "json"
			r.BodySpec.Raw = r.Body
		default:
			r.BodySpec.Type = "text"
			r.BodySpec.Raw = r.Body
		}
	}
	if r.Auth.Type == "" {
		r.Auth.Type = "none"
	}
}

// EnabledRows returns enabled entries, ignoring completely empty rows.
func EnabledRows(rows []KVRow) []KVRow {
	out := make([]KVRow, 0, len(rows))
	for _, row := range rows {
		if !row.Enabled() {
			continue
		}
		if strings.TrimSpace(row.Key) == "" && strings.TrimSpace(row.Value) == "" {
			continue
		}
		out = append(out, row)
	}
	return out
}

type FolderMeta struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type Entry struct {
	ID       string
	Name     string
	RelPath  string
	IsFolder bool
	Request  *Request
	Children []Entry
}

type Store struct {
	Root string
}

func New(root string) *Store {
	if strings.TrimSpace(root) == "" {
		root = "collections"
	}
	return &Store{Root: root}
}
