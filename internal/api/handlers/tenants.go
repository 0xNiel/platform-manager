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

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/platform-manager/platform-manager/api/v1alpha1"
)

// TenantHandler handles tenant-related API requests
type TenantHandler struct {
	client client.Client
}

// NewTenantHandler creates a new TenantHandler
func NewTenantHandler(c client.Client) *TenantHandler {
	return &TenantHandler{client: c}
}

// TenantResponse is the API response for a tenant
type TenantResponse struct {
	Name           string            `json:"name"`
	DisplayName    string            `json:"displayName"`
	Description    string            `json:"description,omitempty"`
	Phase          string            `json:"phase"`
	Namespaces     []string          `json:"namespaces"`
	ArgoProjects   []string          `json:"argoProjects,omitempty"`
	AWSAccounts    []string          `json:"awsAccounts,omitempty"`
	Contacts       []ContactResponse `json:"contacts,omitempty"`
	CostCenter     string            `json:"costCenter,omitempty"`
	Environment    string            `json:"environment,omitempty"`
	NamespaceCount int               `json:"namespaceCount"`
	ResourceCount  int               `json:"resourceCount"`
	HealthRef      string            `json:"healthRef,omitempty"`
	LastReconciled string            `json:"lastReconciled,omitempty"`
	CreatedAt      string            `json:"createdAt"`
}

// ContactResponse is the API response for a tenant contact
type ContactResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

// List returns all tenants
func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantList := &platformv1alpha1.TenantList{}
	if err := h.client.List(ctx, tenantList); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list tenants")
		return
	}

	responses := make([]TenantResponse, 0, len(tenantList.Items))
	for _, tenant := range tenantList.Items {
		responses = append(responses, h.tenantToResponse(&tenant))
	}

	writeJSON(w, http.StatusOK, responses)
}

// Get returns a specific tenant
func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "id")

	tenant := &platformv1alpha1.Tenant{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: tenantID}, tenant); err != nil {
		writeError(w, http.StatusNotFound, "Tenant not found")
		return
	}

	response := h.tenantToResponse(tenant)
	writeJSON(w, http.StatusOK, response)
}

// ListResources returns resources for a specific tenant
func (h *TenantHandler) ListResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "id")

	// Query parameters for filtering
	state := r.URL.Query().Get("state")
	kind := r.URL.Query().Get("kind")
	category := r.URL.Query().Get("category")

	// Get ResourceSummary objects for this tenant
	resourceList := &platformv1alpha1.ResourceSummaryList{}
	if err := h.client.List(ctx, resourceList); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list resources")
		return
	}

	responses := make([]ResourceSummaryResponse, 0)
	for _, rs := range resourceList.Items {
		// Filter by tenant
		if rs.Spec.TenantRef != tenantID {
			continue
		}

		// Apply filters
		if state != "" && string(rs.Status.State) != state {
			continue
		}
		if kind != "" && rs.Spec.Kind != kind {
			continue
		}
		if category != "" && string(rs.Spec.Category) != category {
			continue
		}

		responses = append(responses, resourceSummaryToResponse(&rs))
	}

	writeJSON(w, http.StatusOK, responses)
}

// ResourceSummaryResponse is the API response for a resource summary
type ResourceSummaryResponse struct {
	Name           string `json:"name"`
	Group          string `json:"group,omitempty"`
	Version        string `json:"version"`
	Kind           string `json:"kind"`
	Namespace      string `json:"namespace,omitempty"`
	ResourceName   string `json:"resourceName"`
	TenantRef      string `json:"tenantRef,omitempty"`
	Category       string `json:"category,omitempty"`
	Provider       string `json:"provider,omitempty"`
	State          string `json:"state"`
	Message        string `json:"message,omitempty"`
	Age            string `json:"age,omitempty"`
	LastTransition string `json:"lastTransition,omitempty"`
	LastSeen       string `json:"lastSeen,omitempty"`
	SyncStatus     string `json:"syncStatus,omitempty"`
	HealthStatus   string `json:"healthStatus,omitempty"`
	DriftDetected  bool   `json:"driftDetected,omitempty"`
}

// resourceSummaryToResponse converts a ResourceSummary to API response
func resourceSummaryToResponse(rs *platformv1alpha1.ResourceSummary) ResourceSummaryResponse {
	response := ResourceSummaryResponse{
		Name:          rs.Name,
		Group:         rs.Spec.Group,
		Version:       rs.Spec.Version,
		Kind:          rs.Spec.Kind,
		Namespace:     rs.Spec.Namespace,
		ResourceName:  rs.Spec.Name,
		TenantRef:     rs.Spec.TenantRef,
		Category:      string(rs.Spec.Category),
		Provider:      rs.Spec.Provider,
		State:         string(rs.Status.State),
		Message:       rs.Status.Message,
		Age:           rs.Status.Age,
		SyncStatus:    rs.Status.SyncStatus,
		HealthStatus:  rs.Status.HealthStatus,
		DriftDetected: rs.Status.DriftDetected,
	}

	if rs.Status.LastTransition != nil {
		response.LastTransition = rs.Status.LastTransition.Format("2006-01-02T15:04:05Z")
	}
	if rs.Status.LastSeen != nil {
		response.LastSeen = rs.Status.LastSeen.Format("2006-01-02T15:04:05Z")
	}

	return response
}

// tenantToResponse converts a Tenant to API response
func (h *TenantHandler) tenantToResponse(tenant *platformv1alpha1.Tenant) TenantResponse {
	response := TenantResponse{
		Name:           tenant.Name,
		DisplayName:    tenant.Spec.DisplayName,
		Description:    tenant.Spec.Description,
		Phase:          string(tenant.Status.Phase),
		Namespaces:     tenant.Spec.Namespaces,
		ArgoProjects:   tenant.Spec.ArgoProjects,
		AWSAccounts:    tenant.Spec.AWSAccounts,
		CostCenter:     tenant.Spec.CostCenter,
		Environment:    tenant.Spec.Environment,
		NamespaceCount: tenant.Status.NamespaceCount,
		ResourceCount:  tenant.Status.ResourceCount,
		HealthRef:      tenant.Status.HealthRef,
		CreatedAt:      tenant.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
	}

	if tenant.Status.LastReconciled != nil {
		response.LastReconciled = tenant.Status.LastReconciled.Format("2006-01-02T15:04:05Z")
	}

	if tenant.Spec.Namespaces == nil {
		response.Namespaces = []string{}
	}

	response.Contacts = make([]ContactResponse, 0, len(tenant.Spec.Contacts))
	for _, c := range tenant.Spec.Contacts {
		response.Contacts = append(response.Contacts, ContactResponse{
			Name:  c.Name,
			Email: c.Email,
			Role:  c.Role,
		})
	}

	return response
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
