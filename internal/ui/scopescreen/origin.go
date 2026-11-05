package scopescreen

import (
	"net"
	"net/url"
	"strings"
)

// OriginInfo adalah hasil parsing satu origin pada Global Scope.
// Nilai nol (Valid=false) berarti origin tidak bisa dipakai.
type OriginInfo struct {
	Raw    string
	Valid  bool
	Scheme string
	Host   string
	Port   string
	Path   string
	Type   string // LOCAL, LAN, PUBLIC, atau UNKNOWN
	Risk   string // LOW, MEDIUM, atau HIGH
}

// parseOrigin memvalidasi dan memecah origin menjadi komponen yang
// ditampilkan pada panel detail.
func parseOrigin(raw string) OriginInfo {
	info := OriginInfo{
		Raw:    raw,
		Scheme: "—",
		Host:   "—",
		Port:   "—",
		Path:   "—",
		Type:   "UNKNOWN",
		Risk:   "—",
	}

	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return info
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return info
	}
	host := u.Hostname()
	if host == "" {
		return info
	}

	info.Valid = true
	info.Scheme = scheme
	info.Host = host
	info.Port = u.Port()
	if info.Port == "" {
		if scheme == "https" {
			info.Port = "443"
		} else {
			info.Port = "80"
		}
	}
	info.Path = u.EscapedPath()
	if info.Path == "" {
		info.Path = "/"
	}
	info.Type = hostType(host)
	info.Risk = originRisk(scheme, info.Type)
	return info
}

// hostType mengklasifikasikan host menjadi LOCAL, LAN, atau PUBLIC.
func hostType(host string) string {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") || h == "::1" {
		return "LOCAL"
	}
	if ip := net.ParseIP(h); ip != nil {
		switch {
		case ip.IsLoopback():
			return "LOCAL"
		case ip.IsPrivate(), ip.IsLinkLocalUnicast():
			return "LAN"
		default:
			return "PUBLIC"
		}
	}
	if strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".internal") {
		return "LAN"
	}
	return "PUBLIC"
}

// originRisk menilai risiko pengiriman kredensial ke origin tersebut.
func originRisk(scheme, hostType string) string {
	if scheme == "https" {
		return "LOW"
	}
	if hostType == "PUBLIC" {
		return "HIGH"
	}
	return "MEDIUM"
}

// ScopeStats merangkum isi daftar origin untuk header dan panel informasi.
type ScopeStats struct {
	Total   int
	HTTPS   int
	HTTP    int
	Local   int
	Public  int
	Invalid int
}

func buildStats(origins []string) ScopeStats {
	var stats ScopeStats
	stats.Total = len(origins)
	for _, raw := range origins {
		info := parseOrigin(raw)
		if !info.Valid {
			stats.Invalid++
			continue
		}
		if info.Scheme == "https" {
			stats.HTTPS++
		} else {
			stats.HTTP++
		}
		if info.Type == "LOCAL" || info.Type == "LAN" {
			stats.Local++
		} else {
			stats.Public++
		}
	}
	return stats
}
