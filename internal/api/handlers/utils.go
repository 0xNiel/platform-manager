/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package handlers

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, ErrorResponse{Error: message})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
