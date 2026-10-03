/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package terminal

import (
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
)

// CommandRecord represents a single command execution
type CommandRecord struct {
	SessionID string    `json:"sessionId"`
	Username  string    `json:"username"`
	TenantID  string    `json:"tenantId"`
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
	ExitCode  int       `json:"exitCode,omitempty"`
	Duration  int64     `json:"duration,omitempty"` // milliseconds
}

// CommandRecorder handles audit logging of terminal commands
type CommandRecorder struct {
	sessionID string
	username  string
	tenantID  string
	logger    logr.Logger
}

// NewCommandRecorder creates a new command recorder
func NewCommandRecorder(sessionID, username, tenantID string, logger logr.Logger) *CommandRecorder {
	return &CommandRecorder{
		sessionID: sessionID,
		username:  username,
		tenantID:  tenantID,
		logger:    logger.WithValues("sessionID", sessionID, "username", username, "tenantID", tenantID),
	}
}

// RecordCommand logs a command execution
func (r *CommandRecorder) RecordCommand(command string, exitCode int, duration time.Duration) {
	record := CommandRecord{
		SessionID: r.sessionID,
		Username:  r.username,
		TenantID:  r.tenantID,
		Command:   command,
		Timestamp: time.Now(),
		ExitCode:  exitCode,
		Duration:  duration.Milliseconds(),
	}

	// Log as structured JSON
	recordJSON, _ := json.Marshal(record)
	r.logger.Info("terminal command executed",
		"audit", string(recordJSON),
		"command", command,
		"exitCode", exitCode,
		"duration", duration.Milliseconds())
}

// RecordSessionStart logs session start
func (r *CommandRecorder) RecordSessionStart() {
	r.logger.Info("terminal session started",
		"event", "session_start",
		"timestamp", time.Now())
}

// RecordSessionEnd logs session end
func (r *CommandRecorder) RecordSessionEnd(reason string) {
	r.logger.Info("terminal session ended",
		"event", "session_end",
		"reason", reason,
		"timestamp", time.Now())
}

// RecordError logs an error event
func (r *CommandRecorder) RecordError(err error, context string) {
	r.logger.Error(err, "terminal session error",
		"event", "session_error",
		"context", context,
		"timestamp", time.Now())
}
