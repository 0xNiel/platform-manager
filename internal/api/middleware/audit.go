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
	"time"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var auditLog = logf.Log.WithName("audit")

// AuditLog represents an audit log entry
type AuditLog struct {
	Timestamp   time.Time `json:"timestamp"`
	User        string    `json:"user"`
	UserRole    string    `json:"userRole"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	ResourceGVK string    `json:"resourceGVK,omitempty"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	RequestID   string    `json:"requestID,omitempty"`
	Duration    string    `json:"duration,omitempty"`
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	// LogAction logs an action performed by a user
	LogAction(ctx context.Context, action string, resource string, success bool, err error)

	// LogActionWithDetails logs an action with additional details
	LogActionWithDetails(ctx context.Context, action string, resource string, resourceGVK string, success bool, err error)
}

// DefaultAuditLogger implements AuditLogger using structured logging
type DefaultAuditLogger struct {
	logger logr.Logger
}

// NewDefaultAuditLogger creates a new DefaultAuditLogger
func NewDefaultAuditLogger() AuditLogger {
	return &DefaultAuditLogger{
		logger: auditLog,
	}
}

// NewAuditLoggerWithLogger creates a new DefaultAuditLogger with a custom logger
func NewAuditLoggerWithLogger(logger logr.Logger) AuditLogger {
	return &DefaultAuditLogger{
		logger: logger,
	}
}

// LogAction logs an action performed by a user
func (a *DefaultAuditLogger) LogAction(ctx context.Context, action string, resource string, success bool, err error) {
	a.LogActionWithDetails(ctx, action, resource, "", success, err)
}

// LogActionWithDetails logs an action with additional details
func (a *DefaultAuditLogger) LogActionWithDetails(ctx context.Context, action string, resource string, resourceGVK string, success bool, err error) {
	user := UserFromContext(ctx)

	entry := AuditLog{
		Timestamp:   time.Now(),
		User:        user.Username,
		UserRole:    string(user.Role),
		Action:      action,
		Resource:    resource,
		ResourceGVK: resourceGVK,
		Success:     success,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Extract request ID if available
	if requestID := ctx.Value("requestID"); requestID != nil {
		if id, ok := requestID.(string); ok {
			entry.RequestID = id
		}
	}

	// Log with structured fields
	keysAndValues := []interface{}{
		"timestamp", entry.Timestamp,
		"user", entry.User,
		"userRole", entry.UserRole,
		"action", entry.Action,
		"resource", entry.Resource,
		"success", entry.Success,
	}

	if entry.ResourceGVK != "" {
		keysAndValues = append(keysAndValues, "resourceGVK", entry.ResourceGVK)
	}

	if entry.RequestID != "" {
		keysAndValues = append(keysAndValues, "requestID", entry.RequestID)
	}

	if entry.Error != "" {
		keysAndValues = append(keysAndValues, "error", entry.Error)
		a.logger.Error(nil, "Audit: Action failed", keysAndValues...)
	} else {
		a.logger.Info("Audit: Action performed", keysAndValues...)
	}
}

// NoOpAuditLogger is an audit logger that does nothing (for testing)
type NoOpAuditLogger struct{}

// NewNoOpAuditLogger creates a new NoOpAuditLogger
func NewNoOpAuditLogger() AuditLogger {
	return &NoOpAuditLogger{}
}

// LogAction does nothing
func (n *NoOpAuditLogger) LogAction(ctx context.Context, action string, resource string, success bool, err error) {
	// No-op
}

// LogActionWithDetails does nothing
func (n *NoOpAuditLogger) LogActionWithDetails(ctx context.Context, action string, resource string, resourceGVK string, success bool, err error) {
	// No-op
}
