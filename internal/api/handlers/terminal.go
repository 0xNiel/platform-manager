/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-logr/logr"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/0xNiel/platform-manager/internal/api/middleware"
	"github.com/0xNiel/platform-manager/internal/terminal"
)

// TerminalHandler handles terminal WebSocket connections
type TerminalHandler struct {
	manager        *terminal.Manager
	clientset      *kubernetes.Clientset
	restConfig     *rest.Config
	logger         logr.Logger
	securityConfig terminal.SecurityConfig
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(k8sClient client.Client, terminalMgr *terminal.Manager, logger logr.Logger, securityConfig terminal.SecurityConfig) (*TerminalHandler, error) {
	// Get rest config - try in-cluster first, then fallback to kubeconfig
	config, err := rest.InClusterConfig()
	if err != nil {
		// Development mode: use kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get REST config (tried in-cluster and kubeconfig): %w", err)
		}
		logger.Info("Using kubeconfig for terminal handler (development mode)")
	} else {
		logger.Info("Using in-cluster config for terminal handler")
	}

	// Create clientset for exec
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &TerminalHandler{
		manager:        terminalMgr,
		clientset:      clientset,
		restConfig:     config,
		logger:         logger.WithName("terminal-handler"),
		securityConfig: securityConfig,
	}, nil
}

// RegisterRoutes registers terminal routes
func (h *TerminalHandler) RegisterRoutes(r chi.Router) {
	// WebSocket endpoint - registered OUTSIDE /terminal route to avoid middleware inheritance
	// This endpoint validates session existence instead of checking user capabilities
	// (browsers can't send custom headers with WebSocket connections)
	r.Get("/terminal/sessions/{sessionId}/ws", h.HandleWebSocket)

	r.Route("/terminal", func(r chi.Router) {
		// Config endpoint doesn't need auth - it returns whether terminal is enabled and if user has access
		r.Get("/config", h.GetConfig)

		// All other terminal routes require terminal to be enabled and user to have capability
		r.Group(func(r chi.Router) {
			r.Use(h.requireTerminalEnabled)
			r.Use(middleware.RequireCapability(middleware.CapUseTerminal))

			r.Post("/sessions", h.CreateSession)
			r.Get("/sessions", h.ListSessions)
			r.Get("/sessions/{sessionId}", h.GetSession)
			r.Delete("/sessions/{sessionId}", h.DeleteSession)
		})
	})
}

// createUpgrader creates a WebSocket upgrader with proper origin validation
func (h *TerminalHandler) createUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			allowed := h.securityConfig.IsOriginAllowed(origin)

			if !allowed {
				h.logger.Info("rejected websocket connection from unauthorized origin",
					"origin", origin,
					"remoteAddr", r.RemoteAddr)
			}

			return allowed
		},
	}
}

// requireTerminalEnabled middleware checks if terminal feature is enabled
func (h *TerminalHandler) requireTerminalEnabled(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.manager.IsEnabled() {
			http.Error(w, `{"error": "Terminal feature is disabled"}`, http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CreateSession creates a new terminal session
// POST /api/v1/terminal/sessions
// The terminal is a shared toolbox pod - no tenant ID required
func (h *TerminalHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	// Body is optional - no tenant ID required for shared toolbox
	// Just drain the body to avoid issues
	_, _ = io.ReadAll(r.Body)

	// Create session with shared toolbox
	opts := terminal.SessionOptions{
		Username: user.Username,
	}

	session, err := h.manager.CreateSession(r.Context(), opts)
	if err != nil {
		h.logger.Error(err, "failed to create terminal session", "username", user.Username)
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"sessionId": session.ID,
		"podName":   session.PodName,
		"namespace": session.Namespace,
		"createdAt": session.CreatedAt,
		"wsUrl":     fmt.Sprintf("/api/v1/terminal/sessions/%s/ws", session.ID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListSessions lists all active sessions
// GET /api/v1/terminal/sessions
func (h *TerminalHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.manager.ListSessions()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// GetSession gets session details
// GET /api/v1/terminal/sessions/:sessionId
func (h *TerminalHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")

	session, err := h.manager.GetSession(sessionID)
	if err != nil {
		http.Error(w, `{"error": "Session not found"}`, http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"sessionId":    session.ID,
		"username":     session.Username,
		"tenantId":     session.TenantID,
		"podName":      session.PodName,
		"namespace":    session.Namespace,
		"createdAt":    session.CreatedAt,
		"lastActivity": session.LastActivity,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteSession deletes a terminal session
// DELETE /api/v1/terminal/sessions/:sessionId
func (h *TerminalHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")

	if err := h.manager.DeleteSession(r.Context(), sessionID); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleWebSocket handles WebSocket connection for terminal
// GET /api/v1/terminal/sessions/:sessionId/ws
//
// Note: This endpoint doesn't use the normal auth middleware because browsers
// cannot send custom headers with WebSocket connections. Instead, we validate
// that the session exists and the origin is allowed. Security relies on:
// 1. The session was created by an authenticated user (via POST /sessions)
// 2. The sessionId is a UUID that's hard to guess
// 3. Origin validation prevents cross-site WebSocket hijacking
// 4. In production, OAuth2Proxy cookies will provide authentication
func (h *TerminalHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")

	// Get session - this validates the sessionID exists
	session, err := h.manager.GetSession(sessionID)
	if err != nil {
		h.logger.Error(err, "session not found for websocket", "sessionId", sessionID)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Create upgrader with security config
	upgrader := h.createUpgrader()

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error(err, "failed to upgrade to websocket", "sessionId", sessionID)
		return
	}

	h.logger.Info("websocket connected", "sessionId", sessionID, "username", session.Username, "podName", session.PodName)

	// Cancel any existing exec stream for this session
	session.ExecMu.Lock()
	if session.ExecCancel != nil {
		h.logger.Info("canceling previous exec stream", "sessionId", sessionID)
		session.ExecCancel()
	}

	// Create a new cancellable context for this exec
	ctx, cancel := context.WithCancel(context.Background())
	session.ExecContext = ctx
	session.ExecCancel = cancel
	session.Conn = conn
	session.ExecMu.Unlock()

	// Ensure cleanup on function exit
	defer func() {
		h.logger.Info("cleaning up websocket connection", "sessionId", sessionID)
		cancel() // Cancel the exec context
		conn.Close()

		// Clear the connection from session
		session.ExecMu.Lock()
		if session.Conn == conn {
			session.Conn = nil
		}
		session.ExecMu.Unlock()
	}()

	// Create pipes for stdin
	stdinReader, stdinWriter := io.Pipe()

	// Create pipes for stdout/stderr
	stdoutReader, stdoutWriter := io.Pipe()

	// Ensure pipes are closed on exit
	defer func() {
		stdinWriter.Close()
		stdinReader.Close()
		stdoutWriter.Close()
		stdoutReader.Close()
	}()

	// Channel to signal goroutines to stop
	done := make(chan struct{})
	defer close(done)

	// Goroutine to read from WebSocket and write to stdin pipe
	go func() {
		defer stdinWriter.Close()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			default:
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				// Only log as error if not a normal close
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
					h.logger.Error(err, "websocket read error", "sessionId", sessionID)
				} else {
					h.logger.Info("websocket closed", "sessionId", sessionID)
				}
				return
			}

			// Apply rate limiting to prevent input flooding
			if !session.RateLimiter.Allow() {
				h.logger.Info("rate limit exceeded", "sessionId", sessionID, "username", session.Username)
				// Optionally send a message to user (commented out to avoid output spam)
				// conn.WriteMessage(websocket.TextMessage, []byte("\r\n⚠️  Rate limit exceeded.\r\n"))
				time.Sleep(100 * time.Millisecond)
				continue
			}

			h.manager.UpdateActivity(sessionID)
			_, err = stdinWriter.Write(message)
			if err != nil {
				h.logger.Error(err, "stdin write error", "sessionId", sessionID)
				return
			}
		}
	}()

	// Goroutine to read from stdout pipe and write to WebSocket
	go func() {
		defer stdoutReader.Close()
		buf := make([]byte, 8192)
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			default:
			}

			n, err := stdoutReader.Read(buf)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					h.logger.Error(writeErr, "websocket write error", "sessionId", sessionID)
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					h.logger.Error(err, "stdout read error", "sessionId", sessionID)
				}
				return
			}
		}
	}()

	// Create exec into pod
	req := h.clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(session.PodName).
		Namespace(session.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "toolbox",
			Command:   []string{"/bin/bash", "-i"}, // Interactive shell
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(h.restConfig, "POST", req.URL())
	if err != nil {
		h.logger.Error(err, "failed to create executor")
		session.CommandRecorder.RecordError(err, "executor_creation")
		return
	}

	h.logger.Info("starting stream", "sessionId", sessionID, "pod", session.PodName)

	// Create terminal size queue with default size
	terminalSizeQueue := newTerminalSizeQueue(&remotecommand.TerminalSize{
		Width:  80,
		Height: 24,
	}, done)

	// Execute with pipes - use the cancellable context
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             stdinReader,
		Stdout:            stdoutWriter,
		Stderr:            stdoutWriter,
		Tty:               true,
		TerminalSizeQueue: terminalSizeQueue,
	})

	if err != nil {
		// Only log as error if not cancelled
		if ctx.Err() != context.Canceled {
			h.logger.Error(err, "stream error", "sessionId", sessionID)
			session.CommandRecorder.RecordError(err, "stream_error")
		} else {
			h.logger.Info("stream cancelled", "sessionId", sessionID)
		}
	}

	h.logger.Info("websocket disconnected", "sessionId", sessionID)
}

// GetConfig returns terminal configuration
// GET /api/v1/terminal/config
func (h *TerminalHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	config := h.manager.GetConfig()

	response := map[string]interface{}{
		"enabled":     config.Enabled,
		"idleTimeout": int(config.IdleTimeout.Minutes()),
		"maxSessions": config.MaxSessions,
		"hasAccess":   middleware.HasCapability(user, middleware.CapUseTerminal),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// terminalStreamHandler handles bidirectional streaming
type terminalStreamHandler struct {
	conn       *websocket.Conn
	session    *terminal.Session
	manager    *terminal.Manager
	logger     logr.Logger
	resizeChan chan remotecommand.TerminalSize
}

func (h *terminalStreamHandler) Read(p []byte) (int, error) {
	_, message, err := h.conn.ReadMessage()
	if err != nil {
		h.logger.Error(err, "websocket read error")
		return 0, err
	}

	// Update activity
	h.manager.UpdateActivity(h.session.ID)

	n := copy(p, message)
	return n, nil
}

func (h *terminalStreamHandler) Write(p []byte) (int, error) {
	err := h.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		h.logger.Error(err, "websocket write error")
		return 0, err
	}
	return len(p), nil
}

// terminalSizeQueue implements remotecommand.TerminalSizeQueue
type terminalSizeQueue struct {
	sizeChan chan *remotecommand.TerminalSize
	done     <-chan struct{}
}

func newTerminalSizeQueue(initialSize *remotecommand.TerminalSize, done <-chan struct{}) *terminalSizeQueue {
	q := &terminalSizeQueue{
		sizeChan: make(chan *remotecommand.TerminalSize, 1),
		done:     done,
	}
	// Send initial size
	q.sizeChan <- initialSize
	return q
}

func (t *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		return size
	case <-t.done:
		return nil
	}
}
