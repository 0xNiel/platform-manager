/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
)

// Healthz is a simple health check endpoint
func Healthz(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readyz checks if the server is ready to receive traffic
func Readyz(w http.ResponseWriter, r *http.Request) {
	// TODO: Add actual readiness checks (e.g., can connect to K8s API)
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// HealthHandler handles health-related API requests
type HealthHandler struct {
	client client.Client
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(c client.Client) *HealthHandler {
	return &HealthHandler{client: c}
}

// PlatformHealthResponse is the API response for platform health
type PlatformHealthResponse struct {
	OverallHealth       string                        `json:"overallHealth"`
	TotalTenants        int                           `json:"totalTenants"`
	HealthyTenants      int                           `json:"healthyTenants"`
	DegradedTenants     int                           `json:"degradedTenants"`
	CriticalTenants     int                           `json:"criticalTenants"`
	CrossplaneResources ResourceStateCountsResponse   `json:"crossplaneResources"`
	KubernetesResources ResourceStateCountsResponse   `json:"kubernetesResources"`
	Crossplane          *CrossplaneSummaryResponse    `json:"crossplane,omitempty"`
	TotalIAMDrift       int                           `json:"totalIamDrift"`
	ArgoSummary         ArgoSummaryResponse           `json:"argoSummary"`
	Tenants             []TenantHealthSummaryResponse `json:"tenants"`
	TopIssues           []IssueResponse               `json:"topIssues"`
	LastUpdated         string                        `json:"lastUpdated,omitempty"`
}

// CrossplaneSummaryResponse represents Crossplane-specific summary metrics
type CrossplaneSummaryResponse struct {
	Compositions int `json:"compositions"`
	Claims       int `json:"claims"`
	XRs          int `json:"xrs"`
	Failed       int `json:"failed"`
	Paused       int `json:"paused"` // Managed resources with crossplane.io/paused annotation
}

// ResourceStateCountsResponse represents resource state counts in API response
type ResourceStateCountsResponse struct {
	Ready   int `json:"ready"`
	Failed  int `json:"failed"`
	Waiting int `json:"waiting"`
	Unknown int `json:"unknown"`
	Paused  int `json:"paused"`
	Total   int `json:"total"`
}

// ArgoSummaryResponse represents ArgoCD summary in API response
type ArgoSummaryResponse struct {
	TotalApps   int `json:"totalApps"`
	Synced      int `json:"synced"`
	OutOfSync   int `json:"outOfSync"`
	Healthy     int `json:"healthy"`
	Degraded    int `json:"degraded"`
	Progressing int `json:"progressing"`
	Missing     int `json:"missing"`
	Suspended   int `json:"suspended"`
	AutoSyncOff int `json:"autoSyncOff"` // Apps with auto-sync disabled
	PruneOff    int `json:"pruneOff"`    // Apps with prune disabled
	SelfHealOff int `json:"selfHealOff"` // Apps with self-heal disabled
}

// TenantHealthSummaryResponse represents a tenant health summary in API response
type TenantHealthSummaryResponse struct {
	Name            string `json:"name"`
	DisplayName     string `json:"displayName"`
	Health          string `json:"health"`
	FailedResources int    `json:"failedResources"`
	TotalResources  int    `json:"totalResources"`
	IAMDriftCount   int    `json:"iamDriftCount"`
	ArgoOutOfSync   int    `json:"argoOutOfSync"`
}

// IssueResponse represents an issue in API response
type IssueResponse struct {
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	RuleID      string `json:"ruleId,omitempty"`
	ResourceRef string `json:"resourceRef,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// TenantHealthResponse is the API response for tenant health
type TenantHealthResponse struct {
	Name                string                      `json:"name"`
	TenantRef           string                      `json:"tenantRef"`
	OverallHealth       string                      `json:"overallHealth"`
	CrossplaneResources ResourceStateCountsResponse `json:"crossplaneResources"`
	KubernetesResources ResourceStateCountsResponse `json:"kubernetesResources"`
	IAMDrift            IAMDriftSummaryResponse     `json:"iamDrift"`
	Argo                ArgoSummaryResponse         `json:"argo"`
	CPUUsage            string                      `json:"cpuUsage,omitempty"`
	MemoryUsage         string                      `json:"memoryUsage,omitempty"`
	TopIssues           []IssueResponse             `json:"topIssues"`
	LastUpdated         string                      `json:"lastUpdated,omitempty"`
}

// IAMDriftSummaryResponse represents IAM drift summary in API response
type IAMDriftSummaryResponse struct {
	TotalRoles        int    `json:"totalRoles"`
	TotalPolicies     int    `json:"totalPolicies"`
	RolesWithDrift    int    `json:"rolesWithDrift"`
	PoliciesWithDrift int    `json:"policiesWithDrift"`
	ExtraPrivileges   int    `json:"extraPrivileges"`
	MissingPrivileges int    `json:"missingPrivileges"`
	LastChecked       string `json:"lastChecked,omitempty"`
}

// GetPlatformHealth returns the global platform health
func (h *HealthHandler) GetPlatformHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Try to get the PlatformHealth singleton
	platformHealthList := &platformv1alpha1.PlatformHealthList{}
	if err := h.client.List(ctx, platformHealthList); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch platform health")
		return
	}

	// If no PlatformHealth exists, aggregate from TenantHealth resources
	if len(platformHealthList.Items) == 0 {
		response := h.aggregatePlatformHealth(ctx)
		WriteJSON(w, http.StatusOK, response)
		return
	}

	// Use the first (should be singleton) PlatformHealth
	ph := platformHealthList.Items[0]
	response := h.platformHealthToResponse(&ph)
	WriteJSON(w, http.StatusOK, response)
}

// aggregatePlatformHealth creates a platform health response by aggregating tenant health
func (h *HealthHandler) aggregatePlatformHealth(ctx context.Context) *PlatformHealthResponse {
	// Get all TenantHealth resources
	tenantHealthList := &platformv1alpha1.TenantHealthList{}
	if err := h.client.List(ctx, tenantHealthList); err != nil {
		return &PlatformHealthResponse{
			OverallHealth: string(platformv1alpha1.HealthLevelUnknown),
		}
	}

	// Get all Tenants for display names
	tenantList := &platformv1alpha1.TenantList{}
	_ = h.client.List(ctx, tenantList) // Ignore error, just won't have display names

	tenantDisplayNames := make(map[string]string)
	for _, t := range tenantList.Items {
		tenantDisplayNames[t.Name] = t.Spec.DisplayName
	}

	response := &PlatformHealthResponse{
		OverallHealth: string(platformv1alpha1.HealthLevelHealthy),
		TotalTenants:  len(tenantHealthList.Items),
		Tenants:       make([]TenantHealthSummaryResponse, 0),
		TopIssues:     make([]IssueResponse, 0),
	}

	for _, th := range tenantHealthList.Items {
		displayName := tenantDisplayNames[th.Spec.TenantRef]
		if displayName == "" {
			displayName = th.Spec.TenantRef
		}

		summary := TenantHealthSummaryResponse{
			Name:            th.Spec.TenantRef,
			DisplayName:     displayName,
			Health:          string(th.Status.OverallHealth),
			FailedResources: th.Status.CrossplaneResources.Failed + th.Status.KubernetesResources.Failed,
			TotalResources:  th.Status.CrossplaneResources.Total + th.Status.KubernetesResources.Total,
			IAMDriftCount:   th.Status.IAMDrift.RolesWithDrift + th.Status.IAMDrift.PoliciesWithDrift,
			ArgoOutOfSync:   th.Status.Argo.OutOfSync,
		}
		response.Tenants = append(response.Tenants, summary)

		// Aggregate counts
		response.CrossplaneResources.Ready += th.Status.CrossplaneResources.Ready
		response.CrossplaneResources.Failed += th.Status.CrossplaneResources.Failed
		response.CrossplaneResources.Waiting += th.Status.CrossplaneResources.Waiting
		response.CrossplaneResources.Unknown += th.Status.CrossplaneResources.Unknown
		response.CrossplaneResources.Paused += th.Status.CrossplaneResources.Paused
		response.CrossplaneResources.Total += th.Status.CrossplaneResources.Total

		response.KubernetesResources.Ready += th.Status.KubernetesResources.Ready
		response.KubernetesResources.Failed += th.Status.KubernetesResources.Failed
		response.KubernetesResources.Waiting += th.Status.KubernetesResources.Waiting
		response.KubernetesResources.Unknown += th.Status.KubernetesResources.Unknown
		response.KubernetesResources.Paused += th.Status.KubernetesResources.Paused
		response.KubernetesResources.Total += th.Status.KubernetesResources.Total

		response.ArgoSummary.TotalApps += th.Status.Argo.TotalApps
		response.ArgoSummary.Synced += th.Status.Argo.Synced
		response.ArgoSummary.OutOfSync += th.Status.Argo.OutOfSync
		response.ArgoSummary.Healthy += th.Status.Argo.Healthy
		response.ArgoSummary.Degraded += th.Status.Argo.Degraded
		response.ArgoSummary.Progressing += th.Status.Argo.Progressing

		response.TotalIAMDrift += th.Status.IAMDrift.RolesWithDrift + th.Status.IAMDrift.PoliciesWithDrift

		// Count tenant health levels
		switch th.Status.OverallHealth {
		case platformv1alpha1.HealthLevelHealthy:
			response.HealthyTenants++
		case platformv1alpha1.HealthLevelDegraded:
			response.DegradedTenants++
		case platformv1alpha1.HealthLevelCritical:
			response.CriticalTenants++
		}
	}

	// Determine overall health
	if response.CriticalTenants > 0 {
		response.OverallHealth = string(platformv1alpha1.HealthLevelCritical)
	} else if response.DegradedTenants > 0 {
		response.OverallHealth = string(platformv1alpha1.HealthLevelDegraded)
	} else if response.HealthyTenants > 0 {
		response.OverallHealth = string(platformv1alpha1.HealthLevelHealthy)
	} else {
		response.OverallHealth = string(platformv1alpha1.HealthLevelUnknown)
	}

	// Add Crossplane-specific stats
	response.Crossplane = h.getCrossplaneSummary(ctx)

	// Add ArgoCD sync policy stats
	h.enrichArgoSyncPolicyStats(ctx, &response.ArgoSummary)

	return response
}

// platformHealthToResponse converts a PlatformHealth to API response
func (h *HealthHandler) platformHealthToResponse(ph *platformv1alpha1.PlatformHealth) *PlatformHealthResponse {
	response := &PlatformHealthResponse{
		OverallHealth:   string(ph.Status.OverallHealth),
		TotalTenants:    ph.Status.TotalTenants,
		HealthyTenants:  ph.Status.HealthyTenants,
		DegradedTenants: ph.Status.DegradedTenants,
		CriticalTenants: ph.Status.CriticalTenants,
		TotalIAMDrift:   ph.Status.TotalIAMDrift,
		CrossplaneResources: ResourceStateCountsResponse{
			Ready:   ph.Status.CrossplaneResources.Ready,
			Failed:  ph.Status.CrossplaneResources.Failed,
			Waiting: ph.Status.CrossplaneResources.Waiting,
			Unknown: ph.Status.CrossplaneResources.Unknown,
			Paused:  ph.Status.CrossplaneResources.Paused,
			Total:   ph.Status.CrossplaneResources.Total,
		},
		KubernetesResources: ResourceStateCountsResponse{
			Ready:   ph.Status.KubernetesResources.Ready,
			Failed:  ph.Status.KubernetesResources.Failed,
			Waiting: ph.Status.KubernetesResources.Waiting,
			Unknown: ph.Status.KubernetesResources.Unknown,
			Paused:  ph.Status.KubernetesResources.Paused,
			Total:   ph.Status.KubernetesResources.Total,
		},
		ArgoSummary: ArgoSummaryResponse{
			TotalApps:   ph.Status.ArgoSummary.TotalApps,
			Synced:      ph.Status.ArgoSummary.Synced,
			OutOfSync:   ph.Status.ArgoSummary.OutOfSync,
			Healthy:     ph.Status.ArgoSummary.Healthy,
			Degraded:    ph.Status.ArgoSummary.Degraded,
			Progressing: ph.Status.ArgoSummary.Progressing,
		},
		Tenants:   make([]TenantHealthSummaryResponse, 0, len(ph.Status.Tenants)),
		TopIssues: make([]IssueResponse, 0, len(ph.Status.TopIssues)),
	}

	if ph.Status.LastUpdated != nil {
		response.LastUpdated = ph.Status.LastUpdated.Format("2006-01-02T15:04:05Z")
	}

	for _, t := range ph.Status.Tenants {
		response.Tenants = append(response.Tenants, TenantHealthSummaryResponse{
			Name:            t.Name,
			DisplayName:     t.DisplayName,
			Health:          string(t.Health),
			FailedResources: t.FailedResources,
			TotalResources:  t.TotalResources,
			IAMDriftCount:   t.IAMDriftCount,
			ArgoOutOfSync:   t.ArgoOutOfSync,
		})
	}

	for _, i := range ph.Status.TopIssues {
		response.TopIssues = append(response.TopIssues, IssueResponse{
			Severity:    string(i.Severity),
			Message:     i.Message,
			RuleID:      i.RuleID,
			ResourceRef: i.ResourceRef,
			Timestamp:   i.Timestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	return response
}

// ListTenantHealth returns health for all tenants
func (h *HealthHandler) ListTenantHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantHealthList := &platformv1alpha1.TenantHealthList{}
	if err := h.client.List(ctx, tenantHealthList); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to list tenant health")
		return
	}

	responses := make([]TenantHealthResponse, 0, len(tenantHealthList.Items))
	for _, th := range tenantHealthList.Items {
		responses = append(responses, h.tenantHealthToResponse(&th))
	}

	WriteJSON(w, http.StatusOK, responses)
}

// GetTenantHealth returns health for a specific tenant
func (h *HealthHandler) GetTenantHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "id")

	// Look for TenantHealth by tenant reference
	tenantHealthList := &platformv1alpha1.TenantHealthList{}
	if err := h.client.List(ctx, tenantHealthList); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch tenant health")
		return
	}

	for _, th := range tenantHealthList.Items {
		if th.Spec.TenantRef == tenantID || th.Name == tenantID || th.Name == tenantID+"-health" {
			response := h.tenantHealthToResponse(&th)
			WriteJSON(w, http.StatusOK, response)
			return
		}
	}

	WriteError(w, http.StatusNotFound, "Tenant health not found")
}

// tenantHealthToResponse converts a TenantHealth to API response
func (h *HealthHandler) tenantHealthToResponse(th *platformv1alpha1.TenantHealth) TenantHealthResponse {
	response := TenantHealthResponse{
		Name:          th.Name,
		TenantRef:     th.Spec.TenantRef,
		OverallHealth: string(th.Status.OverallHealth),
		CrossplaneResources: ResourceStateCountsResponse{
			Ready:   th.Status.CrossplaneResources.Ready,
			Failed:  th.Status.CrossplaneResources.Failed,
			Waiting: th.Status.CrossplaneResources.Waiting,
			Unknown: th.Status.CrossplaneResources.Unknown,
			Paused:  th.Status.CrossplaneResources.Paused,
			Total:   th.Status.CrossplaneResources.Total,
		},
		KubernetesResources: ResourceStateCountsResponse{
			Ready:   th.Status.KubernetesResources.Ready,
			Failed:  th.Status.KubernetesResources.Failed,
			Waiting: th.Status.KubernetesResources.Waiting,
			Unknown: th.Status.KubernetesResources.Unknown,
			Paused:  th.Status.KubernetesResources.Paused,
			Total:   th.Status.KubernetesResources.Total,
		},
		IAMDrift: IAMDriftSummaryResponse{
			TotalRoles:        th.Status.IAMDrift.TotalRoles,
			TotalPolicies:     th.Status.IAMDrift.TotalPolicies,
			RolesWithDrift:    th.Status.IAMDrift.RolesWithDrift,
			PoliciesWithDrift: th.Status.IAMDrift.PoliciesWithDrift,
			ExtraPrivileges:   th.Status.IAMDrift.ExtraPrivileges,
			MissingPrivileges: th.Status.IAMDrift.MissingPrivileges,
		},
		Argo: ArgoSummaryResponse{
			TotalApps:   th.Status.Argo.TotalApps,
			Synced:      th.Status.Argo.Synced,
			OutOfSync:   th.Status.Argo.OutOfSync,
			Healthy:     th.Status.Argo.Healthy,
			Degraded:    th.Status.Argo.Degraded,
			Progressing: th.Status.Argo.Progressing,
		},
		TopIssues: make([]IssueResponse, 0, len(th.Status.TopIssues)),
	}

	if th.Status.LastUpdated != nil {
		response.LastUpdated = th.Status.LastUpdated.Format("2006-01-02T15:04:05Z")
	}

	if th.Status.IAMDrift.LastChecked != nil {
		response.IAMDrift.LastChecked = th.Status.IAMDrift.LastChecked.Format("2006-01-02T15:04:05Z")
	}

	if !th.Status.CPUUsage.IsZero() {
		response.CPUUsage = th.Status.CPUUsage.String()
	}
	if !th.Status.MemoryUsage.IsZero() {
		response.MemoryUsage = th.Status.MemoryUsage.String()
	}

	for _, i := range th.Status.TopIssues {
		response.TopIssues = append(response.TopIssues, IssueResponse{
			Severity:    string(i.Severity),
			Message:     i.Message,
			RuleID:      i.RuleID,
			ResourceRef: i.ResourceRef,
			Timestamp:   i.Timestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	return response
}

// getCrossplaneSummary queries the cluster for Crossplane-specific resources
func (h *HealthHandler) getCrossplaneSummary(ctx context.Context) *CrossplaneSummaryResponse {
	summary := &CrossplaneSummaryResponse{
		Compositions: 0,
		Claims:       0,
		XRs:          0,
		Failed:       0,
		Paused:       0,
	}

	// Count Compositions using unstructured
	compositions := &unstructured.UnstructuredList{}
	compositions.SetAPIVersion("apiextensions.crossplane.io/v1")
	compositions.SetKind("CompositionList")
	if err := h.client.List(ctx, compositions); err == nil {
		summary.Compositions = len(compositions.Items)
	}

	// Count XRs and claims for every CompositeResourceDefinition in the cluster.
	xrds := &unstructured.UnstructuredList{}
	xrds.SetAPIVersion("apiextensions.crossplane.io/v1")
	xrds.SetKind("CompositeResourceDefinitionList")
	if err := h.client.List(ctx, xrds); err == nil {
		for _, xrd := range xrds.Items {
			h.countCompositeResources(ctx, &xrd, summary)
		}
	}

	// Count paused Crossplane Managed Resources
	// Check ResourceSummaries with category=Crossplane or IAM and state=Paused
	resourceSummaryList := &platformv1alpha1.ResourceSummaryList{}
	if err := h.client.List(ctx, resourceSummaryList); err == nil {
		for _, rs := range resourceSummaryList.Items {
			if (rs.Spec.Category == platformv1alpha1.ResourceCategoryCrossplane ||
				rs.Spec.Category == platformv1alpha1.ResourceCategoryIAM) &&
				rs.Status.State == "Paused" {
				summary.Paused++
			}
		}
	}

	return summary
}

// countCompositeResources adds one XRD's composite resources and claims to the
// summary. An XR counts as failed when its last reconcile errored.
func (h *HealthHandler) countCompositeResources(ctx context.Context, xrd *unstructured.Unstructured, summary *CrossplaneSummaryResponse) {
	group, _, _ := unstructured.NestedString(xrd.Object, "spec", "group")
	kind, _, _ := unstructured.NestedString(xrd.Object, "spec", "names", "kind")
	claimKind, _, _ := unstructured.NestedString(xrd.Object, "spec", "claimNames", "kind")
	version := xrdVersion(xrd)
	if group == "" || kind == "" || version == "" {
		return
	}
	apiVersion := group + "/" + version

	xrs := &unstructured.UnstructuredList{}
	xrs.SetAPIVersion(apiVersion)
	xrs.SetKind(kind + "List")
	if err := h.client.List(ctx, xrs); err == nil {
		summary.XRs += len(xrs.Items)
		for _, xr := range xrs.Items {
			if hasReconcileError(&xr) {
				summary.Failed++
			}
		}
	}

	if claimKind == "" {
		return
	}
	claims := &unstructured.UnstructuredList{}
	claims.SetAPIVersion(apiVersion)
	claims.SetKind(claimKind + "List")
	if err := h.client.List(ctx, claims); err == nil {
		summary.Claims += len(claims.Items)
	}
}

// xrdVersion returns the XRD's referenceable version, or its first served one.
func xrdVersion(xrd *unstructured.Unstructured) string {
	versions, _, _ := unstructured.NestedSlice(xrd.Object, "spec", "versions")
	served := ""
	for _, v := range versions {
		version, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := version["name"].(string)
		if referenceable, _ := version["referenceable"].(bool); referenceable {
			return name
		}
		if isServed, _ := version["served"].(bool); isServed && served == "" {
			served = name
		}
	}
	return served
}

// hasReconcileError reports whether a Crossplane object's Synced condition is
// False with reason ReconcileError.
func hasReconcileError(obj *unstructured.Unstructured) bool {
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, c := range conditions {
		condition, ok := c.(map[string]interface{})
		if ok && condition["type"] == "Synced" && condition["status"] == "False" &&
			condition["reason"] == "ReconcileError" {
			return true
		}
	}
	return false
}

// enrichArgoSyncPolicyStats queries ArgoCD Applications and counts sync policy settings
func (h *HealthHandler) enrichArgoSyncPolicyStats(ctx context.Context, argoSummary *ArgoSummaryResponse) {
	// Query all ArgoCD Applications
	apps := &unstructured.UnstructuredList{}
	apps.SetAPIVersion("argoproj.io/v1alpha1")
	apps.SetKind("ApplicationList")

	if err := h.client.List(ctx, apps); err != nil {
		// If we can't list apps, just return (counts stay at 0)
		return
	}

	for _, app := range apps.Items {
		spec, found, _ := unstructured.NestedMap(app.Object, "spec")
		if !found {
			continue
		}

		// Check syncPolicy
		syncPolicy, found, _ := unstructured.NestedMap(spec, "syncPolicy")
		if !found {
			// No syncPolicy defined = auto-sync off, prune off, self-heal off
			argoSummary.AutoSyncOff++
			argoSummary.PruneOff++
			argoSummary.SelfHealOff++
			continue
		}

		// Check automated sync
		automated, found, _ := unstructured.NestedMap(syncPolicy, "automated")
		if !found {
			// No automated section = auto-sync off
			argoSummary.AutoSyncOff++
			argoSummary.PruneOff++
			argoSummary.SelfHealOff++
			continue
		}

		// Check prune
		if prune, found := automated["prune"].(bool); !found || !prune {
			argoSummary.PruneOff++
		}

		// Check selfHeal
		if selfHeal, found := automated["selfHeal"].(bool); !found || !selfHeal {
			argoSummary.SelfHealOff++
		}
	}
}
