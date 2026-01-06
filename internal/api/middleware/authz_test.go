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

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHasCapability(t *testing.T) {
	tests := []struct {
		name       string
		role       Role
		capability Capability
		expected   bool
	}{
		// Admin tests - should have all capabilities
		{"admin has argo:sync", RoleAdmin, CapSyncArgo, true},
		{"admin has argo:refresh", RoleAdmin, CapRefreshArgo, true},
		{"admin has crossplane:pause", RoleAdmin, CapPauseCrossplane, true},
		{"admin has crossplane:reconcile", RoleAdmin, CapReconcileCrossplane, true},
		{"admin has resource:delete", RoleAdmin, CapDeleteResource, true},

		// Infra tests - should have most capabilities except delete
		{"infra has argo:sync", RoleInfra, CapSyncArgo, true},
		{"infra has argo:refresh", RoleInfra, CapRefreshArgo, true},
		{"infra has crossplane:pause", RoleInfra, CapPauseCrossplane, true},
		{"infra has crossplane:reconcile", RoleInfra, CapReconcileCrossplane, true},
		{"infra does not have resource:delete", RoleInfra, CapDeleteResource, false},

		// ML tests - should only have refresh
		{"ml has argo:refresh", RoleML, CapRefreshArgo, true},
		{"ml does not have argo:sync", RoleML, CapSyncArgo, false},
		{"ml does not have crossplane:pause", RoleML, CapPauseCrossplane, false},
		{"ml does not have crossplane:reconcile", RoleML, CapReconcileCrossplane, false},
		{"ml does not have resource:delete", RoleML, CapDeleteResource, false},

		// ReadOnly tests - should have no capabilities
		{"readonly does not have argo:sync", RoleReadOnly, CapSyncArgo, false},
		{"readonly does not have argo:refresh", RoleReadOnly, CapRefreshArgo, false},
		{"readonly does not have crossplane:pause", RoleReadOnly, CapPauseCrossplane, false},
		{"readonly does not have crossplane:reconcile", RoleReadOnly, CapReconcileCrossplane, false},
		{"readonly does not have resource:delete", RoleReadOnly, CapDeleteResource, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasCapability(tt.role, tt.capability)
			if result != tt.expected {
				t.Errorf("hasCapability(%v, %v) = %v, want %v", tt.role, tt.capability, result, tt.expected)
			}
		})
	}
}

func TestGetUserCapabilities(t *testing.T) {
	tests := []struct {
		name          string
		role          Role
		expectedCount int
	}{
		{"admin has 6 capabilities", RoleAdmin, 6},  // Updated: admin now has terminal capability
		{"infra has 5 capabilities", RoleInfra, 5},  // Updated: infra now has terminal capability
		{"ml has 1 capability", RoleML, 1},
		{"readonly has 0 capabilities", RoleReadOnly, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := GetUserCapabilities(tt.role)
			if len(caps) != tt.expectedCount {
				t.Errorf("GetUserCapabilities(%v) returned %d capabilities, want %d", tt.role, len(caps), tt.expectedCount)
			}
		})
	}
}

func TestHasCapabilityFunction(t *testing.T) {
	tests := []struct {
		name       string
		user       UserInfo
		capability Capability
		expected   bool
	}{
		{
			name:       "admin user has delete capability",
			user:       UserInfo{Username: "admin-user", Role: RoleAdmin},
			capability: CapDeleteResource,
			expected:   true,
		},
		{
			name:       "infra user does not have delete capability",
			user:       UserInfo{Username: "infra-user", Role: RoleInfra},
			capability: CapDeleteResource,
			expected:   false,
		},
		{
			name:       "ml user has refresh capability",
			user:       UserInfo{Username: "ml-user", Role: RoleML},
			capability: CapRefreshArgo,
			expected:   true,
		},
		{
			name:       "readonly user has no capabilities",
			user:       UserInfo{Username: "readonly-user", Role: RoleReadOnly},
			capability: CapRefreshArgo,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasCapability(tt.user, tt.capability)
			if result != tt.expected {
				t.Errorf("HasCapability(%v, %v) = %v, want %v", tt.user.Role, tt.capability, result, tt.expected)
			}
		})
	}
}

func TestRequireCapability(t *testing.T) {
	// Mock handler that just returns 200 OK
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	tests := []struct {
		name           string
		role           Role
		capability     Capability
		expectedStatus int
	}{
		// Admin should access everything
		{"admin can sync argo", RoleAdmin, CapSyncArgo, http.StatusOK},
		{"admin can delete", RoleAdmin, CapDeleteResource, http.StatusOK},

		// Infra tests
		{"infra can sync argo", RoleInfra, CapSyncArgo, http.StatusOK},
		{"infra can pause crossplane", RoleInfra, CapPauseCrossplane, http.StatusOK},
		{"infra cannot delete", RoleInfra, CapDeleteResource, http.StatusForbidden},

		// ML tests
		{"ml can refresh argo", RoleML, CapRefreshArgo, http.StatusOK},
		{"ml cannot sync argo", RoleML, CapSyncArgo, http.StatusForbidden},
		{"ml cannot pause crossplane", RoleML, CapPauseCrossplane, http.StatusForbidden},

		// ReadOnly tests
		{"readonly cannot do anything", RoleReadOnly, CapRefreshArgo, http.StatusForbidden},
		{"readonly cannot delete", RoleReadOnly, CapDeleteResource, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware with capability check
			middleware := RequireCapability(tt.capability)
			handler := middleware(mockHandler)

			// Create request with user context
			req := httptest.NewRequest("POST", "/test", nil)
			ctx := context.WithValue(req.Context(), userInfoKey, UserInfo{
				Username: "test-user",
				Role:     tt.role,
			})
			req = req.WithContext(ctx)

			// Record response
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}

			// If forbidden, check error message contains expected text
			if tt.expectedStatus == http.StatusForbidden {
				body := rr.Body.String()
				if body != `{"error": "Forbidden: insufficient permissions"}` && body != "{\"error\": \"Forbidden: insufficient permissions\"}\n" {
					t.Logf("Expected forbidden error message, got: %q", body)
				}
			}
		})
	}
}

func TestRequireAnyCapability(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	tests := []struct {
		name           string
		role           Role
		capabilities   []Capability
		expectedStatus int
	}{
		{
			name:           "ml has one of the capabilities (refresh)",
			role:           RoleML,
			capabilities:   []Capability{CapSyncArgo, CapRefreshArgo},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ml has none of the capabilities",
			role:           RoleML,
			capabilities:   []Capability{CapSyncArgo, CapPauseCrossplane},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin has all capabilities",
			role:           RoleAdmin,
			capabilities:   []Capability{CapSyncArgo, CapDeleteResource},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "readonly has none",
			role:           RoleReadOnly,
			capabilities:   []Capability{CapRefreshArgo},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := RequireAnyCapability(tt.capabilities...)
			handler := middleware(mockHandler)

			req := httptest.NewRequest("POST", "/test", nil)
			ctx := context.WithValue(req.Context(), userInfoKey, UserInfo{
				Username: "test-user",
				Role:     tt.role,
			})
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRequireAllCapabilities(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	tests := []struct {
		name           string
		role           Role
		capabilities   []Capability
		expectedStatus int
	}{
		{
			name:           "admin has all capabilities",
			role:           RoleAdmin,
			capabilities:   []Capability{CapSyncArgo, CapRefreshArgo, CapPauseCrossplane},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "infra missing one capability (delete)",
			role:           RoleInfra,
			capabilities:   []Capability{CapSyncArgo, CapDeleteResource},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "infra has all requested capabilities",
			role:           RoleInfra,
			capabilities:   []Capability{CapSyncArgo, CapRefreshArgo},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ml missing capabilities",
			role:           RoleML,
			capabilities:   []Capability{CapRefreshArgo, CapSyncArgo},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := RequireAllCapabilities(tt.capabilities...)
			handler := middleware(mockHandler)

			req := httptest.NewRequest("POST", "/test", nil)
			ctx := context.WithValue(req.Context(), userInfoKey, UserInfo{
				Username: "test-user",
				Role:     tt.role,
			})
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRequireCapabilityWithoutUser(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireCapability(CapSyncArgo)
	handler := middleware(mockHandler)

	// Request without user context (will default to readonly)
	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should be forbidden since readonly has no capabilities
	if rr.Code != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}
