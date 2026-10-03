/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package terminal

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Session represents an active terminal session
type Session struct {
	ID              string
	Username        string
	TenantID        string
	Namespace       string
	PodName         string
	CreatedAt       time.Time
	LastActivity    time.Time
	IdleTimeout     time.Duration
	Conn            *websocket.Conn
	CommandRecorder *CommandRecorder
	Done            chan struct{}
	RateLimiter     *RateLimiter // Rate limiter for input throttling

	// ExecContext and ExecCancel track the current exec stream
	// These are used to cancel previous exec when a new WebSocket connects
	ExecContext context.Context
	ExecCancel  context.CancelFunc
	ExecMu      sync.Mutex // Protects ExecContext/ExecCancel
}

// SessionOptions configures a new terminal session
type SessionOptions struct {
	Username    string
	TenantID    string
	Namespace   string
	IdleTimeout time.Duration
	Shell       string // Default: /bin/bash
	WorkingDir  string // Default: /home/toolbox
}

// SessionStatus represents the status of a terminal session
type SessionStatus struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	TenantID     string    `json:"tenantId"`
	PodName      string    `json:"podName"`
	Namespace    string    `json:"namespace"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActivity time.Time `json:"lastActivity"`
	IsActive     bool      `json:"isActive"`
	IdleMinutes  int       `json:"idleMinutes"`
}

// Config holds terminal feature configuration
type Config struct {
	Enabled        bool
	Namespace      string // Namespace for toolbox pods (default: toolbox-sessions)
	ToolboxImage   string // Toolbox container image
	IdleTimeout    time.Duration
	MaxSessions    int
	AllowedTenants []string // Empty = all tenants allowed
	ServiceAccount string   // ServiceAccount for toolbox pods
}

// SecurityConfig holds security settings for terminal WebSocket connections
type SecurityConfig struct {
	AllowedOrigins []string // List of allowed origins for WebSocket connections
	DevMode        bool     // Development mode - allows all origins when true
}

// DefaultSecurityConfig returns secure defaults for terminal security
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		AllowedOrigins: []string{},
		DevMode:        false,
	}
}

// IsOriginAllowed checks if an origin is in the allowed list
func (c *SecurityConfig) IsOriginAllowed(origin string) bool {
	if c.DevMode {
		return true // Allow all in dev mode
	}

	for _, allowed := range c.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// DefaultConfig returns default terminal configuration
func DefaultConfig() Config {
	return Config{
		Enabled:        false, // Disabled by default
		Namespace:      "toolbox-sessions",
		ToolboxImage:   "platform-manager-toolbox:latest",
		IdleTimeout:    10 * time.Minute,
		MaxSessions:    20,
		AllowedTenants: []string{},
		ServiceAccount: "toolbox-session",
	}
}

// IsAllowed checks if a tenant is allowed to use terminal
func (c *Config) IsAllowed(tenantID string) bool {
	if !c.Enabled {
		return false
	}

	// If no specific tenants listed, all are allowed
	if len(c.AllowedTenants) == 0 {
		return true
	}

	// Check if tenant is in allowed list
	for _, allowed := range c.AllowedTenants {
		if allowed == tenantID {
			return true
		}
	}

	return false
}
