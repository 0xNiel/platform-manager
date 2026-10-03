/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package middleware

import (
	"net/http"

	"github.com/0xNiel/platform-manager/internal/metrics"
)

// Capability represents a specific permission in the system
type Capability string

const (
	// CapSyncArgo allows syncing ArgoCD applications
	CapSyncArgo Capability = "argo:sync"
	// CapRefreshArgo allows refreshing ArgoCD applications
	CapRefreshArgo Capability = "argo:refresh"
	// CapPauseCrossplane allows pausing Crossplane resources
	CapPauseCrossplane Capability = "crossplane:pause"
	// CapReconcileCrossplane allows forcing reconciliation of Crossplane resources
	CapReconcileCrossplane Capability = "crossplane:reconcile"
	// CapDeleteResource allows deleting resources
	CapDeleteResource Capability = "resource:delete"
	// CapUseTerminal allows access to web terminal
	CapUseTerminal Capability = "terminal:use"
)

// roleCapabilities maps roles to their allowed capabilities
var roleCapabilities = map[Role][]Capability{
	RoleAdmin: {
		CapSyncArgo,
		CapRefreshArgo,
		CapPauseCrossplane,
		CapReconcileCrossplane,
		CapDeleteResource,
		CapUseTerminal,
	},
	RoleInfra: {
		CapSyncArgo,
		CapRefreshArgo,
		CapPauseCrossplane,
		CapReconcileCrossplane,
		CapUseTerminal,
	},
	RoleML: {
		CapRefreshArgo,
	},
	RoleReadOnly: {},
}

// RequireCapability returns middleware that checks if the user has a specific capability
func RequireCapability(cap Capability) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())

			// Check if user has the required capability
			if !hasCapability(user.Role, cap) {
				// Record authorization denial metric
				actionName := capabilityToActionName(cap)
				if actionName != "" {
					metrics.ActionsMetrics.RecordAuthDenial(actionName, string(user.Role))
				}

				http.Error(w, `{"error": "Forbidden: insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyCapability returns middleware that checks if the user has at least one of the specified capabilities
func RequireAnyCapability(caps ...Capability) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())

			// Check if user has any of the required capabilities
			for _, cap := range caps {
				if hasCapability(user.Role, cap) {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"error": "Forbidden: insufficient permissions"}`, http.StatusForbidden)
		})
	}
}

// RequireAllCapabilities returns middleware that checks if the user has all specified capabilities
func RequireAllCapabilities(caps ...Capability) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())

			// Check if user has all required capabilities
			for _, cap := range caps {
				if !hasCapability(user.Role, cap) {
					http.Error(w, `{"error": "Forbidden: insufficient permissions"}`, http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hasCapability checks if a role has a specific capability
func hasCapability(role Role, cap Capability) bool {
	caps, exists := roleCapabilities[role]
	if !exists {
		return false
	}

	for _, c := range caps {
		if c == cap {
			return true
		}
	}

	return false
}

// GetUserCapabilities returns all capabilities for a given role
func GetUserCapabilities(role Role) []Capability {
	caps, exists := roleCapabilities[role]
	if !exists {
		return []Capability{}
	}
	return caps
}

// HasCapability checks if a user (from context) has a specific capability
func HasCapability(user UserInfo, cap Capability) bool {
	return hasCapability(user.Role, cap)
}

// capabilityToActionName maps capability to metrics action name
func capabilityToActionName(cap Capability) string {
	switch cap {
	case CapSyncArgo:
		return metrics.ActionArgoSync
	case CapRefreshArgo:
		return metrics.ActionArgoRefresh
	case CapPauseCrossplane:
		return metrics.ActionCrossplanePause
	case CapReconcileCrossplane:
		return metrics.ActionCrossplaneReconcile
	case CapDeleteResource:
		return metrics.ActionResourceDelete
	case CapUseTerminal:
		return "terminal:use"
	default:
		return ""
	}
}
