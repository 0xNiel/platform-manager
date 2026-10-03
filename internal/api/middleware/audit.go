/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package middleware

import (
	"context"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
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
	// Set by chi's RequestID middleware, which the API server installs first.
	entry.RequestID = chimiddleware.GetReqID(ctx)

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
