package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourcesHandler handles resource-related API requests
type ResourcesHandler struct {
	Client client.Client
}

// NewResourcesHandler creates a new ResourcesHandler
func NewResourcesHandler(c client.Client) *ResourcesHandler {
	return &ResourcesHandler{Client: c}
}

// ResourceListResponse represents the response for listing resources
type ResourceListResponse struct {
	Total     int                                `json:"total"`
	Resources []platformv1alpha1.ResourceSummary `json:"resources"`
}

// ResourceDetailResponse represents the response for a single resource
type ResourceDetailResponse struct {
	Summary platformv1alpha1.ResourceSummary `json:"summary"`
	YAML    string                           `json:"yaml,omitempty"`
	Events  []EventSummary                   `json:"events,omitempty"`
	Tree    *OwnershipTree                   `json:"tree,omitempty"`
}

// EventSummary represents a Kubernetes event
type EventSummary struct {
	Type      string      `json:"type"`
	Reason    string      `json:"reason"`
	Message   string      `json:"message"`
	Count     int32       `json:"count"`
	FirstSeen metav1.Time `json:"firstSeen"`
	LastSeen  metav1.Time `json:"lastSeen"`
}

// OwnershipTree represents the ownership hierarchy of a resource
type OwnershipTree struct {
	Resource   platformv1alpha1.OwnerRef `json:"resource"`
	Owners     []OwnershipTree           `json:"owners,omitempty"`
	Dependents []OwnershipTree           `json:"dependents,omitempty"`
}

// ListResources handles GET /api/v1/resources
func (h *ResourcesHandler) ListResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	query := r.URL.Query()
	tenantFilter := query.Get("tenant")
	stateFilter := query.Get("state")
	kindFilter := query.Get("kind")
	categoryFilter := query.Get("category")
	providerFilter := query.Get("provider")
	searchFilter := query.Get("search")

	// Build list options
	listOpts := []client.ListOption{}

	// Build label selector
	labelSelector := labels.NewSelector()
	if tenantFilter != "" {
		req, _ := labels.NewRequirement("platform.io/tenant", "=", []string{tenantFilter})
		labelSelector = labelSelector.Add(*req)
	}
	if categoryFilter != "" {
		req, _ := labels.NewRequirement("platform.io/category", "=", []string{categoryFilter})
		labelSelector = labelSelector.Add(*req)
	}
	if !labelSelector.Empty() {
		listOpts = append(listOpts, client.MatchingLabelsSelector{Selector: labelSelector})
	}

	// List all ResourceSummary CRs
	resourceList := &platformv1alpha1.ResourceSummaryList{}
	if err := h.Client.List(ctx, resourceList, listOpts...); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list resources: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter resources based on query parameters
	filteredResources := []platformv1alpha1.ResourceSummary{}
	for _, resource := range resourceList.Items {
		// Filter by state
		if stateFilter != "" && string(resource.Status.State) != stateFilter {
			continue
		}

		// Filter by kind
		if kindFilter != "" && resource.Spec.Kind != kindFilter {
			continue
		}

		// Filter by provider
		if providerFilter != "" && resource.Spec.Provider != providerFilter {
			continue
		}

		// Filter by search (name contains)
		if searchFilter != "" && !strings.Contains(strings.ToLower(resource.Spec.Name), strings.ToLower(searchFilter)) {
			continue
		}

		filteredResources = append(filteredResources, resource)
	}

	// Build response
	response := ResourceListResponse{
		Total:     len(filteredResources),
		Resources: filteredResources,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetResource handles GET /api/v1/resources/{name}
func (h *ResourcesHandler) GetResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract resource name from path
	// Expected path: /api/v1/resources/{name}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/resources/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "Resource name is required", http.StatusBadRequest)
		return
	}
	resourceName := pathParts[0]

	// Get the ResourceSummary
	resourceSummary := &platformv1alpha1.ResourceSummary{}
	if err := h.Client.Get(ctx, types.NamespacedName{Name: resourceName}, resourceSummary); err != nil {
		if errors.IsNotFound(err) {
			http.Error(w, "Resource not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to get resource: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Build response
	response := ResourceDetailResponse{
		Summary: *resourceSummary,
	}

	// Check if YAML is requested
	if r.URL.Query().Get("yaml") == "true" {
		yaml, err := h.getResourceYAML(ctx, resourceSummary)
		if err == nil {
			response.YAML = yaml
		}
	}

	// Check if events are requested
	if r.URL.Query().Get("events") == "true" {
		events, err := h.getResourceEvents(ctx, resourceSummary)
		if err == nil {
			response.Events = events
		}
	}

	// Check if tree is requested
	if r.URL.Query().Get("tree") == "true" {
		tree, err := h.getOwnershipTree(ctx, resourceSummary)
		if err == nil {
			response.Tree = tree
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListTenantResources handles GET /api/v1/tenants/{id}/resources
func (h *ResourcesHandler) ListTenantResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		http.Error(w, "Tenant ID is required", http.StatusBadRequest)
		return
	}
	tenantID := pathParts[0]

	// Parse query parameters
	query := r.URL.Query()
	stateFilter := query.Get("state")
	kindFilter := query.Get("kind")
	categoryFilter := query.Get("category")

	// Build list options with tenant filter
	listOpts := []client.ListOption{
		client.MatchingLabels{"platform.io/tenant": tenantID},
	}
	if categoryFilter != "" {
		listOpts = append(listOpts, client.MatchingLabels{"platform.io/category": categoryFilter})
	}

	// List all ResourceSummary CRs for this tenant
	resourceList := &platformv1alpha1.ResourceSummaryList{}
	if err := h.Client.List(ctx, resourceList, listOpts...); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list tenant resources: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter resources based on query parameters
	filteredResources := []platformv1alpha1.ResourceSummary{}
	for _, resource := range resourceList.Items {
		// Filter by state
		if stateFilter != "" && string(resource.Status.State) != stateFilter {
			continue
		}

		// Filter by kind
		if kindFilter != "" && resource.Spec.Kind != kindFilter {
			continue
		}

		filteredResources = append(filteredResources, resource)
	}

	// Build response
	response := ResourceListResponse{
		Total:     len(filteredResources),
		Resources: filteredResources,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getResourceYAML retrieves the raw YAML for a resource
func (h *ResourcesHandler) getResourceYAML(ctx context.Context, summary *platformv1alpha1.ResourceSummary) (string, error) {
	// Build GVK
	gvk := schema.GroupVersionKind{
		Group:   summary.Spec.Group,
		Version: summary.Spec.Version,
		Kind:    summary.Spec.Kind,
	}

	// Get the resource
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(gvk)

	key := types.NamespacedName{
		Namespace: summary.Spec.Namespace,
		Name:      summary.Spec.Name,
	}

	if err := h.Client.Get(ctx, key, resource); err != nil {
		return "", err
	}

	// Marshal to YAML (simplified - using JSON for now)
	yamlBytes, err := json.MarshalIndent(resource.Object, "", "  ")
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
}

// getResourceEvents retrieves Kubernetes events for a resource
func (h *ResourcesHandler) getResourceEvents(ctx context.Context, summary *platformv1alpha1.ResourceSummary) ([]EventSummary, error) {
	// List events in the resource's namespace
	eventList := &corev1.EventList{}
	listOpts := []client.ListOption{}
	if summary.Spec.Namespace != "" {
		listOpts = append(listOpts, client.InNamespace(summary.Spec.Namespace))
	}

	if err := h.Client.List(ctx, eventList, listOpts...); err != nil {
		return nil, err
	}

	// Filter events related to this resource
	events := []EventSummary{}
	for _, event := range eventList.Items {
		if event.InvolvedObject.Name == summary.Spec.Name &&
			event.InvolvedObject.Kind == summary.Spec.Kind {
			events = append(events, EventSummary{
				Type:      event.Type,
				Reason:    event.Reason,
				Message:   event.Message,
				Count:     event.Count,
				FirstSeen: event.FirstTimestamp,
				LastSeen:  event.LastTimestamp,
			})
		}
	}

	return events, nil
}

// getOwnershipTree builds the ownership tree for a resource
func (h *ResourcesHandler) getOwnershipTree(ctx context.Context, summary *platformv1alpha1.ResourceSummary) (*OwnershipTree, error) {
	tree := &OwnershipTree{
		Resource: platformv1alpha1.OwnerRef{
			APIVersion: fmt.Sprintf("%s/%s", summary.Spec.Group, summary.Spec.Version),
			Kind:       summary.Spec.Kind,
			Name:       summary.Spec.Name,
			Namespace:  summary.Spec.Namespace,
		},
	}

	// Build owners from OwnerChain
	for _, owner := range summary.Status.OwnerChain {
		tree.Owners = append(tree.Owners, OwnershipTree{
			Resource: owner,
		})
	}

	// Find dependents (resources that are owned by this resource)
	dependents, err := h.findDependents(ctx, summary)
	if err == nil {
		tree.Dependents = dependents
	}

	return tree, nil
}

// findDependents finds resources owned by the given resource
func (h *ResourcesHandler) findDependents(ctx context.Context, summary *platformv1alpha1.ResourceSummary) ([]OwnershipTree, error) {
	dependents := []OwnershipTree{}

	// List all ResourceSummary CRs
	resourceList := &platformv1alpha1.ResourceSummaryList{}
	if err := h.Client.List(ctx, resourceList); err != nil {
		return dependents, err
	}

	// Find resources whose owner chain includes this resource
	for _, resource := range resourceList.Items {
		for _, owner := range resource.Status.OwnerChain {
			if owner.Name == summary.Spec.Name &&
				owner.Kind == summary.Spec.Kind &&
				owner.Namespace == summary.Spec.Namespace {
				dependents = append(dependents, OwnershipTree{
					Resource: platformv1alpha1.OwnerRef{
						APIVersion: fmt.Sprintf("%s/%s", resource.Spec.Group, resource.Spec.Version),
						Kind:       resource.Spec.Kind,
						Name:       resource.Spec.Name,
						Namespace:  resource.Spec.Namespace,
					},
				})
				break
			}
		}
	}

	return dependents, nil
}
