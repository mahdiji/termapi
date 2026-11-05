package requestinput

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"api/internal/collectionstore"
	"api/internal/config"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/net/http/httpguts"
)

var sessionJar, _ = cookiejar.New(nil)

func performRequest(spec requestSpec) tea.Msg {
	started := time.Now()
	result := ResponseMsg{Method: spec.Method, URL: spec.URL, RequestName: spec.RequestName, CollectionName: spec.CollectionName, CollectionPath: spec.CollectionPath}

	target, err := buildRequestURL(spec.URL, spec.Params)
	if err != nil {
		result.Err = err.Error()
		result.Duration = time.Since(started)
		return result
	}
	target, err = applyQueryAuth(target, spec.Auth)
	if err != nil {
		result.Err = err.Error()
		result.Duration = time.Since(started)
		return result
	}
	result.URL = target
	globalScope := spec.GlobalScope
	if globalScope == "" {
		if gs, err := config.LoadGlobalScope(""); err == nil {
			globalScope = gs
		}
	}
	allowedOrigins, err := parseAllowedOrigins(globalScope)
	if err != nil {
		result.Err = err.Error()
		result.Duration = time.Since(started)
		return result
	}
	requestURL, _ := url.Parse(target)
	needsScope := hasSensitiveHeaders(spec.Headers) ||
		hasSensitiveQuery(requestURL) ||
		len(collectionstore.EnabledRows(spec.Cookies)) > 0 ||
		(spec.Auth.Type != "" && spec.Auth.Type != "none")
	if needsScope {
		origin, originErr := normalizedOrigin(requestURL)
		if originErr != nil {
			result.Err = originErr.Error()
			result.Duration = time.Since(started)
			return result
		}
		if _, allowed := allowedOrigins[origin]; !allowed {
			result.Err = "credentials blocked: add the exact request origin to .termapi/global.yaml"
			result.Duration = time.Since(started)
			return result
		}
	}

	body, defaultContentType, err := buildRequestBody(spec)
	if err != nil {
		result.Err = err.Error()
		result.Duration = time.Since(started)
		return result
	}
	ctx := spec.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, spec.Method, target, body)
	if err != nil {
		result.Err = err.Error()
		result.Duration = time.Since(started)
		return result
	}
	var dnsStart, connectStart, tlsStart time.Time
	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				result.Timings.DNS += time.Since(dnsStart)
			}
		},
		ConnectStart: func(string, string) { connectStart = time.Now() },
		ConnectDone: func(string, string, error) {
			if !connectStart.IsZero() {
				result.Timings.Connect += time.Since(connectStart)
			}
		},
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tlsStart.IsZero() {
				result.Timings.TLS += time.Since(tlsStart)
			}
		},
		GotFirstResponseByte: func() { result.Timings.FirstByte = time.Since(started) },
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	for _, row := range collectionstore.EnabledRows(spec.Headers) {
		name := strings.TrimSpace(row.Key)
		value := strings.TrimSpace(row.Value)
		if name == "" {
			continue
		}
		if !httpguts.ValidHeaderFieldName(name) || !httpguts.ValidHeaderFieldValue(value) {
			result.Err = fmt.Sprintf("invalid header line: %q", name+": "+value)
			result.Duration = time.Since(started)
			return result
		}
		req.Header.Add(textproto.CanonicalMIMEHeaderKey(name), value)
	}
	applyHeaderAuth(req, spec.Auth)
	applyCookies(req, spec.Cookies)
	if defaultContentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", defaultContentType)
	}

	client := &http.Client{Timeout: 30 * time.Second, Jar: sessionJar}
	client.CheckRedirect = func(redirect *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		if needsScope {
			origin, originErr := normalizedOrigin(redirect.URL)
			if originErr != nil {
				return originErr
			}
			if _, allowed := allowedOrigins[origin]; !allowed {
				return fmt.Errorf("redirect blocked: destination origin is outside the Scope allowlist")
			}
		}
		result.Redirects = append(result.Redirects, redirect.URL.String())
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		result.Err = err.Error()
		result.Retryable = isTransientTransportError(err)
		result.Duration = time.Since(started)
		return result
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		result.Err = fmt.Sprintf("read response body: %v", err)
		result.Retryable = true
		result.Duration = time.Since(started)
		return result
	}
	result.Duration = time.Since(started)
	result.Timings.Total = result.Duration
	result.Size = len(data)
	if len(data) > maxResponseBytes {
		data = data[:maxResponseBytes]
		data = append(data, []byte("\n\n[response truncated at 2 MiB]")...)
	}
	result.Code = resp.StatusCode
	result.Status = resp.Status
	result.RawBody = string(data)
	result.Body = formatResponseBody(data)
	result.ContentType = resp.Header.Get("Content-Type")
	if resp.TLS != nil {
		result.TLS = tlsInfoFromState(resp.TLS)
	}
	for name, values := range resp.Header {
		for _, value := range values {
			result.Headers = append(result.Headers, name+": "+value)
		}
	}
	sort.Strings(result.Headers)
	for _, cookie := range resp.Cookies() {
		result.Cookies = append(result.Cookies, cookie.Name+"="+cookie.Value)
	}
	return result
}

func buildRequestBody(spec requestSpec) (io.Reader, string, error) {
	if spec.Method == http.MethodGet || spec.Method == http.MethodHead {
		return nil, "", nil
	}
	body := spec.Body
	contentType := body.ContentType
	switch body.Type {
	case "", "none":
		return nil, "", nil
	case "json", "graphql":
		if strings.TrimSpace(body.Raw) == "" {
			return nil, "", nil
		}
		if contentType == "" {
			contentType = "application/json"
		}
		return strings.NewReader(body.Raw), contentType, nil
	case "xml":
		if strings.TrimSpace(body.Raw) == "" {
			return nil, "", nil
		}
		if contentType == "" {
			contentType = "application/xml"
		}
		return strings.NewReader(body.Raw), contentType, nil
	case "text":
		if body.Raw == "" {
			return nil, "", nil
		}
		if contentType == "" {
			contentType = "text/plain; charset=utf-8"
		}
		return strings.NewReader(body.Raw), contentType, nil
	case "form":
		values := url.Values{}
		for _, row := range collectionstore.EnabledRows(body.Fields) {
			key := strings.TrimSpace(row.Key)
			if key == "" {
				continue
			}
			values.Add(key, row.Value)
		}
		if contentType == "" {
			contentType = "application/x-www-form-urlencoded"
		}
		return strings.NewReader(values.Encode()), contentType, nil
	case "multipart":
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		for _, row := range collectionstore.EnabledRows(body.Fields) {
			key := strings.TrimSpace(row.Key)
			if key == "" {
				continue
			}
			if row.Kind == "file" {
				path := strings.TrimSpace(row.Value)
				if path == "" {
					continue
				}
				file, err := os.Open(path)
				if err != nil {
					return nil, "", fmt.Errorf("multipart file %q: %w", path, err)
				}
				part, err := writer.CreateFormFile(key, filepath.Base(path))
				if err == nil {
					_, err = io.Copy(part, file)
				}
				closeErr := file.Close()
				if err != nil {
					return nil, "", fmt.Errorf("multipart file %q: %w", path, err)
				}
				if closeErr != nil {
					return nil, "", fmt.Errorf("multipart file %q: %w", path, closeErr)
				}
				continue
			}
			if err := writer.WriteField(key, row.Value); err != nil {
				return nil, "", fmt.Errorf("multipart field %q: %w", key, err)
			}
		}
		if err := writer.Close(); err != nil {
			return nil, "", fmt.Errorf("multipart body: %w", err)
		}
		if contentType == "" {
			contentType = writer.FormDataContentType()
		}
		return bytes.NewReader(buf.Bytes()), contentType, nil
	case "binary":
		path := strings.TrimSpace(body.File)
		if path == "" {
			return nil, "", nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("binary body %q: %w", path, err)
		}
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		return bytes.NewReader(data), contentType, nil
	default:
		if body.Raw == "" {
			return nil, "", nil
		}
		return strings.NewReader(body.Raw), contentType, nil
	}
}

func applyQueryAuth(target string, auth collectionstore.AuthSpec) (string, error) {
	if auth.Type != "api_key" || auth.AddTo != "query" || strings.TrimSpace(auth.Key) == "" {
		return target, nil
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Add(strings.TrimSpace(auth.Key), auth.Value)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func applyHeaderAuth(req *http.Request, auth collectionstore.AuthSpec) {
	switch auth.Type {
	case "bearer":
		if auth.Token != "" && req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "Bearer "+auth.Token)
		}
	case "basic":
		if (auth.Username != "" || auth.Password != "") && req.Header.Get("Authorization") == "" {
			token := base64.StdEncoding.EncodeToString([]byte(auth.Username + ":" + auth.Password))
			req.Header.Set("Authorization", "Basic "+token)
		}
	case "api_key":
		if auth.AddTo != "query" && strings.TrimSpace(auth.Key) != "" && req.Header.Get(auth.Key) == "" {
			req.Header.Set(strings.TrimSpace(auth.Key), auth.Value)
		}
	}
}

func applyCookies(req *http.Request, cookies []collectionstore.KVRow) {
	pairs := make([]string, 0, len(cookies))
	for _, row := range collectionstore.EnabledRows(cookies) {
		key := strings.TrimSpace(row.Key)
		if key == "" {
			continue
		}
		pairs = append(pairs, key+"="+row.Value)
	}
	if len(pairs) == 0 {
		return
	}
	value := strings.Join(pairs, "; ")
	if existing := req.Header.Get("Cookie"); existing != "" {
		value = existing + "; " + value
	}
	req.Header.Set("Cookie", value)
}

func buildRequestURL(raw string, params []collectionstore.KVRow) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("URL is empty")
	}
	if !strings.Contains(raw, "://") {
		if strings.HasPrefix(raw, "localhost") || strings.HasPrefix(raw, "127.0.0.1") || strings.HasPrefix(raw, "[::1]") {
			raw = "http://" + raw
		} else {
			raw = "https://" + raw
		}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid URL: missing host")
	}

	used := make([]bool, len(params))
	path := u.Path
	for i, row := range params {
		if !row.Enabled() {
			continue
		}
		key := strings.TrimSpace(row.Key)
		if key == "" {
			if strings.TrimSpace(row.Value) != "" {
				return "", fmt.Errorf("invalid param: key is empty")
			}
			continue
		}
		braced := "{" + key + "}"
		if strings.Contains(path, braced) {
			path = strings.ReplaceAll(path, braced, row.Value)
			used[i] = true
		}
	}
	segments := strings.Split(path, "/")
	for i, row := range params {
		if used[i] || !row.Enabled() {
			continue
		}
		key := strings.TrimSpace(row.Key)
		if key == "" {
			continue
		}
		for j, segment := range segments {
			if strings.HasPrefix(segment, ":") && strings.TrimPrefix(segment, ":") == key {
				segments[j] = row.Value
				used[i] = true
			}
		}
	}
	u.Path = strings.Join(segments, "/")

	query := u.Query()
	for i, row := range params {
		if used[i] || !row.Enabled() {
			continue
		}
		key := strings.TrimSpace(row.Key)
		if key == "" {
			continue
		}
		query.Add(key, row.Value)
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func hasSensitiveHeaders(rows []collectionstore.KVRow) bool {
	for _, row := range collectionstore.EnabledRows(rows) {
		name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(row.Key), "-", ""))
		if strings.Contains(name, "auth") || strings.Contains(name, "cookie") || strings.Contains(name, "token") || strings.Contains(name, "secret") || strings.Contains(name, "password") || strings.Contains(name, "credential") || strings.Contains(name, "session") || strings.Contains(name, "apikey") || strings.HasSuffix(name, "key") {
			return true
		}
	}
	return false
}

func tlsInfoFromState(state *tls.ConnectionState) *TLSInfo {
	if state == nil {
		return nil
	}
	version := tls.VersionName(state.Version)
	if version == "" {
		version = fmt.Sprintf("0x%x", state.Version)
	}
	info := &TLSInfo{
		Version:    version,
		ServerName: state.ServerName,
	}
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		info.Issuer = cert.Issuer.String()
		info.NotAfter = cert.NotAfter
	}
	return info
}
