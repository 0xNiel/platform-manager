package builtin

import (
	"fmt"
	"strings"

	"github.com/0xNiel/platform-manager/internal/rules"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ArgoSyncFailedRule detects ArgoCD applications stuck in failed sync state
type ArgoSyncFailedRule struct{}

func (r *ArgoSyncFailedRule) ID() string {
	return "argo-sync-failed"
}

func (r *ArgoSyncFailedRule) Name() string {
	return "ArgoCD Sync Failed"
}

func (r *ArgoSyncFailedRule) Description() string {
	return "Detects ArgoCD applications that have been in failed sync state for more than 30 minutes"
}

func (r *ArgoSyncFailedRule) Severity() rules.Severity {
	return rules.SeverityHigh
}

func (r *ArgoSyncFailedRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return resource.GetKind() == "Application" &&
		strings.Contains(resource.GetAPIVersion(), "argoproj.io")
}

func (r *ArgoSyncFailedRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	// Get sync status
	syncStatus, found, err := unstructured.NestedString(resource.Object, "status", "sync", "status")
	if err != nil || !found {
		return nil, nil
	}

	// Check if OutOfSync or Unknown
	if syncStatus != "OutOfSync" && syncStatus != "Unknown" {
		return nil, nil
	}

	// Get operation state
	operationPhase, found, _ := unstructured.NestedString(resource.Object, "status", "operationState", "phase")
	if !found {
		return nil, nil
	}

	// Check if last operation failed
	if operationPhase != "Failed" && operationPhase != "Error" {
		return nil, nil
	}

	// Check how long it's been failing
	operationFinishedAt, found, _ := unstructured.NestedString(resource.Object, "status", "operationState", "finishedAt")
	if !found {
		// Use resource age as fallback
		ageMinutes := rules.GetAgeInMinutes(resource.GetCreationTimestamp(), ctx.Now)
		if ageMinutes < 30 {
			return nil, nil
		}
	}

	// Get error message
	operationMessage, _, _ := unstructured.NestedString(resource.Object, "status", "operationState", "message")

	finding := rules.CreateFinding(
		r,
		resource,
		fmt.Sprintf("ArgoCD application %s has failed sync", resource.GetName()),
		fmt.Sprintf("Application sync has failed. Status: %s, Phase: %s. Message: %s",
			syncStatus, operationPhase, operationMessage),
		"Check application sync status: argocd app get "+resource.GetName()+" -n "+resource.GetNamespace(),
	)
	finding.Metadata["syncStatus"] = syncStatus
	finding.Metadata["operationPhase"] = operationPhase
	if operationFinishedAt != "" {
		finding.Metadata["operationFinishedAt"] = operationFinishedAt
	}

	return []rules.Finding{finding}, nil
}

// ArgoOutOfSyncRule detects ArgoCD applications that are out of sync for extended periods
type ArgoOutOfSyncRule struct{}

func (r *ArgoOutOfSyncRule) ID() string {
	return "argo-out-of-sync-long"
}

func (r *ArgoOutOfSyncRule) Name() string {
	return "ArgoCD Out of Sync"
}

func (r *ArgoOutOfSyncRule) Description() string {
	return "Detects ArgoCD applications that have been out of sync for more than 1 hour"
}

func (r *ArgoOutOfSyncRule) Severity() rules.Severity {
	return rules.SeverityMedium
}

func (r *ArgoOutOfSyncRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return resource.GetKind() == "Application" &&
		strings.Contains(resource.GetAPIVersion(), "argoproj.io")
}

func (r *ArgoOutOfSyncRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	// Get sync status
	syncStatus, found, err := unstructured.NestedString(resource.Object, "status", "sync", "status")
	if err != nil || !found || syncStatus != "OutOfSync" {
		return nil, nil
	}

	// Check if auto-sync is disabled
	autoSyncEnabled := false
	if automated, found, _ := unstructured.NestedMap(resource.Object, "spec", "syncPolicy", "automated"); found && automated != nil {
		autoSyncEnabled = true
	}

	// If auto-sync is enabled, the app should sync automatically
	// If disabled, being out of sync for a while is expected
	if !autoSyncEnabled {
		// Only alert if it's been a very long time
		ageHours := rules.GetAgeInHours(resource.GetCreationTimestamp(), ctx.Now)
		if ageHours < 24 {
			return nil, nil
		}
	}

	// Check age - must be out of sync for > 1 hour
	ageMinutes := rules.GetAgeInMinutes(resource.GetCreationTimestamp(), ctx.Now)
	if ageMinutes < 60 {
		return nil, nil
	}

	// Get revision info
	targetRevision, _, _ := unstructured.NestedString(resource.Object, "spec", "source", "targetRevision")
	currentRevision, _, _ := unstructured.NestedString(resource.Object, "status", "sync", "revision")

	message := fmt.Sprintf("Application has been out of sync for %.0f minutes. ", ageMinutes)
	if autoSyncEnabled {
		message += "Auto-sync is enabled but application is not syncing. "
	} else {
		message += "Auto-sync is disabled. Manual sync may be required. "
	}

	if targetRevision != "" && currentRevision != "" {
		message += fmt.Sprintf("Target: %s, Current: %s", targetRevision, currentRevision)
	}

	finding := rules.CreateFinding(
		r,
		resource,
		fmt.Sprintf("ArgoCD application %s is out of sync", resource.GetName()),
		message,
		"Sync the application manually: argocd app sync "+resource.GetName()+" -n "+resource.GetNamespace(),
	)
	finding.Metadata["syncStatus"] = syncStatus
	finding.Metadata["autoSyncEnabled"] = fmt.Sprintf("%t", autoSyncEnabled)
	finding.Metadata["ageMinutes"] = fmt.Sprintf("%.0f", ageMinutes)

	return []rules.Finding{finding}, nil
}

// HighResourceUsageRule detects namespaces approaching resource quota limits
type HighResourceUsageRule struct{}

func (r *HighResourceUsageRule) ID() string {
	return "high-resource-usage"
}

func (r *HighResourceUsageRule) Name() string {
	return "High Resource Usage"
}

func (r *HighResourceUsageRule) Description() string {
	return "Detects namespaces using more than 80% of their resource quota"
}

func (r *HighResourceUsageRule) Severity() rules.Severity {
	return rules.SeverityMedium
}

func (r *HighResourceUsageRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return resource.GetKind() == "ResourceQuota"
}

func (r *HighResourceUsageRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	// Get hard and used resources
	hard, foundHard, err := unstructured.NestedStringMap(resource.Object, "status", "hard")
	if err != nil || !foundHard {
		return nil, nil
	}

	used, foundUsed, err := unstructured.NestedStringMap(resource.Object, "status", "used")
	if err != nil || !foundUsed {
		return nil, nil
	}

	var findings []rules.Finding

	// Check each resource type
	for resourceType, hardStr := range hard {
		usedStr, exists := used[resourceType]
		if !exists {
			continue
		}

		// Parse quantities (simplified - in production use resource.Quantity)
		hardVal := parseSimpleQuantity(hardStr)
		usedVal := parseSimpleQuantity(usedStr)

		if hardVal == 0 {
			continue
		}

		percentage := (float64(usedVal) / float64(hardVal)) * 100

		if percentage > 80 {
			finding := rules.CreateFinding(
				r,
				resource,
				fmt.Sprintf("High %s usage in namespace %s", resourceType, resource.GetNamespace()),
				fmt.Sprintf("Namespace is using %.1f%% (%s of %s) of %s quota",
					percentage, usedStr, hardStr, resourceType),
				"Consider increasing the resource quota or scaling down workloads in this namespace.",
			)
			finding.Metadata["resourceType"] = resourceType
			finding.Metadata["used"] = usedStr
			finding.Metadata["hard"] = hardStr
			finding.Metadata["percentage"] = fmt.Sprintf("%.1f", percentage)

			findings = append(findings, finding)
		}
	}

	return findings, nil
}

// IAMExtraPrivilegesRule detects IAM roles with extra privileges
// This integrates with the existing IAM drift detection
type IAMExtraPrivilegesRule struct{}

func (r *IAMExtraPrivilegesRule) ID() string {
	return "iam-extra-privileges"
}

func (r *IAMExtraPrivilegesRule) Name() string {
	return "IAM Extra Privileges"
}

func (r *IAMExtraPrivilegesRule) Description() string {
	return "Detects IAM roles that have extra privileges beyond what's defined in Crossplane"
}

func (r *IAMExtraPrivilegesRule) Severity() rules.Severity {
	return rules.SeverityCritical
}

func (r *IAMExtraPrivilegesRule) AppliesTo(resource *unstructured.Unstructured) bool {
	// This rule applies to IAM Role resources from AWS provider
	return resource.GetKind() == "Role" &&
		strings.Contains(resource.GetAPIVersion(), "iam.aws.upbound.io")
}

func (r *IAMExtraPrivilegesRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	// Check for drift annotation (set by IAM drift scanner)
	annotations := resource.GetAnnotations()
	if annotations == nil {
		return nil, nil
	}

	hasDrift, exists := annotations["platform.io/iam-drift"]
	if !exists || hasDrift != "true" {
		return nil, nil
	}

	driftType := annotations["platform.io/drift-type"]
	if driftType != "extra_privileges" && driftType != "policy_mismatch" {
		return nil, nil
	}

	driftMessage := annotations["platform.io/drift-message"]

	finding := rules.CreateFinding(
		r,
		resource,
		fmt.Sprintf("IAM role %s has extra privileges", resource.GetName()),
		fmt.Sprintf("Role has policies or permissions not defined in Crossplane. %s", driftMessage),
		"Review the IAM drift details in the IAM Drift view. Remove unauthorized policies or update Crossplane definition.",
	)
	finding.Metadata["driftType"] = driftType
	if driftMessage != "" {
		finding.Metadata["driftMessage"] = driftMessage
	}

	return []rules.Finding{finding}, nil
}

// Helper function to parse simple quantity strings
func parseSimpleQuantity(s string) int64 {
	// Remove common suffixes for basic parsing
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "Ki")
	s = strings.TrimSuffix(s, "Mi")
	s = strings.TrimSuffix(s, "Gi")
	s = strings.TrimSuffix(s, "m") // millicores

	var val int64
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0
	}
	return val
}
