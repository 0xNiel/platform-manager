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
	"os"
	"strings"
)

// Role represents a user role in the system
type Role string

const (
	// RoleAdmin has full access to all operations
	RoleAdmin Role = "admin"
	// RoleInfra has access to infrastructure operations
	RoleInfra Role = "infra"
	// RoleML has limited access for ML team
	RoleML Role = "ml"
	// RoleReadOnly has read-only access
	RoleReadOnly Role = "readonly"
)

// UserInfo contains extracted user information from OAuth2Proxy headers
type UserInfo struct {
	// Username is the authenticated user's name
	Username string
	// Email is the user's email address
	Email string
	// Groups are the groups the user belongs to
	Groups []string
	// Role is the mapped role based on groups
	Role Role
}

type contextKey string

const userInfoKey contextKey = "userInfo"

// ExtractUser middleware extracts user info from OAuth2Proxy headers
func ExtractUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserInfo{
			Username: r.Header.Get("X-Auth-Request-User"),
			Email:    r.Header.Get("X-Auth-Request-Email"),
			Groups:   parseGroups(r.Header.Get("X-Auth-Request-Groups")),
		}

		// If no user from headers, handle based on environment
		if user.Username == "" {
			// Check if dev mode is explicitly enabled
			devMode := os.Getenv("DEV_MODE") == "true"

			if devMode {
				// Development mode: allow dev header
				if devRole := r.Header.Get("X-Dev-Role"); devRole != "" {
					// Validate role is a real role
					if isValidRole(Role(devRole)) {
						user.Username = "dev-user"
						user.Email = "dev@localhost"
						user.Role = Role(devRole)

						// Log dev mode usage (not too verbose to avoid log spam)
						// In production this should trigger alerts
					} else {
						user.Username = "anonymous"
						user.Role = RoleReadOnly
					}
				} else {
					user.Username = "dev-anonymous"
					user.Role = RoleReadOnly
				}
			} else {
				// Production: no authentication means read-only anonymous
				user.Username = "anonymous"
				user.Role = RoleReadOnly
			}
		} else {
			// Map groups to role
			user.Role = mapGroupsToRole(user.Groups)
		}

		ctx := context.WithValue(r.Context(), userInfoKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// parseGroups splits the groups header into a slice
func parseGroups(groupsHeader string) []string {
	if groupsHeader == "" {
		return nil
	}
	groups := strings.Split(groupsHeader, ",")
	result := make([]string, 0, len(groups))
	for _, g := range groups {
		if trimmed := strings.TrimSpace(g); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// mapGroupsToRole maps user groups to a platform role
func mapGroupsToRole(groups []string) Role {
	for _, g := range groups {
		switch strings.ToLower(strings.TrimSpace(g)) {
		case "platform-admin", "admin", "administrators":
			return RoleAdmin
		case "platform-infra", "infra", "infrastructure":
			return RoleInfra
		case "platform-ml", "ml", "machine-learning":
			return RoleML
		}
	}
	return RoleReadOnly
}

// UserFromContext extracts UserInfo from the request context
func UserFromContext(ctx context.Context) UserInfo {
	if user, ok := ctx.Value(userInfoKey).(UserInfo); ok {
		return user
	}
	return UserInfo{Username: "unknown", Role: RoleReadOnly}
}

// RequireRole returns middleware that requires a specific role or higher
func RequireRole(requiredRoles ...Role) func(http.Handler) http.Handler {
	roleSet := make(map[Role]bool)
	for _, r := range requiredRoles {
		roleSet[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())

			// Admin always has access
			if user.Role == RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Check if user has one of the required roles
			if roleSet[user.Role] {
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, "Forbidden", http.StatusForbidden)
		})
	}
}

// isValidRole checks if a role string is valid
func isValidRole(role Role) bool {
	switch role {
	case RoleAdmin, RoleInfra, RoleML, RoleReadOnly:
		return true
	default:
		return false
	}
}
