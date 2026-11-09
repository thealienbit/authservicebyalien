package http

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// ParseTrustedProxies accepts a comma-separated list of IPs or CIDRs. An empty
// string means no proxy is trusted and X-Forwarded-For is ignored.
func ParseTrustedProxies(raw string) ([]*net.IPNet, error) {
	var networks []*net.IPNet

	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if !strings.Contains(part, "/") {
			ip := net.ParseIP(part)
			if ip == nil {
				return nil, fmt.Errorf("invalid trusted proxy %q", part)
			}
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			part = fmt.Sprintf("%s/%d", part, bits)
		}

		_, network, err := net.ParseCIDR(part)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy %q: %w", part, err)
		}
		networks = append(networks, network)
	}

	return networks, nil
}

// ClientIP returns the end-client IP. X-Forwarded-For is only consulted when
// the immediate peer is a trusted proxy; the header is then walked right to
// left, skipping trusted hops, so a client cannot spoof its own address.
func ClientIP(r *http.Request, trusted []*net.IPNet) string {
	remote := hostOnly(r.RemoteAddr)

	if !ipInNetworks(remote, trusted) {
		return remote
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return remote
	}

	parts := strings.Split(xff, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(parts[i])
		if candidate == "" || ipInNetworks(candidate, trusted) {
			continue
		}
		return candidate
	}

	return remote
}

func ipInNetworks(ipString string, networks []*net.IPNet) bool {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return false
	}

	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func hostOnly(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
