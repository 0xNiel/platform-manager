/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
)

// ArgoWatcher scans ArgoCD Applications and updates TenantHealth
type ArgoWatcher struct {
	client client.Client
}

// NewArgoWatcher creates a new ArgoWatcher
func NewArgoWatcher(c client.Client) *ArgoWatcher {
	return &ArgoWatcher{
		client: c,
	}
}

// ArgoCD Application GVK
var argoAppGVK = schema.GroupVersionKind{
	Group:   "argoproj.io",
	Version: "v1alpha1",
	Kind:    "Application",
}

// ScanArgoApplications scans ArgoCD applications for a tenant and returns summary
func (w *ArgoWatcher) ScanArgoApplications(ctx context.Context, tenant *platformv1alpha1.Tenant) (platformv1alpha1.ArgoSummary, error) {
	log := logf.FromContext(ctx).WithName("argo-watcher")

	summary := platformv1alpha1.ArgoSummary{}

	// List all ArgoCD Applications
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   argoAppGVK.Group,
		Version: argoAppGVK.Version,
		Kind:    "ApplicationList",
	})

	// ArgoCD apps are typically in the argocd namespace
	if err := w.client.List(ctx, list, client.InNamespace("argocd")); err != nil {
		log.V(1).Info("Failed to list ArgoCD applications", "error", err)
		// CRD might not be installed or no access
		return summary, nil
	}

	// Filter applications belonging to this tenant
	for _, app := range list.Items {
		if !w.belongsToTenant(&app, tenant) {
			continue
		}

		summary.TotalApps++

		// Parse sync status
		syncStatus := w.getSyncStatus(&app)
		switch syncStatus {
		case argoSyncStatusSynced:
			summary.Synced++
		case "OutOfSync":
			summary.OutOfSync++
		}

		// Parse health status
		healthStatus := w.getHealthStatus(&app)
		switch healthStatus {
		case "Healthy":
			summary.Healthy++
		case "Degraded":
			summary.Degraded++
		case "Progressing":
			summary.Progressing++
		case "Missing":
			summary.Missing++
		case "Suspended":
			summary.Suspended++
		}
	}

	log.V(1).Info("Scanned ArgoCD applications",
		"tenant", tenant.Name,
		"total", summary.TotalApps,
		"synced", summary.Synced,
		"healthy", summary.Healthy)

	return summary, nil
}

// belongsToTenant checks if an ArgoCD application belongs to the tenant
func (w *ArgoWatcher) belongsToTenant(app *unstructured.Unstructured, tenant *platformv1alpha1.Tenant) bool {
	// Check labels
	labels := app.GetLabels()
	if tenantLabel, ok := labels["platform.io/tenant"]; ok {
		if tenantLabel == tenant.Name || tenantLabel == extractTenantName(tenant.Spec.DisplayName) {
			return true
		}
	}

	// Check if app project matches tenant's ArgoProjects
	if len(tenant.Spec.ArgoProjects) > 0 {
		project, found, err := unstructured.NestedString(app.Object, "spec", "project")
		if err == nil && found {
			for _, tenantProject := range tenant.Spec.ArgoProjects {
				if project == tenantProject {
					return true
				}
			}
		}
	}

	// Check if app destination namespace matches tenant namespaces
	destNamespace, found, err := unstructured.NestedString(app.Object, "spec", "destination", "namespace")
	if err == nil && found {
		for _, ns := range tenant.Spec.Namespaces {
			if destNamespace == ns {
				return true
			}
		}
	}

	return false
}

const (
	argoSyncStatusSynced = "Synced"
	argoStatusUnknown    = "Unknown"
)

// getSyncStatus extracts the sync status from an ArgoCD Application
func (w *ArgoWatcher) getSyncStatus(app *unstructured.Unstructured) string {
	status, found, err := unstructured.NestedString(app.Object, "status", "sync", "status")
	if err != nil || !found {
		return argoStatusUnknown
	}
	return status
}

// getHealthStatus extracts the health status from an ArgoCD Application
func (w *ArgoWatcher) getHealthStatus(app *unstructured.Unstructured) string {
	status, found, err := unstructured.NestedString(app.Object, "status", "health", "status")
	if err != nil || !found {
		return argoStatusUnknown
	}
	return status
}

// extractTenantName extracts a simple tenant name from display name
// e.g., "Alpha - ML Platform" -> "alpha"
func extractTenantName(displayName string) string {
	// Simple heuristic: take first word and lowercase it
	for i, r := range displayName {
		if r == ' ' || r == '-' {
			name := displayName[:i]
			return toLower(name)
		}
	}
	return toLower(displayName)
}

// toLower converts string to lowercase (simple ASCII version)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + ('a' - 'A')
		} else {
			result[i] = c
		}
	}
	return string(result)
}
