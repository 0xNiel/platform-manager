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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HealthLevel represents the overall health level
type HealthLevel string

const (
	// HealthLevelHealthy indicates everything is working correctly
	HealthLevelHealthy HealthLevel = "Healthy"
	// HealthLevelDegraded indicates some resources have issues
	HealthLevelDegraded HealthLevel = "Degraded"
	// HealthLevelCritical indicates critical failures
	HealthLevelCritical HealthLevel = "Critical"
	// HealthLevelUnknown indicates health cannot be determined
	HealthLevelUnknown HealthLevel = "Unknown"
)

// ResourceStateCounts holds counts for each resource state
type ResourceStateCounts struct {
	// Ready is the count of resources in Ready state
	Ready int `json:"ready"`
	// Failed is the count of resources in Failed state
	Failed int `json:"failed"`
	// Waiting is the count of resources in Waiting/Pending state
	Waiting int `json:"waiting"`
	// Unknown is the count of resources in Unknown state
	Unknown int `json:"unknown"`
	// Paused is the count of resources that are paused
	Paused int `json:"paused"`
	// Total is the total count of resources
	Total int `json:"total"`
}

// IAMDriftSummary summarizes IAM drift for the tenant
type IAMDriftSummary struct {
	// TotalRoles is the total number of IAM roles managed by Crossplane
	TotalRoles int `json:"totalRoles"`
	// TotalPolicies is the total number of IAM policies managed by Crossplane
	TotalPolicies int `json:"totalPolicies"`
	// RolesWithDrift is the number of roles with detected drift
	RolesWithDrift int `json:"rolesWithDrift"`
	// PoliciesWithDrift is the number of policies with detected drift
	PoliciesWithDrift int `json:"policiesWithDrift"`
	// ExtraPrivileges is the count of extra privileges found in AWS vs spec
	ExtraPrivileges int `json:"extraPrivileges"`
	// MissingPrivileges is the count of missing privileges in AWS vs spec
	MissingPrivileges int `json:"missingPrivileges"`
	// LastChecked is when drift was last checked
	// +optional
	LastChecked *metav1.Time `json:"lastChecked,omitempty"`
}

// ArgoSummary summarizes ArgoCD application states for the tenant
type ArgoSummary struct {
	// TotalApps is the total number of ArgoCD applications
	TotalApps int `json:"totalApps"`
	// Synced is the count of applications in Synced state
	Synced int `json:"synced"`
	// OutOfSync is the count of applications that are out of sync
	OutOfSync int `json:"outOfSync"`
	// Healthy is the count of healthy applications
	Healthy int `json:"healthy"`
	// Degraded is the count of degraded applications
	Degraded int `json:"degraded"`
	// Progressing is the count of applications currently syncing
	Progressing int `json:"progressing"`
	// Missing is the count of applications with missing resources
	Missing int `json:"missing"`
	// Suspended is the count of suspended applications
	Suspended int `json:"suspended"`
}

// IssueSeverity represents the severity of an issue
type IssueSeverity string

const (
	IssueSeverityCritical IssueSeverity = "critical"
	IssueSeverityHigh     IssueSeverity = "high"
	IssueSeverityWarning  IssueSeverity = "warning"
	IssueSeverityInfo     IssueSeverity = "info"
)

// Issue represents a detected issue for the tenant
type Issue struct {
	// Severity of the issue (critical, warning, info)
	Severity IssueSeverity `json:"severity"`
	// Message describing the issue
	Message string `json:"message"`
	// RuleID is the identifier of the rule that detected this issue
	// +optional
	RuleID string `json:"ruleId,omitempty"`
	// ResourceRef is a reference to the affected resource
	// +optional
	ResourceRef string `json:"resourceRef,omitempty"`
	// Timestamp when the issue was detected
	Timestamp metav1.Time `json:"timestamp"`
}

// TenantHealthSpec defines the desired state of TenantHealth
type TenantHealthSpec struct {
	// TenantRef is the name of the Tenant this health summary is for
	// +kubebuilder:validation:MinLength=1
	TenantRef string `json:"tenantRef"`
}

// TenantHealthStatus defines the observed state of TenantHealth
type TenantHealthStatus struct {
	// OverallHealth indicates the overall health level (Healthy, Degraded, Critical, Unknown)
	// +optional
	OverallHealth HealthLevel `json:"overallHealth,omitempty"`

	// CrossplaneResources shows state counts for Crossplane-managed resources
	// +optional
	CrossplaneResources ResourceStateCounts `json:"crossplaneResources,omitempty"`

	// KubernetesResources shows state counts for Kubernetes resources (Deployments, Pods, etc.)
	// +optional
	KubernetesResources ResourceStateCounts `json:"kubernetesResources,omitempty"`

	// IAMDrift summarizes IAM drift detection results
	// +optional
	IAMDrift IAMDriftSummary `json:"iamDrift,omitempty"`

	// Argo summarizes ArgoCD application states
	// +optional
	Argo ArgoSummary `json:"argo,omitempty"`

	// CPUUsage is the current CPU usage across all tenant namespaces
	// +optional
	CPUUsage resource.Quantity `json:"cpuUsage,omitempty"`

	// MemoryUsage is the current memory usage across all tenant namespaces
	// +optional
	MemoryUsage resource.Quantity `json:"memoryUsage,omitempty"`

	// CPURequest is the total CPU requested across all tenant namespaces
	// +optional
	CPURequest resource.Quantity `json:"cpuRequest,omitempty"`

	// MemoryRequest is the total memory requested across all tenant namespaces
	// +optional
	MemoryRequest resource.Quantity `json:"memoryRequest,omitempty"`

	// LastUpdated is when this health status was last updated
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// TopIssues contains the top issues for this tenant (max 10)
	// +optional
	TopIssues []Issue `json:"topIssues,omitempty"`

	// ActiveAlerts is the count of currently active alerts
	// +optional
	ActiveAlerts int `json:"activeAlerts,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantRef`
// +kubebuilder:printcolumn:name="Health",type=string,JSONPath=`.status.overallHealth`
// +kubebuilder:printcolumn:name="Failed",type=integer,JSONPath=`.status.crossplaneResources.failed`
// +kubebuilder:printcolumn:name="IAM Drift",type=integer,JSONPath=`.status.iamDrift.rolesWithDrift`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TenantHealth is the Schema for the tenanthealth API
type TenantHealth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantHealthSpec   `json:"spec,omitempty"`
	Status TenantHealthStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TenantHealthList contains a list of TenantHealth
type TenantHealthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantHealth `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TenantHealth{}, &TenantHealthList{})
}

// Now returns the current time as metav1.Time
func Now() metav1.Time {
	return metav1.Now()
}
