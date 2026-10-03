/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderHealthSummary summarizes the health of a Crossplane provider
type ProviderHealthSummary struct {
	// Name is the name of the provider
	Name string `json:"name"`
	// Package is the provider package reference
	Package string `json:"package"`
	// Version is the installed version
	Version string `json:"version,omitempty"`
	// Healthy indicates if the provider is healthy
	Healthy bool `json:"healthy"`
	// InstalledRevision is the current revision
	// +optional
	InstalledRevision string `json:"installedRevision,omitempty"`
	// Message provides details about the provider status
	// +optional
	Message string `json:"message,omitempty"`
}

// TenantHealthSummary summarizes a tenant's health for the global view
type TenantHealthSummary struct {
	// Name is the tenant name
	Name string `json:"name"`
	// DisplayName is the human-readable name
	DisplayName string `json:"displayName"`
	// Health is the overall health level
	Health HealthLevel `json:"health"`
	// FailedResources is the count of failed resources
	FailedResources int `json:"failedResources"`
	// TotalResources is the total resource count
	TotalResources int `json:"totalResources"`
	// IAMDriftCount is the count of resources with IAM drift
	IAMDriftCount int `json:"iamDriftCount"`
	// ArgoOutOfSync is the count of ArgoCD apps out of sync
	ArgoOutOfSync int `json:"argoOutOfSync"`
}

// PlatformHealthSpec defines the desired state of PlatformHealth
type PlatformHealthSpec struct {
	// Name is the name of this platform health instance (usually "default")
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="default"
	Name string `json:"name,omitempty"`
}

// PlatformHealthStatus defines the observed state of PlatformHealth
type PlatformHealthStatus struct {
	// OverallHealth indicates the global platform health level
	// +optional
	OverallHealth HealthLevel `json:"overallHealth,omitempty"`

	// TotalTenants is the total number of tenants
	// +optional
	TotalTenants int `json:"totalTenants,omitempty"`

	// HealthyTenants is the count of healthy tenants
	// +optional
	HealthyTenants int `json:"healthyTenants,omitempty"`

	// DegradedTenants is the count of degraded tenants
	// +optional
	DegradedTenants int `json:"degradedTenants,omitempty"`

	// CriticalTenants is the count of tenants with critical issues
	// +optional
	CriticalTenants int `json:"criticalTenants,omitempty"`

	// CrossplaneResources shows aggregated state counts for all Crossplane resources
	// +optional
	CrossplaneResources ResourceStateCounts `json:"crossplaneResources,omitempty"`

	// KubernetesResources shows aggregated state counts for Kubernetes resources
	// +optional
	KubernetesResources ResourceStateCounts `json:"kubernetesResources,omitempty"`

	// TotalIAMDrift is the total count of resources with IAM drift
	// +optional
	TotalIAMDrift int `json:"totalIamDrift,omitempty"`

	// ArgoSummary shows aggregated ArgoCD application states
	// +optional
	ArgoSummary ArgoSummary `json:"argoSummary,omitempty"`

	// Providers shows the health of Crossplane providers
	// +optional
	Providers []ProviderHealthSummary `json:"providers,omitempty"`

	// TotalCPUUsage is the total CPU usage across all tenants
	// +optional
	TotalCPUUsage resource.Quantity `json:"totalCpuUsage,omitempty"`

	// TotalMemoryUsage is the total memory usage across all tenants
	// +optional
	TotalMemoryUsage resource.Quantity `json:"totalMemoryUsage,omitempty"`

	// TotalCPURequest is the total CPU requested across all tenants
	// +optional
	TotalCPURequest resource.Quantity `json:"totalCpuRequest,omitempty"`

	// TotalMemoryRequest is the total memory requested across all tenants
	// +optional
	TotalMemoryRequest resource.Quantity `json:"totalMemoryRequest,omitempty"`

	// Tenants provides a summary of each tenant's health
	// +optional
	Tenants []TenantHealthSummary `json:"tenants,omitempty"`

	// TopIssues contains the top issues across all tenants (max 10)
	// +optional
	TopIssues []Issue `json:"topIssues,omitempty"`

	// LastUpdated is when this status was last updated
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// ActiveAlerts is the total count of active alerts across all tenants
	// +optional
	ActiveAlerts int `json:"activeAlerts,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Health",type=string,JSONPath=`.status.overallHealth`
// +kubebuilder:printcolumn:name="Tenants",type=integer,JSONPath=`.status.totalTenants`
// +kubebuilder:printcolumn:name="Healthy",type=integer,JSONPath=`.status.healthyTenants`
// +kubebuilder:printcolumn:name="Critical",type=integer,JSONPath=`.status.criticalTenants`
// +kubebuilder:printcolumn:name="IAM Drift",type=integer,JSONPath=`.status.totalIamDrift`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PlatformHealth is the Schema for the platformhealths API (singleton)
type PlatformHealth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlatformHealthSpec   `json:"spec,omitempty"`
	Status PlatformHealthStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PlatformHealthList contains a list of PlatformHealth
type PlatformHealthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlatformHealth `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlatformHealth{}, &PlatformHealthList{})
}
