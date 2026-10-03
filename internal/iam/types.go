/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package iam

import (
	"encoding/json"
	"time"
)

// DriftSeverity represents the severity of drift
type DriftSeverity string

const (
	// DriftSeverityCritical indicates critical security issues (extra privileges)
	DriftSeverityCritical DriftSeverity = "critical"
	// DriftSeverityHigh indicates significant drift (missing privileges)
	DriftSeverityHigh DriftSeverity = "high"
	// DriftSeverityWarning indicates minor drift (tag changes, etc.)
	DriftSeverityWarning DriftSeverity = "warning"
	// DriftSeverityNone indicates no drift
	DriftSeverityNone DriftSeverity = "none"
)

// DriftType represents the type of drift detected
type DriftType string

const (
	// DriftTypeExtraPrivileges indicates privileges in AWS but not in spec
	DriftTypeExtraPrivileges DriftType = "extra_privileges"
	// DriftTypeMissingPrivileges indicates privileges in spec but not in AWS
	DriftTypeMissingPrivileges DriftType = "missing_privileges"
	// DriftTypePolicyMismatch indicates policy content differs
	DriftTypePolicyMismatch DriftType = "policy_mismatch"
	// DriftTypeOrphanedResource indicates resource in AWS but not managed by Crossplane
	DriftTypeOrphanedResource DriftType = "orphaned_resource"
	// DriftTypeReconciliationLag indicates status.atProvider differs from spec.forProvider
	DriftTypeReconciliationLag DriftType = "reconciliation_lag"
)

// ResourceType represents the type of IAM resource
type ResourceType string

const (
	// ResourceTypeRole represents an IAM Role
	ResourceTypeRole ResourceType = "Role"
	// ResourceTypePolicy represents an IAM Policy
	ResourceTypePolicy ResourceType = "Policy"
	// ResourceTypeRolePolicy represents an inline role policy
	ResourceTypeRolePolicy ResourceType = "RolePolicy"
	// ResourceTypePolicyAttachment represents a policy attachment
	ResourceTypePolicyAttachment ResourceType = "PolicyAttachment"
)

// DriftResult represents the result of a drift check for a single resource
type DriftResult struct {
	// ResourceType is the type of IAM resource (Role, Policy, etc.)
	ResourceType ResourceType `json:"resourceType"`
	// ResourceName is the name of the resource in Kubernetes
	ResourceName string `json:"resourceName"`
	// ResourceARN is the AWS ARN of the resource (if available)
	ResourceARN string `json:"resourceArn,omitempty"`
	// TenantName is the tenant this resource belongs to
	TenantName string `json:"tenantName"`
	// Namespace is the Kubernetes namespace
	Namespace string `json:"namespace,omitempty"`

	// HasDrift indicates if drift was detected
	HasDrift bool `json:"hasDrift"`
	// Severity is the drift severity level
	Severity DriftSeverity `json:"severity"`
	// DriftTypes is the list of drift types detected
	DriftTypes []DriftType `json:"driftTypes,omitempty"`

	// Details contains detailed drift information
	Details []DriftDetail `json:"details,omitempty"`

	// DesiredState is the desired state from spec.forProvider
	DesiredState map[string]interface{} `json:"desiredState,omitempty"`
	// ObservedState is the observed state from status.atProvider
	ObservedState map[string]interface{} `json:"observedState,omitempty"`
	// ActualState is the actual state from AWS API
	ActualState map[string]interface{} `json:"actualState,omitempty"`

	// CheckedAt is when the drift check was performed
	CheckedAt time.Time `json:"checkedAt"`
	// Error contains any error encountered during drift check
	Error string `json:"error,omitempty"`
}

// DriftDetail represents a specific drift finding
type DriftDetail struct {
	// Type is the drift type
	Type DriftType `json:"type"`
	// Severity is the severity of this specific drift
	Severity DriftSeverity `json:"severity"`
	// Path is the JSON path where drift was detected (e.g., "Statement[0].Action")
	Path string `json:"path,omitempty"`
	// Message describes the drift in human-readable form
	Message string `json:"message"`
	// Expected is the expected value (from spec)
	Expected interface{} `json:"expected,omitempty"`
	// Actual is the actual value (from AWS)
	Actual interface{} `json:"actual,omitempty"`
	// Diff is a textual diff representation
	Diff string `json:"diff,omitempty"`
}

// TenantDriftSummary aggregates drift for a single tenant
type TenantDriftSummary struct {
	TenantName        string        `json:"tenantName"`
	TotalRoles        int           `json:"totalRoles"`
	TotalPolicies     int           `json:"totalPolicies"`
	RolesWithDrift    int           `json:"rolesWithDrift"`
	PoliciesWithDrift int           `json:"policiesWithDrift"`
	CriticalDrifts    int           `json:"criticalDrifts"`
	HighDrifts        int           `json:"highDrifts"`
	WarningDrifts     int           `json:"warningDrifts"`
	LastChecked       time.Time     `json:"lastChecked"`
	Drifts            []DriftResult `json:"drifts,omitempty"`
}

// PlatformDriftSummary aggregates drift for the entire platform
type PlatformDriftSummary struct {
	TotalTenants      int                            `json:"totalTenants"`
	TotalRoles        int                            `json:"totalRoles"`
	TotalPolicies     int                            `json:"totalPolicies"`
	RolesWithDrift    int                            `json:"rolesWithDrift"`
	PoliciesWithDrift int                            `json:"policiesWithDrift"`
	CriticalDrifts    int                            `json:"criticalDrifts"`
	HighDrifts        int                            `json:"highDrifts"`
	WarningDrifts     int                            `json:"warningDrifts"`
	LastChecked       time.Time                      `json:"lastChecked"`
	TenantSummaries   map[string]*TenantDriftSummary `json:"tenantSummaries"`
}

// PolicyDocument represents an IAM policy document
type PolicyDocument struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

// Statement represents a single statement in an IAM policy
type Statement struct {
	Sid       string      `json:"Sid,omitempty"`
	Effect    string      `json:"Effect"`
	Principal interface{} `json:"Principal,omitempty"`
	Action    interface{} `json:"Action,omitempty"`
	Resource  interface{} `json:"Resource,omitempty"`
	Condition interface{} `json:"Condition,omitempty"`
}

// ParsePolicyDocument parses a policy document JSON string
func ParsePolicyDocument(jsonStr string) (*PolicyDocument, error) {
	var doc PolicyDocument
	if err := json.Unmarshal([]byte(jsonStr), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// NormalizeActions converts Action to []string for comparison
func NormalizeActions(action interface{}) []string {
	if action == nil {
		return []string{}
	}

	switch v := action.(type) {
	case string:
		return []string{v}
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case []string:
		return v
	default:
		return []string{}
	}
}

// NormalizeResources converts Resource to []string for comparison
func NormalizeResources(resource interface{}) []string {
	return NormalizeActions(resource) // Same logic
}
