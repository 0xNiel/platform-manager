/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantPhase represents the current phase of the tenant
type TenantPhase string

const (
	// TenantPhaseActive indicates the tenant is active
	TenantPhaseActive TenantPhase = "Active"
	// TenantPhaseSuspended indicates the tenant is suspended
	TenantPhaseSuspended TenantPhase = "Suspended"
	// TenantPhaseDeleting indicates the tenant is being deleted
	TenantPhaseDeleting TenantPhase = "Deleting"
)

// TenantContact represents a contact for the tenant
type TenantContact struct {
	// Name is the name of the contact
	Name string `json:"name"`
	// Email is the email address of the contact
	Email string `json:"email"`
	// Role is the role of the contact (owner, developer, oncall)
	// +optional
	Role string `json:"role,omitempty"`
}

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	// DisplayName is the human-readable name for this tenant
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	DisplayName string `json:"displayName"`

	// Description provides additional context about the tenant
	// +optional
	Description string `json:"description,omitempty"`

	// Namespaces that belong to this tenant
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// LabelSelector to match resources belonging to this tenant
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// AWSAccounts associated with this tenant (for IAM tracking)
	// +optional
	AWSAccounts []string `json:"awsAccounts,omitempty"`

	// ArgoProjects that belong to this tenant
	// +optional
	ArgoProjects []string `json:"argoProjects,omitempty"`

	// Contacts for this tenant (for alerts/notifications)
	// +optional
	Contacts []TenantContact `json:"contacts,omitempty"`

	// CostCenter for billing/chargeback purposes
	// +optional
	CostCenter string `json:"costCenter,omitempty"`

	// Environment indicates the environment type (dev, staging, prod)
	// +optional
	Environment string `json:"environment,omitempty"`
}

// TenantStatus defines the observed state of Tenant
type TenantStatus struct {
	// Phase represents the current phase (Active, Suspended, Deleting)
	// +optional
	Phase TenantPhase `json:"phase,omitempty"`

	// LastReconciled timestamp
	// +optional
	LastReconciled *metav1.Time `json:"lastReconciled,omitempty"`

	// HealthRef points to the TenantHealth resource
	// +optional
	HealthRef string `json:"healthRef,omitempty"`

	// NamespaceCount is the number of namespaces belonging to this tenant
	// +optional
	NamespaceCount int `json:"namespaceCount,omitempty"`

	// ResourceCount is the total number of resources belonging to this tenant
	// +optional
	ResourceCount int `json:"resourceCount,omitempty"`

	// Conditions represent the latest available observations of the tenant's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Namespaces",type=integer,JSONPath=`.status.namespaceCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Tenant represents a logical tenant in the platform
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TenantList contains a list of Tenant
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}
