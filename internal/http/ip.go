package http

import (
	"net"
	"net/http"
	"strings"
)

func ClientIP(r *http.Request) string {
	// Trust X-Forwarded-For ONLY if behind a trusted proxy
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

