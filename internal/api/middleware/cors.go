/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
	DevMode        bool
}

// DefaultCORSConfig returns default CORS configuration from environment
func DefaultCORSConfig() CORSConfig {
	allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := []string{}
	if allowedOriginsStr != "" {
		for _, origin := range strings.Split(allowedOriginsStr, ",") {
			trimmed := strings.TrimSpace(origin)
			if trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}

	return CORSConfig{
		AllowedOrigins: allowedOrigins,
		DevMode:        os.Getenv("DEV_MODE") == "true",
	}
}

// CORSWithConfig adds CORS headers to responses with configuration
func CORSWithConfig(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// In dev mode, allow all origins
			if config.DevMode {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if isAllowedOrigin(origin, config.AllowedOrigins) {
				// In production, only allow configured origins
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Auth-Request-User, X-Auth-Request-Email, X-Auth-Request-Groups")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "300")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS adds CORS headers to responses for development (legacy function, now uses config)
func CORS(next http.Handler) http.Handler {
	config := DefaultCORSConfig()
	return CORSWithConfig(config)(next)
}

// isAllowedOrigin checks if an origin is in the allowed list
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
