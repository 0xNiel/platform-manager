// Package rules provides a rule-based troubleshooting engine
// for detecting common issues in the platform.
package rules

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Severity indicates the importance of a finding
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// FindingStatus indicates whether an issue is active or resolved
type FindingStatus string

const (
	FindingStatusActive   FindingStatus = "active"
	FindingStatusResolved FindingStatus = "resolved"
)

// Finding represents a detected issue
type Finding struct {
	// ID is a unique identifier for this finding
	ID string `json:"id"`

	// RuleID is the ID of the rule that generated this finding
	RuleID string `json:"ruleId"`

	// RuleName is the human-readable name of the rule
	RuleName string `json:"ruleName"`

	// Severity of the issue
	Severity Severity `json:"severity"`

	// Status of the finding
	Status FindingStatus `json:"status"`

	// Title is a short description of the issue
	Title string `json:"title"`

	// Message provides detailed explanation
	Message string `json:"message"`

	// Recommendation suggests how to fix the issue
	Recommendation string `json:"recommendation,omitempty"`

	// ResourceRef identifies the affected resource
	ResourceRef ResourceReference `json:"resourceRef"`

	// TenantName is the tenant this finding belongs to
	TenantName string `json:"tenantName,omitempty"`

	// FirstSeen when this issue was first detected
	FirstSeen metav1.Time `json:"firstSeen"`

	// LastSeen when this issue was last observed
	LastSeen metav1.Time `json:"lastSeen"`

	// Occurrences count how many times this issue was detected
	Occurrences int `json:"occurrences"`

	// Metadata for additional context
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ResourceReference identifies a Kubernetes resource
type ResourceReference struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	UID       string `json:"uid,omitempty"`
}

// String returns a human-readable resource reference
func (r ResourceReference) String() string {
	if r.Namespace != "" {
		return r.Kind + "/" + r.Namespace + "/" + r.Name
	}
	return r.Kind + "/" + r.Name
}

// RuleContext provides context for rule evaluation
type RuleContext struct {
	// Ctx is the context for the evaluation
	Ctx context.Context

	// Resource is the resource being evaluated
	Resource *unstructured.Unstructured

	// TenantName is the tenant this resource belongs to
	TenantName string

	// RelatedResources are resources related to the main resource
	RelatedResources map[string]*unstructured.Unstructured

	// Now is the current time (for testing)
	Now time.Time
}

// Rule is the interface that all rules must implement
type Rule interface {
	// ID returns a unique identifier for this rule
	ID() string

	// Name returns a human-readable name
	Name() string

	// Description provides details about what this rule checks
	Description() string

	// Severity returns the severity level for findings from this rule
	Severity() Severity

	// AppliesTo checks if this rule should be evaluated for the given resource
	AppliesTo(resource *unstructured.Unstructured) bool

	// Evaluate checks the resource and returns findings
	Evaluate(ctx RuleContext) ([]Finding, error)
}

// EvaluationResult holds results from evaluating all rules
type EvaluationResult struct {
	// Findings is the list of all detected issues
	Findings []Finding `json:"findings"`

	// RulesEvaluated is the number of rules that were run
	RulesEvaluated int `json:"rulesEvaluated"`

	// ResourcesChecked is the number of resources examined
	ResourcesChecked int `json:"resourcesChecked"`

	// Duration is how long the evaluation took
	Duration time.Duration `json:"duration"`

	// Timestamp when the evaluation completed
	Timestamp metav1.Time `json:"timestamp"`

	// Errors encountered during evaluation
	Errors []string `json:"errors,omitempty"`
}

// SummaryByTenant provides per-tenant summary of findings
type SummaryByTenant struct {
	TenantName string `json:"tenantName"`
	Total      int    `json:"total"`
	Critical   int    `json:"critical"`
	High       int    `json:"high"`
	Medium     int    `json:"medium"`
	Low        int    `json:"low"`
	Info       int    `json:"info"`
}

// PlatformSummary provides platform-wide summary
type PlatformSummary struct {
	TotalFindings  int               `json:"totalFindings"`
	Critical       int               `json:"critical"`
	High           int               `json:"high"`
	Medium         int               `json:"medium"`
	Low            int               `json:"low"`
	Info           int               `json:"info"`
	ByTenant       []SummaryByTenant `json:"byTenant"`
	TopFindings    []Finding         `json:"topFindings"`
	LastEvaluation metav1.Time       `json:"lastEvaluation"`
}
