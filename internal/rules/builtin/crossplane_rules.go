package builtin

import (
	"fmt"
	"strings"

	"github.com/platform-manager/platform-manager/internal/rules"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CrossplaneFailedIAMRule detects Crossplane resources that failed due to IAM issues
type CrossplaneFailedIAMRule struct{}

func (r *CrossplaneFailedIAMRule) ID() string {
	return "xr-failed-iam"
}

func (r *CrossplaneFailedIAMRule) Name() string {
	return "Crossplane Resource Failed - IAM"
}

func (r *CrossplaneFailedIAMRule) Description() string {
	return "Detects Crossplane resources that failed due to IAM permission issues"
}

func (r *CrossplaneFailedIAMRule) Severity() rules.Severity {
	return rules.SeverityCritical
}

func (r *CrossplaneFailedIAMRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return rules.IsCrossplaneResource(resource)
}

func (r *CrossplaneFailedIAMRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	conditions := rules.GetNestedConditions(resource)
	if len(conditions) == 0 {
		return nil, nil
	}

	// Check if Ready condition is False
	readyStatus := rules.GetConditionStatus(conditions, "Ready")
	if readyStatus != "False" {
		return nil, nil
	}

	// Look for IAM-related errors in conditions
	syncedMessage := rules.GetConditionMessage(conditions, "Synced")
	syncedReason := rules.GetConditionReason(conditions, "Synced")
	readyMessage := rules.GetConditionMessage(conditions, "Ready")

	// Check for IAM-related keywords
	combinedMessage := strings.ToLower(syncedMessage + " " + syncedReason + " " + readyMessage)
	iamKeywords := []string{"accessdenied", "unauthorized", "forbidden", "iam", "permission", "not authorized"}

	isIAMIssue := false
	for _, keyword := range iamKeywords {
		if strings.Contains(combinedMessage, keyword) {
			isIAMIssue = true
			break
		}
	}

	if !isIAMIssue {
		return nil, nil
	}

	finding := rules.CreateFinding(
		r,
		resource,
		fmt.Sprintf("Crossplane resource %s failed due to IAM issue", resource.GetName()),
		fmt.Sprintf("Resource is in Failed state due to IAM permissions. Message: %s", syncedMessage),
		"Check IAM role permissions for the Crossplane provider. Verify the provider has necessary permissions to create/update this resource.",
	)
	finding.Metadata["syncedReason"] = syncedReason
	finding.Metadata["readyStatus"] = readyStatus

	return []rules.Finding{finding}, nil
}

// PausedButSyncingRule detects Crossplane resources that are paused but still being modified by ArgoCD
type PausedButSyncingRule struct{}

func (r *PausedButSyncingRule) ID() string {
	return "paused-but-syncing"
}

func (r *PausedButSyncingRule) Name() string {
	return "Paused Resource Still Syncing"
}

func (r *PausedButSyncingRule) Description() string {
	return "Detects Crossplane resources marked as paused but ArgoCD is still trying to sync them"
}

func (r *PausedButSyncingRule) Severity() rules.Severity {
	return rules.SeverityMedium
}

func (r *PausedButSyncingRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return rules.IsCrossplaneResource(resource) && rules.IsPaused(resource)
}

func (r *PausedButSyncingRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	// Check if managed by ArgoCD
	annotations := resource.GetAnnotations()
	if annotations == nil {
		return nil, nil
	}

	argoApp, hasArgoAnnotation := annotations["argocd.argoproj.io/instance"]
	if !hasArgoAnnotation {
		return nil, nil
	}

	// Check for recent modifications
	managedFields := resource.GetManagedFields()
	for _, field := range managedFields {
		if field.Manager == "argocd-controller" {
			// ArgoCD is still managing this resource
			ageMinutes := rules.GetAgeInMinutes(resource.GetCreationTimestamp(), ctx.Now)

			finding := rules.CreateFinding(
				r,
				resource,
				fmt.Sprintf("Paused resource %s still managed by ArgoCD", resource.GetName()),
				fmt.Sprintf("Resource is marked as paused (crossplane.io/paused=true) but ArgoCD application '%s' is still managing it. This may cause conflicts.",
					argoApp),
				"Either remove the pause annotation or exclude this resource from ArgoCD sync using ignoreDifferences.",
			)
			finding.Metadata["argoApp"] = argoApp
			finding.Metadata["ageMinutes"] = fmt.Sprintf("%.0f", ageMinutes)

			return []rules.Finding{finding}, nil
		}
	}

	return nil, nil
}

// ProviderUnhealthyRule detects unhealthy Crossplane providers
type ProviderUnhealthyRule struct{}

func (r *ProviderUnhealthyRule) ID() string {
	return "provider-unhealthy"
}

func (r *ProviderUnhealthyRule) Name() string {
	return "Crossplane Provider Unhealthy"
}

func (r *ProviderUnhealthyRule) Description() string {
	return "Detects Crossplane providers that are not in healthy state"
}

func (r *ProviderUnhealthyRule) Severity() rules.Severity {
	return rules.SeverityCritical
}

func (r *ProviderUnhealthyRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return resource.GetKind() == "Provider" &&
		strings.Contains(resource.GetAPIVersion(), "pkg.crossplane.io")
}

func (r *ProviderUnhealthyRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	conditions := rules.GetNestedConditions(resource)
	if len(conditions) == 0 {
		return nil, nil
	}

	// Check Healthy condition
	healthyStatus := rules.GetConditionStatus(conditions, "Healthy")
	if healthyStatus == "True" {
		return nil, nil
	}

	healthyReason := rules.GetConditionReason(conditions, "Healthy")
	healthyMessage := rules.GetConditionMessage(conditions, "Healthy")

	// Check Installed condition
	installedStatus := rules.GetConditionStatus(conditions, "Installed")
	installedReason := rules.GetConditionReason(conditions, "Installed")

	var message strings.Builder
	message.WriteString(fmt.Sprintf("Provider is not healthy. Status: %s", healthyStatus))
	if healthyMessage != "" {
		message.WriteString(fmt.Sprintf(". Message: %s", healthyMessage))
	}
	if installedStatus != "True" {
		message.WriteString(fmt.Sprintf(". Installation status: %s (%s)", installedStatus, installedReason))
	}

	finding := rules.CreateFinding(
		r,
		resource,
		fmt.Sprintf("Provider %s is unhealthy", resource.GetName()),
		message.String(),
		"Check provider logs: kubectl logs -n crossplane-system deployment/"+resource.GetName(),
	)
	finding.Metadata["healthyStatus"] = healthyStatus
	finding.Metadata["healthyReason"] = healthyReason
	finding.Metadata["installedStatus"] = installedStatus

	return []rules.Finding{finding}, nil
}

// StaleResourceRule detects ResourceSummary objects that haven't been updated recently
type StaleResourceRule struct{}

func (r *StaleResourceRule) ID() string {
	return "stale-resource"
}

func (r *StaleResourceRule) Name() string {
	return "Stale Resource Summary"
}

func (r *StaleResourceRule) Description() string {
	return "Detects ResourceSummary objects that haven't been updated in over 1 hour"
}

func (r *StaleResourceRule) Severity() rules.Severity {
	return rules.SeverityMedium
}

func (r *StaleResourceRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return resource.GetKind() == "ResourceSummary" &&
		strings.Contains(resource.GetAPIVersion(), "platform.io")
}

func (r *StaleResourceRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	// Get last updated time from status
	lastSeen, found, err := unstructured.NestedString(resource.Object, "status", "lastSeen")
	if err != nil || !found {
		// Use creation time as fallback
		ageHours := rules.GetAgeInHours(resource.GetCreationTimestamp(), ctx.Now)
		if ageHours > 1 {
			finding := rules.CreateFinding(
				r,
				resource,
				fmt.Sprintf("ResourceSummary %s has no lastSeen timestamp", resource.GetName()),
				fmt.Sprintf("Resource has been created %.1f hours ago but never updated", ageHours),
				"Check if the resource scanner is running and has permissions to update ResourceSummary objects.",
			)
			finding.Metadata["ageHours"] = fmt.Sprintf("%.1f", ageHours)
			return []rules.Finding{finding}, nil
		}
		return nil, nil
	}

	// Parse the timestamp (RFC3339 format)
	// For simplicity, we'll check if it's been more than an hour since creation
	ageHours := rules.GetAgeInHours(resource.GetCreationTimestamp(), ctx.Now)
	if ageHours > 1 {
		finding := rules.CreateFinding(
			r,
			resource,
			fmt.Sprintf("ResourceSummary %s is stale", resource.GetName()),
			fmt.Sprintf("Resource hasn't been updated recently. Last seen: %s (%.1f hours old)", lastSeen, ageHours),
			"Investigate why the resource scanner hasn't updated this resource. The underlying resource may no longer exist.",
		)
		finding.Metadata["lastSeen"] = lastSeen
		finding.Metadata["ageHours"] = fmt.Sprintf("%.1f", ageHours)
		return []rules.Finding{finding}, nil
	}

	return nil, nil
}
