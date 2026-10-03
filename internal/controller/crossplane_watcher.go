/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
)

// CrossplaneWatcher scans Crossplane resources and updates TenantHealth
type CrossplaneWatcher struct {
	client client.Client
}

// NewCrossplaneWatcher creates a new CrossplaneWatcher
func NewCrossplaneWatcher(c client.Client) *CrossplaneWatcher {
	return &CrossplaneWatcher{
		client: c,
	}
}

// CrossplaneResourceState represents the normalized state of a Crossplane resource
type CrossplaneResourceState string

const (
	CrossplaneStateReady   CrossplaneResourceState = "Ready"
	CrossplaneStateFailed  CrossplaneResourceState = "Failed"
	CrossplaneStateWaiting CrossplaneResourceState = "Waiting"
	CrossplaneStateUnknown CrossplaneResourceState = "Unknown"
	CrossplaneStatePaused  CrossplaneResourceState = "Paused"
)

// Crossplane resource GVKs to watch
var crossplaneGVKs = []schema.GroupVersionKind{
	// Crossplane AWS IAM resources
	{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "Role"},
	{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "Policy"},
	{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "RolePolicyAttachment"},
	{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "User"},
	{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "Group"},
	// Add more as needed
}

// ScanCrossplaneResources scans all Crossplane resources for a tenant and returns counts
func (w *CrossplaneWatcher) ScanCrossplaneResources(ctx context.Context, tenant *platformv1alpha1.Tenant) (platformv1alpha1.ResourceStateCounts, error) {
	log := logf.FromContext(ctx).WithName("crossplane-watcher")

	counts := platformv1alpha1.ResourceStateCounts{}

	// Scan each type of Crossplane resource
	for _, gvk := range crossplaneGVKs {
		resources, err := w.listResourcesByGVK(ctx, gvk, tenant)
		if err != nil {
			// Log but don't fail - CRD might not be installed
			log.V(1).Info("Failed to list resources", "gvk", gvk.String(), "error", err)
			continue
		}

		// Process each resource
		for _, resource := range resources {
			state := w.normalizeResourceState(&resource)

			switch state {
			case CrossplaneStateReady:
				counts.Ready++
			case CrossplaneStateFailed:
				counts.Failed++
			case CrossplaneStateWaiting:
				counts.Waiting++
			case CrossplaneStatePaused:
				counts.Paused++
			default:
				counts.Unknown++
			}
			counts.Total++
		}
	}

	log.V(1).Info("Scanned Crossplane resources",
		"tenant", tenant.Name,
		"total", counts.Total,
		"ready", counts.Ready,
		"failed", counts.Failed)

	return counts, nil
}

// listResourcesByGVK lists resources of a specific GVK that belong to the tenant
func (w *CrossplaneWatcher) listResourcesByGVK(ctx context.Context, gvk schema.GroupVersionKind, tenant *platformv1alpha1.Tenant) ([]unstructured.Unstructured, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	})

	// List all resources of this type
	if err := w.client.List(ctx, list); err != nil {
		return nil, err
	}

	// Filter by tenant labels if labelSelector is defined
	var filtered []unstructured.Unstructured
	if tenant.Spec.LabelSelector != nil {
		selector := labels.SelectorFromSet(labels.Set(tenant.Spec.LabelSelector.MatchLabels))

		for _, item := range list.Items {
			if selector.Matches(labels.Set(item.GetLabels())) {
				filtered = append(filtered, item)
			}
		}
	} else {
		// If no label selector, match by namespace (if resources are namespaced)
		for _, item := range list.Items {
			// Check if resource is in one of tenant's namespaces
			for _, ns := range tenant.Spec.Namespaces {
				if item.GetNamespace() == ns {
					filtered = append(filtered, item)
					break
				}
			}
			// Also check for cluster-scoped resources with tenant label
			if item.GetNamespace() == "" {
				if tenantLabel, ok := item.GetLabels()["platform.io/tenant"]; ok {
					if tenantLabel == tenant.Name || tenantLabel == tenant.Spec.DisplayName {
						filtered = append(filtered, item)
					}
				}
			}
		}
	}

	return filtered, nil
}

// normalizeResourceState determines the state of a Crossplane resource
func (w *CrossplaneWatcher) normalizeResourceState(resource *unstructured.Unstructured) CrossplaneResourceState {
	// Check if resource is paused
	annotations := resource.GetAnnotations()
	if paused, ok := annotations["crossplane.io/paused"]; ok && paused == "true" {
		return CrossplaneStatePaused
	}

	// Get status conditions
	conditions, found, err := unstructured.NestedSlice(resource.Object, "status", "conditions")
	if err != nil || !found || len(conditions) == 0 {
		return CrossplaneStateUnknown
	}

	// Look for Ready condition (standard Crossplane pattern)
	for _, c := range conditions {
		condition, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		condType, ok := condition["type"].(string)
		if !ok || condType != "Ready" {
			continue
		}

		status, ok := condition["status"].(string)
		if !ok {
			continue
		}

		reason, _ := condition["reason"].(string)

		switch status {
		case "True":
			return CrossplaneStateReady
		case "False":
			// Check reason to distinguish between failed and waiting
			if reason == "ReconcileError" || reason == "CreateFailed" || reason == "UpdateFailed" {
				return CrossplaneStateFailed
			}
			return CrossplaneStateWaiting
		case "Unknown":
			return CrossplaneStateUnknown
		}
	}

	// If no Ready condition found, check for Synced condition
	for _, c := range conditions {
		condition, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		condType, ok := condition["type"].(string)
		if !ok || condType != "Synced" {
			continue
		}

		status, ok := condition["status"].(string)
		if !ok {
			continue
		}

		if status == "True" {
			// Synced but not ready means waiting
			return CrossplaneStateWaiting
		}
	}

	return CrossplaneStateUnknown
}

// ScanIAMResources scans IAM-specific resources and returns drift summary
func (w *CrossplaneWatcher) ScanIAMResources(ctx context.Context, tenant *platformv1alpha1.Tenant) (platformv1alpha1.IAMDriftSummary, error) {
	log := logf.FromContext(ctx).WithName("crossplane-watcher")

	summary := platformv1alpha1.IAMDriftSummary{}

	// IAM Role GVK
	roleGVK := schema.GroupVersionKind{
		Group:   "iam.aws.upbound.io",
		Version: "v1beta1",
		Kind:    "Role",
	}

	roles, err := w.listResourcesByGVK(ctx, roleGVK, tenant)
	if err != nil {
		log.V(1).Info("Failed to list IAM roles", "error", err)
	} else {
		summary.TotalRoles = len(roles)
		// TODO: Implement actual drift detection in Phase 4
		// For now, just count roles
	}

	// IAM Policy GVK
	policyGVK := schema.GroupVersionKind{
		Group:   "iam.aws.upbound.io",
		Version: "v1beta1",
		Kind:    "Policy",
	}

	policies, err := w.listResourcesByGVK(ctx, policyGVK, tenant)
	if err != nil {
		log.V(1).Info("Failed to list IAM policies", "error", err)
	} else {
		summary.TotalPolicies = len(policies)
		// TODO: Implement actual drift detection in Phase 4
	}

	log.V(1).Info("Scanned IAM resources",
		"tenant", tenant.Name,
		"roles", summary.TotalRoles,
		"policies", summary.TotalPolicies)

	return summary, nil
}
