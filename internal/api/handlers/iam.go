/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package handlers

import (
	"net/http"

	"github.com/0xNiel/platform-manager/internal/controller"
	"github.com/go-chi/chi/v5"
	"github.com/go-logr/logr"
)

// IAMHandler handles IAM drift-related API requests
type IAMHandler struct {
	driftScanner *controller.IAMDriftScanner
	logger       logr.Logger
}

// NewIAMHandler creates a new IAM handler
func NewIAMHandler(driftScanner *controller.IAMDriftScanner, logger logr.Logger) *IAMHandler {
	return &IAMHandler{
		driftScanner: driftScanner,
		logger:       logger,
	}
}

// RegisterRoutes registers IAM-related routes
func (h *IAMHandler) RegisterRoutes(r chi.Router) {
	r.Route("/iam/drift", func(r chi.Router) {
		r.Get("/platform", h.GetPlatformDrift)
		r.Get("/tenants", h.GetAllTenantsDrift)
		r.Get("/tenants/{tenant}", h.GetTenantDrift)
		r.Post("/scan", h.TriggerScan)
	})
}

// GetPlatformDrift returns platform-wide IAM drift summary
func (h *IAMHandler) GetPlatformDrift(w http.ResponseWriter, r *http.Request) {
	summary := h.driftScanner.GetPlatformSummary()

	response := map[string]interface{}{
		"summary":      summary,
		"lastScanTime": h.driftScanner.GetLastScanTime(),
	}

	WriteJSON(w, http.StatusOK, response)
}

// GetAllTenantsDrift returns drift summary for all tenants
func (h *IAMHandler) GetAllTenantsDrift(w http.ResponseWriter, r *http.Request) {
	platformSummary := h.driftScanner.GetPlatformSummary()

	tenantSummaries := make([]interface{}, 0, len(platformSummary.TenantSummaries))
	for _, summary := range platformSummary.TenantSummaries {
		tenantSummaries = append(tenantSummaries, map[string]interface{}{
			"tenantName":        summary.TenantName,
			"totalRoles":        summary.TotalRoles,
			"totalPolicies":     summary.TotalPolicies,
			"rolesWithDrift":    summary.RolesWithDrift,
			"policiesWithDrift": summary.PoliciesWithDrift,
			"criticalDrifts":    summary.CriticalDrifts,
			"highDrifts":        summary.HighDrifts,
			"warningDrifts":     summary.WarningDrifts,
			"lastChecked":       summary.LastChecked,
		})
	}

	response := map[string]interface{}{
		"tenants":      tenantSummaries,
		"total":        len(tenantSummaries),
		"lastScanTime": h.driftScanner.GetLastScanTime(),
	}

	WriteJSON(w, http.StatusOK, response)
}

// GetTenantDrift returns detailed drift information for a specific tenant
func (h *IAMHandler) GetTenantDrift(w http.ResponseWriter, r *http.Request) {
	tenantName := chi.URLParam(r, "tenant")
	if tenantName == "" {
		http.Error(w, "tenant name is required", http.StatusBadRequest)
		return
	}

	summary, ok := h.driftScanner.GetTenantSummary(tenantName)
	if !ok {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"tenant":  summary,
		"details": summary.Drifts,
	}

	WriteJSON(w, http.StatusOK, response)
}

// TriggerScan triggers an immediate drift scan
func (h *IAMHandler) TriggerScan(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Manual drift scan triggered via API")

	// Trigger scan in background
	go func() {
		if err := h.driftScanner.ScanDrift(r.Context()); err != nil {
			h.logger.Error(err, "Manual drift scan failed")
		}
	}()

	response := map[string]interface{}{
		"status":  "scanning",
		"message": "Drift scan initiated",
	}

	WriteJSON(w, http.StatusAccepted, response)
}
