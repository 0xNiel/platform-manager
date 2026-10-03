/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func preflight(t *testing.T, config CORSConfig, origin string) *httptest.ResponseRecorder {
	t.Helper()
	handler := CORSWithConfig(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("preflight should not reach the next handler")
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/tenants", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "x-dev-role")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCORS_DevModeAllowsDevRoleAndEchoesOrigin(t *testing.T) {
	rec := preflight(t, CORSConfig{DevMode: true}, "http://localhost:9082")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:9082" {
		t.Errorf("Allow-Origin = %q, want the request origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "X-Dev-Role") {
		t.Errorf("Allow-Headers = %q, want it to include X-Dev-Role", got)
	}
}

func TestCORS_ProductionRejectsDevRoleAndUnknownOrigins(t *testing.T) {
	config := CORSConfig{AllowedOrigins: []string{"https://portal.example.com"}}

	rec := preflight(t, config, "https://portal.example.com")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://portal.example.com" {
		t.Errorf("Allow-Origin = %q, want the allowed origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); strings.Contains(got, "X-Dev-Role") {
		t.Errorf("Allow-Headers = %q, must not include X-Dev-Role outside dev mode", got)
	}

	rec = preflight(t, config, "https://evil.example.com")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want empty for an unlisted origin", got)
	}
}
