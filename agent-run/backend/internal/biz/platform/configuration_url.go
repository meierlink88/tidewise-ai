package agentrun

import (
	"net"
	"net/url"
	"strings"
)

// ConfigurationBaseURLValid applies the URL policy shared by Admin writes and
// Agent execution readiness.
func ConfigurationBaseURLValid(raw, environment string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return true
	case "http":
		if environment != "dev" {
			return false
		}
		host := parsed.Hostname()
		if strings.EqualFold(host, "localhost") {
			return true
		}
		address := net.ParseIP(host)
		return address != nil && address.IsLoopback()
	default:
		return false
	}
}
