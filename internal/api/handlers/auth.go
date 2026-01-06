/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/platform-manager/platform-manager/internal/api/middleware"
)

// AuthHandler handles authentication-related endpoints
type AuthHandler struct{}

// NewAuthHandler creates a new auth handler
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// GetCurrentUser returns current user info and capabilities
// GET /api/v1/auth/me
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	capabilities := middleware.GetUserCapabilities(user.Role)
	capStrings := make([]string, len(capabilities))
	for i, cap := range capabilities {
		capStrings[i] = string(cap)
	}

	response := map[string]interface{}{
		"username":     user.Username,
		"email":        user.Email,
		"role":         string(user.Role),
		"groups":       user.Groups,
		"capabilities": capStrings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetCapabilities returns user capabilities
// GET /api/v1/auth/capabilities
func (h *AuthHandler) GetCapabilities(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	capabilities := middleware.GetUserCapabilities(user.Role)

	// Create a map for easier frontend consumption
	capMap := make(map[string]bool)
	for _, cap := range capabilities {
		capMap[string(cap)] = true
	}

	response := map[string]interface{}{
		"role":         string(user.Role),
		"capabilities": capMap,
		"features": map[string]bool{
			"canSyncArgo":            capMap[string(middleware.CapSyncArgo)],
			"canRefreshArgo":         capMap[string(middleware.CapRefreshArgo)],
			"canPauseCrossplane":     capMap[string(middleware.CapPauseCrossplane)],
			"canReconcileCrossplane": capMap[string(middleware.CapReconcileCrossplane)],
			"canDeleteResource":      capMap[string(middleware.CapDeleteResource)],
			"canUseTerminal":         capMap[string(middleware.CapUseTerminal)],
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
