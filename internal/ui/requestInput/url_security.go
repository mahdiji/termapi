package requestinput

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"syscall"
)

func isTransientTransportError(err error) bool {
	if err == nil {
		return false
	}
	var networkError net.Error
	if errors.As(err, &networkError) && (networkError.Timeout() || networkError.Temporary()) {
		return true
	}
	return errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ENETUNREACH) || errors.Is(err, syscall.EHOSTUNREACH)
}

func parseAllowedOrigins(raw string) (map[string]struct{}, error) {
	allowed := make(map[string]struct{})
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		u, err := url.Parse(line)
		if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return nil, fmt.Errorf("invalid scope origin %q; enter scheme://host[:port] with no path", line)
		}
		origin, err := normalizedOrigin(u)
		if err != nil {
			return nil, fmt.Errorf("invalid scope origin %q: %w", line, err)
		}
		allowed[origin] = struct{}{}
	}
	return allowed, nil
}

func normalizedOrigin(u *url.URL) (string, error) {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Hostname() == "" {
		return "", fmt.Errorf("only http and https origins are supported")
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return scheme + "://" + host, nil
}

func hasSensitiveQuery(u *url.URL) bool {
	if u == nil {
		return false
	}
	for name := range u.Query() {
		name = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, "-", ""), "_", ""))
		if strings.Contains(name, "auth") || strings.Contains(name, "token") || strings.Contains(name, "secret") || strings.Contains(name, "password") || strings.Contains(name, "credential") || strings.Contains(name, "session") || strings.Contains(name, "apikey") || strings.HasSuffix(name, "key") {
			return true
		}
	}
	return false
}

func formatResponseBody(data []byte) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "(empty response body)"
	}
	if json.Valid(trimmed) {
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, trimmed, "", "  "); err == nil {
			return formatted.String()
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\t", "    ")
}
