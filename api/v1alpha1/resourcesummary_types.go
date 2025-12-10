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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceState represents the normalized state of a resource
type ResourceState string

const (
	// ResourceStateReady indicates the resource is ready and healthy
	ResourceStateReady ResourceState = "Ready"
	// ResourceStateFailed indicates the resource has failed
	ResourceStateFailed ResourceState = "Failed"
	// ResourceStateWaiting indicates the resource is waiting/pending
	ResourceStateWaiting ResourceState = "Waiting"
	// ResourceStateUnknown indicates the resource state cannot be determined
	ResourceStateUnknown ResourceState = "Unknown"
	// ResourceStatePaused indicates the resource is paused
	ResourceStatePaused ResourceState = "Paused"
)

// ResourceCategory represents the category of resource
type ResourceCategory string

const (
	ResourceCategoryCrossplane ResourceCategory = "Crossplane"
	ResourceCategoryKubernetes ResourceCategory = "Kubernetes"
	ResourceCategoryArgoCD     ResourceCategory = "ArgoCD"
	ResourceCategoryIAM        ResourceCategory = "IAM"
)

// ConditionSummary is a condensed view of a condition
type ConditionSummary struct {
	// Type of the condition
	Type string `json:"type"`
	// Status of the condition (True, False, Unknown)
	Status string `json:"status"`
	// Reason for the condition
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message providing details about the condition
	// +optional
	Message string `json:"message,omitempty"`
	// LastTransitionTime is when the condition last changed
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
}

// OwnerRef represents a reference to an owner resource
type OwnerRef struct {
	// APIVersion of the owner
	APIVersion string `json:"apiVersion"`
	// Kind of the owner
	Kind string `json:"kind"`
	// Name of the owner
	Name string `json:"name"`
	// Namespace of the owner (empty for cluster-scoped)
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// UID of the owner
	// +optional
	UID string `json:"uid,omitempty"`
}

// ResourceSummarySpec captures the resource reference
type ResourceSummarySpec struct {
	// Group is the API group of the resource
	// +optional
	Group string `json:"group,omitempty"`
	// Version is the API version of the resource
	Version string `json:"version"`
	// Kind is the kind of the resource
	Kind string `json:"kind"`
	// Namespace is the namespace of the resource (empty for cluster-scoped)
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Name is the name of the resource
	Name string `json:"name"`

	// TenantRef links to the owning tenant
	// +optional
	TenantRef string `json:"tenantRef,omitempty"`

	// Category classifies the resource (Crossplane, Kubernetes, ArgoCD, IAM)
	// +optional
	Category ResourceCategory `json:"category,omitempty"`

	// Provider is the Crossplane provider for this resource (if applicable)
	// +optional
	Provider string `json:"provider,omitempty"`
}

// ResourceSummaryStatus captures normalized health
type ResourceSummaryStatus struct {
	// State is the normalized state (Ready, Failed, Waiting, Unknown, Paused)
	// +optional
	State ResourceState `json:"state,omitempty"`

	// Conditions from the source resource (condensed)
	// +optional
	Conditions []ConditionSummary `json:"conditions,omitempty"`

	// Message explaining current state
	// +optional
	Message string `json:"message,omitempty"`

	// OwnerChain for traversing resource tree (from child to root)
	// +optional
	OwnerChain []OwnerRef `json:"ownerChain,omitempty"`

	// CompositionRef is the composition this resource belongs to (for XRs)
	// +optional
	CompositionRef string `json:"compositionRef,omitempty"`

	// ClaimRef is the claim that created this resource (for XRs)
	// +optional
	ClaimRef string `json:"claimRef,omitempty"`

	// Age is how long the resource has existed
	// +optional
	Age string `json:"age,omitempty"`

	// LastTransition is when state last changed
	// +optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`

	// LastSeen is when resource was last observed
	// +optional
	LastSeen *metav1.Time `json:"lastSeen,omitempty"`

	// SyncStatus for ArgoCD resources
	// +optional
	SyncStatus string `json:"syncStatus,omitempty"`

	// HealthStatus for ArgoCD resources
	// +optional
	HealthStatus string `json:"healthStatus,omitempty"`

	// DriftDetected indicates if drift was detected (for IAM resources)
	// +optional
	DriftDetected bool `json:"driftDetected,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.kind`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.namespace`
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantRef`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ResourceSummary is the Schema for the resourcesummaries API
type ResourceSummary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceSummarySpec   `json:"spec,omitempty"`
	Status ResourceSummaryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ResourceSummaryList contains a list of ResourceSummary
type ResourceSummaryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceSummary `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceSummary{}, &ResourceSummaryList{})
}
