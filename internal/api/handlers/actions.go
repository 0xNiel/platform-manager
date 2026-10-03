/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/0xNiel/platform-manager/internal/api/middleware"
	platformmetrics "github.com/0xNiel/platform-manager/internal/metrics"
)

var actionsLog = logf.Log.WithName("actions-handler")

// crossplanePausedAnnotation stops Crossplane from reconciling a resource when set to "true".
const (
	crossplanePausedAnnotation = "crossplane.io/paused"
	annotationValueTrue        = "true"
)

// ActionsHandler handles mutating operations on resources
type ActionsHandler struct {
	client      client.Client
	auditLogger middleware.AuditLogger
}

// NewActionsHandler creates a new ActionsHandler
func NewActionsHandler(c client.Client, auditLogger middleware.AuditLogger) *ActionsHandler {
	return &ActionsHandler{
		client:      c,
		auditLogger: auditLogger,
	}
}

// === Request/Response Types ===

// ArgoSyncRequest represents a request to sync an ArgoCD application
type ArgoSyncRequest struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Prune     bool   `json:"prune"`
	Force     bool   `json:"force"`
	DryRun    bool   `json:"dryRun"`
}

// ArgoRefreshRequest represents a request to refresh an ArgoCD application
type ArgoRefreshRequest struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// CrossplaneResourceRequest represents a request to act on a Crossplane resource
type CrossplaneResourceRequest struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

// DeleteResourceRequest represents a request to delete a resource
type DeleteResourceRequest struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	Confirm   string `json:"confirm"` // Must match resource name
}

// ActionResponse is a generic response for action operations
type ActionResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// actionMetricsWrapper wraps action execution with metrics
func (h *ActionsHandler) recordActionMetrics(actionName string, userRole string, startTime time.Time, err error) {
	duration := time.Since(startTime)
	success := err == nil
	platformmetrics.ActionsMetrics.RecordAction(actionName, userRole, success, duration)
}

// === ArgoCD Actions ===

// SyncArgoApp syncs an ArgoCD application
// POST /api/v1/actions/argo/sync
func (h *ActionsHandler) SyncArgoApp(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()
	user := middleware.UserFromContext(ctx)
	var actionErr error
	defer func() {
		h.recordActionMetrics(platformmetrics.ActionArgoSync, string(user.Role), startTime, actionErr)
	}()

	var req ArgoSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		actionsLog.Error(err, "Failed to decode sync request")
		actionErr = err
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Name == "" || req.Namespace == "" {
		actionErr = errors.New("missing required fields")
		WriteError(w, http.StatusBadRequest, "name and namespace are required")
		return
	}

	actionsLog.Info("Syncing ArgoCD application", "name", req.Name, "namespace", req.Namespace, "user", user.Username)

	// Fetch the Application
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	err := h.client.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, app)

	if err != nil {
		actionsLog.Error(err, "Failed to get ArgoCD application", "name", req.Name)
		actionErr = err
		h.auditLogger.LogAction(ctx, "argo:sync", req.Name, false, err)
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Application not found: %s", req.Name))
		return
	}

	// Check if already syncing (unless force is specified)
	if !req.Force {
		operation, found, _ := unstructured.NestedMap(app.Object, "operation")
		if found && operation != nil {
			msg := "Application already has an operation in progress. Use force sync to override."
			actionsLog.Info(msg, "name", req.Name)
			actionErr = errors.New("operation in progress")
			h.auditLogger.LogAction(ctx, "argo:sync", req.Name, false, actionErr)
			WriteError(w, http.StatusConflict, msg)
			return
		}
	}

	// Set sync operation
	syncOperation := map[string]interface{}{
		"sync": map[string]interface{}{
			"syncStrategy": map[string]interface{}{
				"hook": map[string]interface{}{
					"force": req.Force,
				},
			},
			"prune":  req.Prune,
			"dryRun": req.DryRun,
		},
	}

	if err := unstructured.SetNestedMap(app.Object, syncOperation, "operation"); err != nil {
		actionsLog.Error(err, "Failed to set sync operation", "name", req.Name)
		actionErr = err
		h.auditLogger.LogAction(ctx, "argo:sync", req.Name, false, err)
		WriteError(w, http.StatusInternalServerError, "Failed to set sync operation")
		return
	}

	// Update the application
	if err := h.client.Update(ctx, app); err != nil {
		actionsLog.Error(err, "Failed to update ArgoCD application", "name", req.Name)
		actionErr = err
		h.auditLogger.LogAction(ctx, "argo:sync", req.Name, false, err)
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update application: %v", err))
		return
	}

	actionsLog.Info("Successfully initiated sync", "name", req.Name, "prune", req.Prune, "force", req.Force, "dryRun", req.DryRun)
	h.auditLogger.LogAction(ctx, "argo:sync", req.Name, true, nil)

	WriteJSON(w, http.StatusOK, ActionResponse{
		Success: true,
		Message: "Sync operation initiated",
		Details: map[string]interface{}{
			"application": req.Name,
			"namespace":   req.Namespace,
			"prune":       req.Prune,
			"force":       req.Force,
			"dryRun":      req.DryRun,
		},
	})
}

// RefreshArgoApp refreshes an ArgoCD application
// POST /api/v1/actions/argo/refresh
func (h *ActionsHandler) RefreshArgoApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.UserFromContext(ctx)

	var req ArgoRefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		actionsLog.Error(err, "Failed to decode refresh request")
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Name == "" || req.Namespace == "" {
		WriteError(w, http.StatusBadRequest, "name and namespace are required")
		return
	}

	actionsLog.Info("Refreshing ArgoCD application", "name", req.Name, "namespace", req.Namespace, "user", user.Username)

	// Fetch the Application
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	err := h.client.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, app)

	if err != nil {
		actionsLog.Error(err, "Failed to get ArgoCD application", "name", req.Name)
		h.auditLogger.LogAction(ctx, "argo:refresh", req.Name, false, err)
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Application not found: %s", req.Name))
		return
	}

	// Add refresh annotation
	annotations := app.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["argocd.argoproj.io/refresh"] = "normal"
	app.SetAnnotations(annotations)

	// Update the application
	if err := h.client.Update(ctx, app); err != nil {
		actionsLog.Error(err, "Failed to update ArgoCD application", "name", req.Name)
		h.auditLogger.LogAction(ctx, "argo:refresh", req.Name, false, err)
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update application: %v", err))
		return
	}

	actionsLog.Info("Successfully initiated refresh", "name", req.Name)
	h.auditLogger.LogAction(ctx, "argo:refresh", req.Name, true, nil)

	WriteJSON(w, http.StatusOK, ActionResponse{
		Success: true,
		Message: "Refresh initiated",
		Details: map[string]interface{}{
			"application": req.Name,
			"namespace":   req.Namespace,
		},
	})
}

// === Crossplane Actions ===

// PauseCrossplaneResource pauses a Crossplane resource
// POST /api/v1/actions/crossplane/pause
func (h *ActionsHandler) PauseCrossplaneResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.UserFromContext(ctx)

	var req CrossplaneResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		actionsLog.Error(err, "Failed to decode pause request")
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Group == "" || req.Version == "" || req.Kind == "" || req.Name == "" {
		WriteError(w, http.StatusBadRequest, "group, version, kind, and name are required")
		return
	}

	resourceID := fmt.Sprintf("%s/%s/%s/%s", req.Group, req.Version, req.Kind, req.Name)
	actionsLog.Info("Pausing Crossplane resource", "resource", resourceID, "user", user.Username)

	// Fetch the resource
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   req.Group,
		Version: req.Version,
		Kind:    req.Kind,
	})

	err := h.client.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, resource)

	if err != nil {
		actionsLog.Error(err, "Failed to get resource", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "crossplane:pause", req.Name, resourceID, false, err)
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Resource not found: %s", resourceID))
		return
	}

	// Add pause annotation
	annotations := resource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Check if already paused
	if annotations[crossplanePausedAnnotation] == annotationValueTrue {
		actionsLog.Info("Resource already paused", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "crossplane:pause", req.Name, resourceID, true, nil)
		WriteJSON(w, http.StatusOK, ActionResponse{
			Success: true,
			Message: "Resource already paused",
			Details: map[string]interface{}{
				"resource": resourceID,
				"paused":   true,
			},
		})
		return
	}

	annotations[crossplanePausedAnnotation] = annotationValueTrue
	resource.SetAnnotations(annotations)

	// Update the resource
	if err := h.client.Update(ctx, resource); err != nil {
		actionsLog.Error(err, "Failed to update resource", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "crossplane:pause", req.Name, resourceID, false, err)
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pause resource: %v", err))
		return
	}

	actionsLog.Info("Successfully paused resource", "resource", resourceID)
	h.auditLogger.LogActionWithDetails(ctx, "crossplane:pause", req.Name, resourceID, true, nil)

	WriteJSON(w, http.StatusOK, ActionResponse{
		Success: true,
		Message: "Resource paused",
		Details: map[string]interface{}{
			"resource": resourceID,
			"paused":   true,
		},
	})
}

// UnpauseCrossplaneResource unpauses a Crossplane resource
// POST /api/v1/actions/crossplane/unpause
func (h *ActionsHandler) UnpauseCrossplaneResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.UserFromContext(ctx)

	var req CrossplaneResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		actionsLog.Error(err, "Failed to decode unpause request")
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Group == "" || req.Version == "" || req.Kind == "" || req.Name == "" {
		WriteError(w, http.StatusBadRequest, "group, version, kind, and name are required")
		return
	}

	resourceID := fmt.Sprintf("%s/%s/%s/%s", req.Group, req.Version, req.Kind, req.Name)
	actionsLog.Info("Unpausing Crossplane resource", "resource", resourceID, "user", user.Username)

	// Fetch the resource
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   req.Group,
		Version: req.Version,
		Kind:    req.Kind,
	})

	err := h.client.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, resource)

	if err != nil {
		actionsLog.Error(err, "Failed to get resource", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "crossplane:unpause", req.Name, resourceID, false, err)
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Resource not found: %s", resourceID))
		return
	}

	// Remove pause annotation
	annotations := resource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Check if not paused
	if annotations[crossplanePausedAnnotation] != annotationValueTrue {
		actionsLog.Info("Resource not paused", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "crossplane:unpause", req.Name, resourceID, true, nil)
		WriteJSON(w, http.StatusOK, ActionResponse{
			Success: true,
			Message: "Resource not paused",
			Details: map[string]interface{}{
				"resource": resourceID,
				"paused":   false,
			},
		})
		return
	}

	delete(annotations, crossplanePausedAnnotation)
	resource.SetAnnotations(annotations)

	// Update the resource
	if err := h.client.Update(ctx, resource); err != nil {
		actionsLog.Error(err, "Failed to update resource", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "crossplane:unpause", req.Name, resourceID, false, err)
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to unpause resource: %v", err))
		return
	}

	actionsLog.Info("Successfully unpaused resource", "resource", resourceID)
	h.auditLogger.LogActionWithDetails(ctx, "crossplane:unpause", req.Name, resourceID, true, nil)

	WriteJSON(w, http.StatusOK, ActionResponse{
		Success: true,
		Message: "Resource unpaused",
		Details: map[string]interface{}{
			"resource": resourceID,
			"paused":   false,
		},
	})
}

// ReconcileCrossplaneResource forces reconciliation of a Crossplane resource
// POST /api/v1/actions/crossplane/reconcile
func (h *ActionsHandler) ReconcileCrossplaneResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.UserFromContext(ctx)

	var req CrossplaneResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		actionsLog.Error(err, "Failed to decode reconcile request")
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Group == "" || req.Version == "" || req.Kind == "" || req.Name == "" {
		WriteError(w, http.StatusBadRequest, "group, version, kind, and name are required")
		return
	}

	resourceID := fmt.Sprintf("%s/%s/%s/%s", req.Group, req.Version, req.Kind, req.Name)
	actionsLog.Info("Forcing reconciliation of Crossplane resource", "resource", resourceID, "user", user.Username)

	// Fetch the resource
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   req.Group,
		Version: req.Version,
		Kind:    req.Kind,
	})

	err := h.client.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, resource)

	if err != nil {
		actionsLog.Error(err, "Failed to get resource", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "crossplane:reconcile", req.Name, resourceID, false, err)
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Resource not found: %s", resourceID))
		return
	}

	// Add reconcile annotation with timestamp
	annotations := resource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["crossplane.io/reconcile"] = fmt.Sprintf("now-%d", time.Now().Unix())
	resource.SetAnnotations(annotations)

	// Update the resource
	if err := h.client.Update(ctx, resource); err != nil {
		actionsLog.Error(err, "Failed to update resource", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "crossplane:reconcile", req.Name, resourceID, false, err)
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to force reconcile: %v", err))
		return
	}

	actionsLog.Info("Successfully requested reconciliation", "resource", resourceID)
	h.auditLogger.LogActionWithDetails(ctx, "crossplane:reconcile", req.Name, resourceID, true, nil)

	WriteJSON(w, http.StatusOK, ActionResponse{
		Success: true,
		Message: "Reconciliation requested",
		Details: map[string]interface{}{
			"resource": resourceID,
		},
	})
}

// === Resource Delete ===

// DeleteResource deletes a resource with safeguards
// DELETE /api/v1/actions/resources
func (h *ActionsHandler) DeleteResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.UserFromContext(ctx)

	var req DeleteResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		actionsLog.Error(err, "Failed to decode delete request")
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Group == "" || req.Version == "" || req.Kind == "" || req.Name == "" {
		WriteError(w, http.StatusBadRequest, "group, version, kind, and name are required")
		return
	}

	// Validate confirmation
	if req.Confirm != req.Name {
		WriteError(w, http.StatusBadRequest, "confirmation does not match resource name")
		return
	}

	resourceID := fmt.Sprintf("%s/%s/%s/%s", req.Group, req.Version, req.Kind, req.Name)
	actionsLog.Info("Deleting resource", "resource", resourceID, "user", user.Username)

	// Fetch the resource
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   req.Group,
		Version: req.Version,
		Kind:    req.Kind,
	})

	err := h.client.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, resource)

	if err != nil {
		actionsLog.Error(err, "Failed to get resource", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "resource:delete", req.Name, resourceID, false, err)
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Resource not found: %s", resourceID))
		return
	}

	// Apply safeguards
	if err := h.canDelete(resource); err != nil {
		actionsLog.Info("Delete prevented by safeguard", "resource", resourceID, "reason", err.Error())
		h.auditLogger.LogActionWithDetails(ctx, "resource:delete", req.Name, resourceID, false, err)
		WriteError(w, http.StatusForbidden, err.Error())
		return
	}

	// Delete the resource
	if err := h.client.Delete(ctx, resource); err != nil {
		actionsLog.Error(err, "Failed to delete resource", "resource", resourceID)
		h.auditLogger.LogActionWithDetails(ctx, "resource:delete", req.Name, resourceID, false, err)
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete resource: %v", err))
		return
	}

	actionsLog.Info("Successfully deleted resource", "resource", resourceID)
	h.auditLogger.LogActionWithDetails(ctx, "resource:delete", req.Name, resourceID, true, nil)

	WriteJSON(w, http.StatusOK, ActionResponse{
		Success: true,
		Message: "Resource deleted",
		Details: map[string]interface{}{
			"resource": resourceID,
		},
	})
}

// canDelete checks if a resource can be safely deleted
func (h *ActionsHandler) canDelete(resource *unstructured.Unstructured) error {
	namespace := resource.GetNamespace()
	gvk := resource.GetObjectKind().GroupVersionKind()

	// Safeguard 1: System namespaces
	systemNamespaces := []string{"kube-system", "kube-public", "default", "kube-node-lease"}
	for _, ns := range systemNamespaces {
		if namespace == ns {
			return fmt.Errorf("cannot delete resources in system namespace: %s", namespace)
		}
	}

	// Safeguard 2: Crossplane Providers
	if gvk.Group == "pkg.crossplane.io" && gvk.Kind == "Provider" {
		return errors.New("cannot delete Crossplane providers")
	}

	// Safeguard 3: Tenant CRs (could add force flag later)
	if gvk.Group == "platform.platform.io" && gvk.Kind == "Tenant" {
		return errors.New("cannot delete Tenant CRs")
	}

	return nil
}
