package rules

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Engine evaluates rules against resources and manages findings
type Engine struct {
	rules   []Rule
	mu      sync.RWMutex
	cache   map[string]Finding // Cache of active findings by ID
	history []EvaluationResult // Recent evaluation results
}

// NewEngine creates a new rule engine
func NewEngine() *Engine {
	return &Engine{
		rules:   make([]Rule, 0),
		cache:   make(map[string]Finding),
		history: make([]EvaluationResult, 0, 100), // Keep last 100 results
	}
}

// RegisterRule adds a rule to the engine
func (e *Engine) RegisterRule(rule Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
}

// RegisterRules adds multiple rules to the engine
func (e *Engine) RegisterRules(rules []Rule) {
	for _, rule := range rules {
		e.RegisterRule(rule)
	}
}

// GetRules returns all registered rules
func (e *Engine) GetRules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]Rule, len(e.rules))
	copy(result, e.rules)
	return result
}

// EvaluateResource runs all applicable rules against a single resource
func (e *Engine) EvaluateResource(ctx context.Context, resource *unstructured.Unstructured, tenantName string) ([]Finding, error) {
	logger := log.FromContext(ctx)

	e.mu.RLock()
	rules := e.rules
	e.mu.RUnlock()

	var allFindings []Finding
	ruleCtx := RuleContext{
		Ctx:              ctx,
		Resource:         resource,
		TenantName:       tenantName,
		RelatedResources: make(map[string]*unstructured.Unstructured),
		Now:              time.Now(),
	}

	for _, rule := range rules {
		if !rule.AppliesTo(resource) {
			continue
		}

		findings, err := rule.Evaluate(ruleCtx)
		if err != nil {
			logger.Error(err, "rule evaluation failed",
				"rule", rule.ID(),
				"resource", fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()),
			)
			continue
		}

		allFindings = append(allFindings, findings...)
	}

	// Update cache
	e.mu.Lock()
	defer e.mu.Unlock()

	now := metav1.Now()
	for i := range allFindings {
		finding := &allFindings[i]

		// Generate stable finding ID
		finding.ID = generateFindingID(finding.RuleID, &finding.ResourceRef)
		finding.TenantName = tenantName

		// Check if this is a known finding
		if existing, found := e.cache[finding.ID]; found {
			// Update existing finding
			finding.FirstSeen = existing.FirstSeen
			finding.Occurrences = existing.Occurrences + 1
			finding.LastSeen = now
		} else {
			// New finding
			finding.FirstSeen = now
			finding.LastSeen = now
			finding.Occurrences = 1
			finding.Status = FindingStatusActive
		}

		e.cache[finding.ID] = *finding
	}

	return allFindings, nil
}

// EvaluateResources runs rules against multiple resources
func (e *Engine) EvaluateResources(ctx context.Context, resources []*unstructured.Unstructured, tenantMap map[string]string) EvaluationResult {
	startTime := time.Now()

	result := EvaluationResult{
		Findings:         make([]Finding, 0),
		RulesEvaluated:   len(e.GetRules()),
		ResourcesChecked: len(resources),
		Timestamp:        metav1.Now(),
		Errors:           make([]string, 0),
	}

	for _, resource := range resources {
		tenantName := tenantMap[string(resource.GetUID())]
		if tenantName == "" {
			// Try to extract from labels
			labels := resource.GetLabels()
			if labels != nil {
				tenantName = labels["platform.io/tenant"]
			}
		}

		findings, err := e.EvaluateResource(ctx, resource, tenantName)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", resource.GetName(), err))
			continue
		}

		result.Findings = append(result.Findings, findings...)
	}

	result.Duration = time.Since(startTime)

	// Add to history
	e.mu.Lock()
	e.history = append(e.history, result)
	if len(e.history) > 100 {
		e.history = e.history[1:]
	}
	e.mu.Unlock()

	return result
}

// GetFindings returns all active findings, optionally filtered by tenant
func (e *Engine) GetFindings(tenantName string) []Finding {
	e.mu.RLock()
	defer e.mu.RUnlock()

	findings := make([]Finding, 0, len(e.cache))
	for _, finding := range e.cache {
		if finding.Status != FindingStatusActive {
			continue
		}
		if tenantName != "" && finding.TenantName != tenantName {
			continue
		}
		findings = append(findings, finding)
	}

	// Sort by severity (critical first) then by first seen
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return severityWeight(findings[i].Severity) > severityWeight(findings[j].Severity)
		}
		return findings[i].FirstSeen.Before(&findings[j].FirstSeen)
	})

	return findings
}

// GetPlatformSummary returns a summary of all findings
func (e *Engine) GetPlatformSummary() PlatformSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()

	summary := PlatformSummary{
		ByTenant:    make([]SummaryByTenant, 0),
		TopFindings: make([]Finding, 0),
	}

	// Count findings by severity
	tenantCounts := make(map[string]*SummaryByTenant)

	for _, finding := range e.cache {
		if finding.Status != FindingStatusActive {
			continue
		}

		summary.TotalFindings++
		switch finding.Severity {
		case SeverityCritical:
			summary.Critical++
		case SeverityHigh:
			summary.High++
		case SeverityMedium:
			summary.Medium++
		case SeverityLow:
			summary.Low++
		case SeverityInfo:
			summary.Info++
		}

		// Update tenant counts
		if finding.TenantName != "" {
			if _, exists := tenantCounts[finding.TenantName]; !exists {
				tenantCounts[finding.TenantName] = &SummaryByTenant{
					TenantName: finding.TenantName,
				}
			}
			tc := tenantCounts[finding.TenantName]
			tc.Total++
			switch finding.Severity {
			case SeverityCritical:
				tc.Critical++
			case SeverityHigh:
				tc.High++
			case SeverityMedium:
				tc.Medium++
			case SeverityLow:
				tc.Low++
			case SeverityInfo:
				tc.Info++
			}
		}
	}

	// Convert tenant counts to slice
	for _, tc := range tenantCounts {
		summary.ByTenant = append(summary.ByTenant, *tc)
	}

	// Sort tenants by total findings
	sort.Slice(summary.ByTenant, func(i, j int) bool {
		return summary.ByTenant[i].Total > summary.ByTenant[j].Total
	})

	// Get top findings (highest severity, most recent)
	allFindings := e.GetFindings("")
	maxTop := 10
	if len(allFindings) < maxTop {
		maxTop = len(allFindings)
	}
	summary.TopFindings = allFindings[:maxTop]

	// Set last evaluation time
	if len(e.history) > 0 {
		summary.LastEvaluation = e.history[len(e.history)-1].Timestamp
	}

	return summary
}

// GetTenantSummary returns summary for a specific tenant
func (e *Engine) GetTenantSummary(tenantName string) SummaryByTenant {
	findings := e.GetFindings(tenantName)

	summary := SummaryByTenant{
		TenantName: tenantName,
		Total:      len(findings),
	}

	for _, finding := range findings {
		switch finding.Severity {
		case SeverityCritical:
			summary.Critical++
		case SeverityHigh:
			summary.High++
		case SeverityMedium:
			summary.Medium++
		case SeverityLow:
			summary.Low++
		case SeverityInfo:
			summary.Info++
		}
	}

	return summary
}

// ClearResolved removes findings that are no longer active
func (e *Engine) ClearResolved(maxAge time.Duration) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleared := 0

	for id, finding := range e.cache {
		if finding.Status == FindingStatusResolved && finding.LastSeen.Time.Before(cutoff) {
			delete(e.cache, id)
			cleared++
		}
	}

	return cleared
}

// MarkResolved marks a finding as resolved
func (e *Engine) MarkResolved(findingID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if finding, found := e.cache[findingID]; found {
		finding.Status = FindingStatusResolved
		finding.LastSeen = metav1.Now()
		e.cache[findingID] = finding
		return true
	}

	return false
}

// GetLastEvaluation returns the most recent evaluation result
func (e *Engine) GetLastEvaluation() *EvaluationResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.history) == 0 {
		return nil
	}

	return &e.history[len(e.history)-1]
}

// Helper functions

func generateFindingID(ruleID string, ref *ResourceReference) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%s", ruleID, ref.Group, ref.Kind, ref.Namespace, ref.Name)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8])
}

func severityWeight(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}
