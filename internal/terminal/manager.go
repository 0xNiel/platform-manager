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

package terminal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Manager manages terminal sessions and toolbox pod lifecycle
type Manager struct {
	client   client.Client
	config   Config
	sessions map[string]*Session
	mu       sync.RWMutex
	logger   logr.Logger
}

// NewManager creates a new terminal manager
func NewManager(client client.Client, config Config, logger logr.Logger) *Manager {
	return &Manager{
		client:   client,
		config:   config,
		sessions: make(map[string]*Session),
		logger:   logger.WithName("terminal-manager"),
	}
}

// CreateSession creates a new terminal session using a shared toolbox pod
// The toolbox pod is shared per user (not per tenant) - it's a general-purpose terminal
func (m *Manager) CreateSession(ctx context.Context, opts SessionOptions) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if terminal is enabled
	if !m.config.Enabled {
		return nil, fmt.Errorf("terminal feature is disabled")
	}

	// Check max sessions limit
	if len(m.sessions) >= m.config.MaxSessions {
		return nil, fmt.Errorf("maximum number of sessions reached (%d)", m.config.MaxSessions)
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Check if user already has a toolbox pod running - reuse it
	podName := fmt.Sprintf("toolbox-%s", opts.Username)

	// Set defaults
	if opts.IdleTimeout == 0 {
		opts.IdleTimeout = m.config.IdleTimeout
	}
	if opts.Shell == "" {
		opts.Shell = "/bin/bash"
	}
	if opts.WorkingDir == "" {
		opts.WorkingDir = "/home/toolbox"
	}
	if opts.Namespace == "" {
		opts.Namespace = m.config.Namespace
	}

	// Check if pod already exists and is ready
	existingPod := &corev1.Pod{}
	key := client.ObjectKey{Namespace: opts.Namespace, Name: podName}
	podExists := false
	
	if err := m.client.Get(ctx, key, existingPod); err == nil {
		// Pod exists - check if it's running
		if existingPod.Status.Phase == corev1.PodRunning {
			podExists = true
			m.logger.Info("reusing existing toolbox pod", "podName", podName, "username", opts.Username)
		} else if existingPod.Status.Phase == corev1.PodFailed || existingPod.Status.Phase == corev1.PodSucceeded {
			// Pod is in a terminal state - delete it and create a new one
			m.logger.Info("deleting failed/completed toolbox pod", "podName", podName, "phase", existingPod.Status.Phase)
			_ = m.client.Delete(ctx, existingPod)
			podExists = false
		}
	}

	// Create toolbox pod if it doesn't exist
	if !podExists {
		pod, err := m.createToolboxPod(ctx, podName, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create toolbox pod: %w", err)
		}

		// Wait for pod to be ready (with timeout)
		if err := m.waitForPodReady(ctx, pod.Namespace, pod.Name, 60*time.Second); err != nil {
			// Clean up pod on failure
			_ = m.client.Delete(ctx, pod)
			return nil, fmt.Errorf("pod failed to become ready: %w", err)
		}
		m.logger.Info("created new toolbox pod", "podName", podName, "username", opts.Username)
	}

	// Create session
	session := &Session{
		ID:           sessionID,
		Username:     opts.Username,
		TenantID:     "", // No longer tied to a tenant
		Namespace:    opts.Namespace,
		PodName:      podName,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		IdleTimeout:  opts.IdleTimeout,
		Done:         make(chan struct{}),
		RateLimiter:  NewRateLimiter(100, 10), // 100 tokens, refill 10/sec
		CommandRecorder: NewCommandRecorder(
			sessionID,
			opts.Username,
			"", // No tenant
			m.logger,
		),
	}

	m.sessions[sessionID] = session

	// Start idle timeout goroutine
	go m.monitorSession(session)

	session.CommandRecorder.RecordSessionStart()
	m.logger.Info("terminal session created",
		"sessionID", sessionID,
		"username", opts.Username,
		"podName", podName)

	return session, nil
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// DeleteSession deletes a session but keeps the shared toolbox pod running
// The pod is reused for future sessions from the same user
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Cancel any running exec context
	session.ExecMu.Lock()
	if session.ExecCancel != nil {
		session.ExecCancel()
	}
	session.ExecMu.Unlock()

	// Close websocket if open
	if session.Conn != nil {
		_ = session.Conn.Close()
	}

	// Signal session done
	select {
	case <-session.Done:
		// Already closed
	default:
		close(session.Done)
	}

	// NOTE: We intentionally do NOT delete the toolbox pod here
	// The pod is shared and will be reused for future sessions
	// Pods are only deleted when they fail or after idle timeout

	// Remove from sessions map
	delete(m.sessions, sessionID)

	session.CommandRecorder.RecordSessionEnd("user_requested")
	m.logger.Info("terminal session deleted",
		"sessionID", sessionID,
		"username", session.Username,
		"podName", session.PodName)

	return nil
}

// ListSessions returns all active sessions
func (m *Manager) ListSessions() []SessionStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]SessionStatus, 0, len(m.sessions))
	now := time.Now()

	for _, session := range m.sessions {
		idle := now.Sub(session.LastActivity)
		statuses = append(statuses, SessionStatus{
			ID:           session.ID,
			Username:     session.Username,
			TenantID:     session.TenantID,
			PodName:      session.PodName,
			Namespace:    session.Namespace,
			CreatedAt:    session.CreatedAt,
			LastActivity: session.LastActivity,
			IsActive:     idle < session.IdleTimeout,
			IdleMinutes:  int(idle.Minutes()),
		})
	}

	return statuses
}

// UpdateActivity updates the last activity time for a session
func (m *Manager) UpdateActivity(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[sessionID]; exists {
		session.LastActivity = time.Now()
	}
}

// GetConfig returns the terminal configuration
func (m *Manager) GetConfig() Config {
	return m.config
}

// IsEnabled returns whether terminal feature is enabled
func (m *Manager) IsEnabled() bool {
	return m.config.Enabled
}

// createToolboxPod creates a shared toolbox pod for terminal sessions
// The pod is per-user, not per-tenant, and can be used for any platform operations
func (m *Manager) createToolboxPod(ctx context.Context, podName string, opts SessionOptions) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				"app":                      "platform-manager-toolbox",
				"platform.io/component":    "terminal",
				"platform.io/session-user": opts.Username,
			},
			Annotations: map[string]string{
				"platform.io/created-by": "platform-manager",
				"platform.io/created-at": time.Now().Format(time.RFC3339),
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: m.config.ServiceAccount,
			RestartPolicy:      corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "toolbox",
					Image:           m.config.ToolboxImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{opts.Shell},
					Args:            []string{"-c", "sleep infinity"},
					WorkingDir:      opts.WorkingDir,
					Env: []corev1.EnvVar{
						{Name: "USERNAME", Value: opts.Username},
						{Name: "PS1", Value: fmt.Sprintf("[%s@toolbox \\W]$ ", opts.Username)},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    mustParseQuantity("100m"),
							corev1.ResourceMemory: mustParseQuantity("128Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    mustParseQuantity("500m"),
							corev1.ResourceMemory: mustParseQuantity("512Mi"),
						},
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot:             ptrBool(true),
						RunAsUser:                ptrInt64(1000),
						AllowPrivilegeEscalation: ptrBool(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
					},
					TTY:   true,
					Stdin: true,
				},
			},
			SecurityContext: &corev1.PodSecurityContext{
				FSGroup: ptrInt64(1000),
			},
		},
	}

	if err := m.client.Create(ctx, pod); err != nil {
		return nil, err
	}

	return pod, nil
}

// waitForPodReady waits for pod to be ready
func (m *Manager) waitForPodReady(ctx context.Context, namespace, podName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for pod to be ready")
		case <-ticker.C:
			pod := &corev1.Pod{}
			key := client.ObjectKey{Namespace: namespace, Name: podName}
			if err := m.client.Get(ctx, key, pod); err != nil {
				continue
			}

			// Check if pod is running and all containers are ready
			if pod.Status.Phase == corev1.PodRunning {
				for _, cond := range pod.Status.Conditions {
					if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
						return nil
					}
				}
			}

			// Check for failed status
			if pod.Status.Phase == corev1.PodFailed {
				return fmt.Errorf("pod failed: %s", pod.Status.Message)
			}
		}
	}
}

// monitorSession monitors session for idle timeout
func (m *Manager) monitorSession(session *Session) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-session.Done:
			return
		case <-ticker.C:
			m.mu.RLock()
			idle := time.Since(session.LastActivity)
			m.mu.RUnlock()

			if idle > session.IdleTimeout {
				m.logger.Info("session idle timeout",
					"sessionID", session.ID,
					"username", session.Username,
					"idle", idle)

				ctx := context.Background()
				_ = m.DeleteSession(ctx, session.ID)
				return
			}
		}
	}
}

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrInt64(i int64) *int64 {
	return &i
}

func mustParseQuantity(s string) resource.Quantity {
	// Parse quantity using resource package
	q, _ := resource.ParseQuantity(s)
	return q
}
