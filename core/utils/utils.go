package utils

import (
	"net/http"
	"strings"
)

// Pointer pointer
func Pointer[Value any](v Value) *Value {
	return &v
}

// GetIP get the client's ip address
func GetIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ip := strings.Split(xff, ",")[0]
		if ip != "" {
			return ip
		}
	}

	remoteAddr := r.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}

	return remoteAddr
}

// GetUserAgent get the client's user-agent
func GetUserAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}
