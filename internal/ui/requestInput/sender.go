package requestinput

import (
	"context"
	"net/http"
	"strings"
	"time"

	"api/internal/collectionstore"

	tea "charm.land/bubbletea/v2"
)

const maxResponseBytes = 2 << 20

type RequestStartedMsg struct{}
type SaveRequestedMsg struct{}

// RequestTimings holds a coarse breakdown of how long each request phase took.
type RequestTimings struct {
	DNS       time.Duration
	Connect   time.Duration
	TLS       time.Duration
	FirstByte time.Duration
	Total     time.Duration
}

// TLSInfo is a compact summary of the TLS connection used for a response.
type TLSInfo struct {
	Version    string
	ServerName string
	Issuer     string
	NotAfter   time.Time
}

type ResponseMsg struct {
	Method         string
	URL            string
	RequestName    string
	CollectionName string
	CollectionPath string
	Status         string
	Code           int
	Duration       time.Duration
	Body           string
	RawBody        string
	Headers        []string
	Cookies        []string
	Size           int
	ContentType    string
	Err            string
	Retryable      bool
	Attempts       int
	Redirects      []string
	Timings        RequestTimings
	TLS            *TLSInfo
}

type requestSpec struct {
	Method         string
	URL            string
	Params         []collectionstore.KVRow
	Headers        []collectionstore.KVRow
	Body           collectionstore.BodySpec
	Auth           collectionstore.AuthSpec
	Cookies        []collectionstore.KVRow
	GlobalScope    string
	RequestName    string
	CollectionName string
	CollectionPath string
	ctx            context.Context
}

func (m *Model) sendRequest() tea.Cmd {
	if m == nil {
		return nil
	}
	m.commitActiveEdit()
	m.syncParamsFromURL()
	params, err := m.Params.EffectiveRows()
	if err != nil {
		return failedRequestCmd("params: " + err.Error())
	}
	headers, err := m.Headers.EffectiveRows()
	if err != nil {
		return failedRequestCmd("headers: " + err.Error())
	}
	cookies, err := m.Cookies.EffectiveRows()
	if err != nil {
		return failedRequestCmd("cookies: " + err.Error())
	}
	body := m.Body.Spec()
	auth := m.Auth.Spec()
	urlVal := m.Input.Value()

	replacer := m.variableReplacer()
	urlVal = replacer.Replace(urlVal)
	headers = replaceRows(replacer, headers)
	params = replaceRows(replacer, params)
	cookies = replaceRows(replacer, cookies)
	body.Raw = replacer.Replace(body.Raw)
	body.File = replacer.Replace(body.File)
	body.Fields = replaceRows(replacer, body.Fields)
	auth = replaceAuth(replacer, auth)

	if m.State != nil {
		if tokenVal, ok := m.State.SessionVariables["token"]; ok && tokenVal != "" {
			for i := range headers {
				if strings.Contains(headers[i].Value, "Bearer token") {
					headers[i].Value = strings.ReplaceAll(headers[i].Value, "Bearer token", "Bearer "+tokenVal)
				}
				if strings.Contains(headers[i].Value, "bearer token") {
					headers[i].Value = strings.ReplaceAll(headers[i].Value, "bearer token", "Bearer "+tokenVal)
				}
			}
			if auth.Type == "none" && !hasHeader(headers, "authorization") {
				headers = append(headers, collectionstore.KVRow{Key: "Authorization", Value: "Bearer " + tokenVal})
			}
		}
	}

	globalScope := ""
	if m.State != nil {
		globalScope = m.State.GlobalScope
	}

	ctx, cancel := context.WithCancel(context.Background())
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = cancel

	spec := requestSpec{
		Method:         m.Method,
		URL:            urlVal,
		Params:         params,
		Headers:        headers,
		Body:           body,
		Auth:           auth,
		Cookies:        cookies,
		GlobalScope:    globalScope,
		RequestName:    m.RequestName,
		CollectionName: m.CollectionName,
		CollectionPath: m.CollectionPath,
		ctx:            ctx,
	}
	return tea.Sequence(
		func() tea.Msg { return RequestStartedMsg{} },
		func() tea.Msg { return performRequestWithRetry(spec, 2, true) },
	)
}

func failedRequestCmd(message string) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { return RequestStartedMsg{} },
		func() tea.Msg { return ResponseMsg{Err: message, Attempts: 1} },
	)
}

func (m *Model) variableReplacer() *strings.Replacer {
	replacements := make([]string, 0)
	if m.State == nil {
		return strings.NewReplacer(replacements...)
	}
	for key, val := range m.State.SessionVariables {
		replacements = append(replacements, "{{"+key+"}}", val)
	}
	for key, val := range m.State.EnvVars {
		replacements = append(replacements, "{{"+key+"}}", val)
	}
	return strings.NewReplacer(replacements...)
}

func replaceRows(replacer *strings.Replacer, rows []collectionstore.KVRow) []collectionstore.KVRow {
	out := make([]collectionstore.KVRow, len(rows))
	for i, row := range rows {
		row.Key = replacer.Replace(row.Key)
		row.Value = replacer.Replace(row.Value)
		row.Description = replacer.Replace(row.Description)
		row.Kind = replacer.Replace(row.Kind)
		out[i] = row
	}
	return out
}

func replaceAuth(replacer *strings.Replacer, auth collectionstore.AuthSpec) collectionstore.AuthSpec {
	auth.Token = replacer.Replace(auth.Token)
	auth.Username = replacer.Replace(auth.Username)
	auth.Password = replacer.Replace(auth.Password)
	auth.Key = replacer.Replace(auth.Key)
	auth.Value = replacer.Replace(auth.Value)
	return auth
}

func hasHeader(rows []collectionstore.KVRow, name string) bool {
	for _, row := range rows {
		if !row.Enabled() {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row.Key), name) {
			return true
		}
	}
	return false
}

// CancelRequest aborts the in-flight request, if any.
func (m *Model) CancelRequest() {
	if m == nil || m.cancel == nil {
		return
	}
	m.cancel()
	m.cancel = nil
}

// SendRequest runs the manual request submission.
func (m *Model) SendRequest() tea.Cmd {
	return m.sendRequest()
}

// BatchSpec describes a saved request passed to the collection runner.
type BatchSpec struct {
	Request        collectionstore.Request
	GlobalScope    string
	CollectionName string
	CollectionPath string
}

func (s BatchSpec) internalSpec() requestSpec {
	s.Request.Normalize()
	return requestSpec{
		Method:         s.Request.Method,
		URL:            s.Request.URL,
		Params:         s.Request.ParamsList,
		Headers:        s.Request.HeadersList,
		Body:           s.Request.BodySpec,
		Auth:           s.Request.Auth,
		Cookies:        s.Request.Cookies,
		GlobalScope:    s.GlobalScope,
		RequestName:    s.Request.Name,
		CollectionName: s.CollectionName,
		CollectionPath: s.CollectionPath,
	}
}

// RunBatchRequest executes one saved request using the same safety checks as the editor.
func RunBatchRequest(spec BatchSpec) ResponseMsg {
	return performRequest(spec.internalSpec()).(ResponseMsg)
}

// RunBatchRequestWithRetry retries transient transport failures and selected HTTP statuses.
func RunBatchRequestWithRetry(spec BatchSpec, retries int) ResponseMsg {
	return performRequestWithRetry(spec.internalSpec(), retries, false)
}

func performRequestWithRetry(spec requestSpec, retries int, idempotentOnly bool) ResponseMsg {
	for attempt := 1; ; attempt++ {
		response := performRequest(spec).(ResponseMsg)
		response.Attempts = attempt
		if idempotentOnly && !isIdempotentMethod(spec.Method) {
			return response
		}
		shouldRetry := response.Retryable || response.Code == 408 || response.Code == 429 || response.Code >= 500
		if !shouldRetry || attempt > retries {
			return response
		}
		delay := time.Duration(500*(1<<min(attempt-1, 4))) * time.Millisecond
		time.Sleep(delay)
	}
}

func isIdempotentMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}
