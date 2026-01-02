package rules

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Helper functions for common rule patterns

// GetPodFromResource extracts a Pod from an unstructured resource
func GetPodFromResource(resource *unstructured.Unstructured) (*corev1.Pod, error) {
	if resource.GetKind() != "Pod" {
		return nil, fmt.Errorf("resource is not a Pod")
	}

	pod := &corev1.Pod{}
	err := convertUnstructuredTo(resource, pod)
	return pod, err
}

// GetAgeInMinutes returns how old a resource is in minutes
func GetAgeInMinutes(creationTime metav1.Time, now time.Time) float64 {
	return now.Sub(creationTime.Time).Minutes()
}

// GetAgeInHours returns how old a resource is in hours
func GetAgeInHours(creationTime metav1.Time, now time.Time) float64 {
	return now.Sub(creationTime.Time).Hours()
}

// HasLabel checks if a resource has a specific label
func HasLabel(resource *unstructured.Unstructured, key, value string) bool {
	labels := resource.GetLabels()
	if labels == nil {
		return false
	}
	v, exists := labels[key]
	return exists && v == value
}

// GetAnnotation gets an annotation value
func GetAnnotation(resource *unstructured.Unstructured, key string) (string, bool) {
	annotations := resource.GetAnnotations()
	if annotations == nil {
		return "", false
	}
	v, exists := annotations[key]
	return v, exists
}

// IsPaused checks if a Crossplane resource is paused
func IsPaused(resource *unstructured.Unstructured) bool {
	val, exists := GetAnnotation(resource, "crossplane.io/paused")
	if !exists {
		return false
	}
	paused, _ := strconv.ParseBool(val)
	return paused
}

// GetCondition finds a condition by type
func GetCondition(conditions []interface{}, condType string) (map[string]interface{}, bool) {
	for _, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := condMap["type"].(string); t == condType {
			return condMap, true
		}
	}
	return nil, false
}

// GetConditionStatus gets the status of a condition
func GetConditionStatus(conditions []interface{}, condType string) string {
	cond, found := GetCondition(conditions, condType)
	if !found {
		return ""
	}
	status, _ := cond["status"].(string)
	return status
}

// GetConditionReason gets the reason of a condition
func GetConditionReason(conditions []interface{}, condType string) string {
	cond, found := GetCondition(conditions, condType)
	if !found {
		return ""
	}
	reason, _ := cond["reason"].(string)
	return reason
}

// GetConditionMessage gets the message of a condition
func GetConditionMessage(conditions []interface{}, condType string) string {
	cond, found := GetCondition(conditions, condType)
	if !found {
		return ""
	}
	message, _ := cond["message"].(string)
	return message
}

// GetNestedConditions extracts conditions from status.conditions
func GetNestedConditions(resource *unstructured.Unstructured) []interface{} {
	conditions, found, err := unstructured.NestedSlice(resource.Object, "status", "conditions")
	if err != nil || !found {
		return nil
	}
	return conditions
}

// IsCrossplaneResource checks if a resource is managed by Crossplane
func IsCrossplaneResource(resource *unstructured.Unstructured) bool {
	group := resource.GetObjectKind().GroupVersionKind().Group
	return strings.Contains(group, ".crossplane.io") ||
		strings.Contains(group, ".upbound.io") ||
		strings.Contains(group, ".aws.upbound.io")
}

// IsArgoResource checks if a resource is an ArgoCD resource
func IsArgoResource(resource *unstructured.Unstructured) bool {
	group := resource.GetObjectKind().GroupVersionKind().Group
	return strings.Contains(group, "argoproj.io")
}

// CreateResourceRef creates a ResourceReference from an unstructured resource
func CreateResourceRef(resource *unstructured.Unstructured) ResourceReference {
	gvk := resource.GroupVersionKind()
	return ResourceReference{
		Group:     gvk.Group,
		Version:   gvk.Version,
		Kind:      gvk.Kind,
		Namespace: resource.GetNamespace(),
		Name:      resource.GetName(),
		UID:       string(resource.GetUID()),
	}
}

// CreateFinding creates a Finding with common fields populated
func CreateFinding(rule Rule, resource *unstructured.Unstructured, title, message, recommendation string) Finding {
	return Finding{
		RuleID:         rule.ID(),
		RuleName:       rule.Name(),
		Severity:       rule.Severity(),
		Title:          title,
		Message:        message,
		Recommendation: recommendation,
		ResourceRef:    CreateResourceRef(resource),
		Status:         FindingStatusActive,
		Metadata:       make(map[string]string),
	}
}

// convertUnstructuredTo converts an unstructured object to a typed object
func convertUnstructuredTo(u *unstructured.Unstructured, obj interface{}) error {
	return unstructuredToTyped(u.Object, obj)
}

// This is a simplified conversion - in production you'd use runtime.DefaultUnstructuredConverter
func unstructuredToTyped(in map[string]interface{}, out interface{}) error {
	// This is a placeholder - in real implementation, use:
	// return runtime.DefaultUnstructuredConverter.FromUnstructured(in, out)
	// For now, we'll handle unstructured directly in rules
	return fmt.Errorf("conversion not implemented - use unstructured methods")
}
