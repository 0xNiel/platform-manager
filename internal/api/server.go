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

package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/platform-manager/platform-manager/internal/api/handlers"
	"github.com/platform-manager/platform-manager/internal/api/middleware"
)

var log = logf.Log.WithName("api-server")

// Server represents the HTTP API server
type Server struct {
	client     client.Client
	httpServer *http.Server
	addr       string
}

// NewServer creates a new API server
func NewServer(addr string, k8sClient client.Client) *Server {
	return &Server{
		client: k8sClient,
		addr:   addr,
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	router := s.setupRouter()

	s.httpServer = &http.Server{
		Addr:         s.addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info("Starting API server", "addr", s.addr)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		log.Info("Shutting down API server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}
}

// setupRouter configures the Chi router with all routes and middleware
func (s *Server) setupRouter() *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// CORS middleware for development
	r.Use(middleware.CORS)

	// Auth middleware - extracts user info from OAuth2Proxy headers
	r.Use(middleware.ExtractUser)

	// Health check (no auth required)
	r.Get("/healthz", handlers.Healthz)
	r.Get("/readyz", handlers.Readyz)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Create handlers with client
		healthHandler := handlers.NewHealthHandler(s.client)
		tenantHandler := handlers.NewTenantHandler(s.client)

		// Health endpoints
		r.Route("/health", func(r chi.Router) {
			r.Get("/platform", healthHandler.GetPlatformHealth)
			r.Get("/tenants", healthHandler.ListTenantHealth)
			r.Get("/tenants/{id}", healthHandler.GetTenantHealth)
		})

		// Tenant endpoints
		r.Route("/tenants", func(r chi.Router) {
			r.Get("/", tenantHandler.List)
			r.Get("/{id}", tenantHandler.Get)
			r.Get("/{id}/resources", tenantHandler.ListResources)
		})
	})

	return r
}
