package middleware

import (
	"net"
	"net/http"
	"strings"
)

type KeyFunc func(r *http.Request) string

func DefaultKeyByIp(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		parts := strings.Split(f, ",")

		return strings.TrimSpace(parts[0])
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	if ip == "" {
		return "global"
	}
	return ip
}
