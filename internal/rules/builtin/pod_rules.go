package builtin

import (
	"fmt"
	"strings"

	"github.com/0xNiel/platform-manager/internal/rules"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const kindPod = "Pod"

// CrashLoopBackOffRule detects pods in CrashLoopBackOff state
type CrashLoopBackOffRule struct{}

func (r *CrashLoopBackOffRule) ID() string {
	return "crashloop-backoff"
}

func (r *CrashLoopBackOffRule) Name() string {
	return "Pod CrashLoopBackOff"
}

func (r *CrashLoopBackOffRule) Description() string {
	return "Detects pods that are repeatedly crashing with more than 5 restarts"
}

func (r *CrashLoopBackOffRule) Severity() rules.Severity {
	return rules.SeverityHigh
}

func (r *CrashLoopBackOffRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return resource.GetKind() == kindPod
}

func (r *CrashLoopBackOffRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	// Get container statuses
	containerStatuses, found, err := unstructured.NestedSlice(resource.Object, "status", "containerStatuses")
	if err != nil || !found {
		return nil, nil
	}

	var findings []rules.Finding

	for _, statusInterface := range containerStatuses {
		status, ok := statusInterface.(map[string]interface{})
		if !ok {
			continue
		}

		containerName, _ := status["name"].(string)
		restartCount, _ := status["restartCount"].(int64)

		// Check for CrashLoopBackOff state
		state, found, _ := unstructured.NestedMap(status, "state", "waiting")
		if found {
			reason, _ := state["reason"].(string)
			message, _ := state["message"].(string)

			if reason == "CrashLoopBackOff" && restartCount > 5 {
				finding := rules.CreateFinding(
					r,
					resource,
					fmt.Sprintf("Pod %s is in CrashLoopBackOff", resource.GetName()),
					fmt.Sprintf("Container '%s' has crashed %d times. Last message: %s",
						containerName, restartCount, message),
					"Check container logs for crash details: kubectl logs "+resource.GetName()+" -n "+resource.GetNamespace()+" -c "+containerName,
				)
				finding.Metadata["containerName"] = containerName
				finding.Metadata["restartCount"] = fmt.Sprintf("%d", restartCount)
				finding.Metadata["reason"] = reason
				findings = append(findings, finding)
			}
		}
	}

	return findings, nil
}

// ImagePullBackOffRule detects pods with image pull failures
type ImagePullBackOffRule struct{}

func (r *ImagePullBackOffRule) ID() string {
	return "image-pull-failed"
}

func (r *ImagePullBackOffRule) Name() string {
	return "Image Pull Failed"
}

func (r *ImagePullBackOffRule) Description() string {
	return "Detects pods that cannot pull container images for more than 10 minutes"
}

func (r *ImagePullBackOffRule) Severity() rules.Severity {
	return rules.SeverityHigh
}

func (r *ImagePullBackOffRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return resource.GetKind() == kindPod
}

func (r *ImagePullBackOffRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	// Check pod age
	ageMinutes := rules.GetAgeInMinutes(resource.GetCreationTimestamp(), ctx.Now)
	if ageMinutes < 10 {
		return nil, nil // Too new, give it time
	}

	// Get container statuses
	containerStatuses, found, err := unstructured.NestedSlice(resource.Object, "status", "containerStatuses")
	if err != nil || !found {
		return nil, nil
	}

	var findings []rules.Finding

	for _, statusInterface := range containerStatuses {
		status, ok := statusInterface.(map[string]interface{})
		if !ok {
			continue
		}

		containerName, _ := status["name"].(string)
		image, _ := status["image"].(string)

		// Check for ImagePullBackOff or ErrImagePull
		state, found, _ := unstructured.NestedMap(status, "state", "waiting")
		if found {
			reason, _ := state["reason"].(string)
			message, _ := state["message"].(string)

			if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
				finding := rules.CreateFinding(
					r,
					resource,
					fmt.Sprintf("Pod %s cannot pull image", resource.GetName()),
					fmt.Sprintf("Container '%s' failed to pull image '%s'. Error: %s",
						containerName, image, message),
					"Verify image exists and registry credentials are configured correctly. Check if image tag is valid.",
				)
				finding.Metadata["containerName"] = containerName
				finding.Metadata["image"] = image
				finding.Metadata["reason"] = reason
				findings = append(findings, finding)
			}
		}
	}

	return findings, nil
}

// PodPendingRule detects pods stuck in pending state
type PodPendingRule struct{}

func (r *PodPendingRule) ID() string {
	return "pod-pending-long"
}

func (r *PodPendingRule) Name() string {
	return "Pod Stuck Pending"
}

func (r *PodPendingRule) Description() string {
	return "Detects pods that have been pending for more than 15 minutes"
}

func (r *PodPendingRule) Severity() rules.Severity {
	return rules.SeverityMedium
}

func (r *PodPendingRule) AppliesTo(resource *unstructured.Unstructured) bool {
	return resource.GetKind() == kindPod
}

func (r *PodPendingRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
	resource := ctx.Resource

	// Get pod phase
	phase, found, err := unstructured.NestedString(resource.Object, "status", "phase")
	if err != nil || !found || phase != "Pending" {
		return nil, nil
	}

	// Check age
	ageMinutes := rules.GetAgeInMinutes(resource.GetCreationTimestamp(), ctx.Now)
	if ageMinutes < 15 {
		return nil, nil
	}

	// Get conditions to understand why it's pending
	conditions := rules.GetNestedConditions(resource)
	podScheduledStatus := rules.GetConditionStatus(conditions, "PodScheduled")
	podScheduledReason := rules.GetConditionReason(conditions, "PodScheduled")
	podScheduledMessage := rules.GetConditionMessage(conditions, "PodScheduled")

	var message strings.Builder
	message.WriteString(fmt.Sprintf("Pod has been pending for %.0f minutes. ", ageMinutes))

	if podScheduledStatus == "False" {
		message.WriteString(fmt.Sprintf("Scheduling issue: %s - %s", podScheduledReason, podScheduledMessage))
	} else {
		message.WriteString("Reason unknown. Check pod events for details.")
	}

	finding := rules.CreateFinding(
		r,
		resource,
		fmt.Sprintf("Pod %s stuck in Pending state", resource.GetName()),
		message.String(),
		"Check pod events: kubectl describe pod "+resource.GetName()+" -n "+resource.GetNamespace(),
	)
	finding.Metadata["phase"] = phase
	finding.Metadata["ageMinutes"] = fmt.Sprintf("%.0f", ageMinutes)
	if podScheduledReason != "" {
		finding.Metadata["schedulingReason"] = podScheduledReason
	}

	return []rules.Finding{finding}, nil
}
