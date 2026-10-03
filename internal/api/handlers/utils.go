/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package handlers

import (
	"encoding/json"
	"net/http"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var handlersLog = logf.Log.WithName("api-handlers")

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// Headers are already sent, so all we can do with an encode error is log it.
	if err := json.NewEncoder(w).Encode(data); err != nil {
		handlersLog.Error(err, "Failed to encode JSON response")
	}
}

// queryFlag reports whether a boolean query parameter is set to "true"
func queryFlag(r *http.Request, name string) bool {
	return r.URL.Query().Get(name) == "true"
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, ErrorResponse{Error: message})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
