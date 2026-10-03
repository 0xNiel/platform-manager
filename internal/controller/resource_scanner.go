package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ResourceScanner scans resources and creates/updates ResourceSummary CRs
type ResourceScanner struct {
	client.Client
}

// NewResourceScanner creates a new ResourceScanner
func NewResourceScanner(c client.Client) *ResourceScanner {
	return &ResourceScanner{Client: c}
}

// ScanTenantResources scans all resources for a given tenant and creates/updates ResourceSummary CRs
func (s *ResourceScanner) ScanTenantResources(ctx context.Context, tenant *platformv1alpha1.Tenant) error {
	log := logf.FromContext(ctx).WithValues("tenant", tenant.Name)
	log.Info("Scanning tenant resources")

	// 1. Scan Kubernetes resources
	if err := s.scanKubernetesResources(ctx, tenant); err != nil {
		log.Error(err, "Failed to scan Kubernetes resources")
		return err
	}

	// 2. Scan Crossplane resources
	if err := s.scanCrossplaneResources(ctx, tenant); err != nil {
		log.Error(err, "Failed to scan Crossplane resources")
		return err
	}

	// 3. Scan ArgoCD resources
	if err := s.scanArgoResources(ctx, tenant); err != nil {
		log.Error(err, "Failed to scan ArgoCD resources")
		return err
	}

	// 4. Cleanup orphaned ResourceSummaries (resources that no longer exist)
	if err := s.cleanupOrphanedResourceSummaries(ctx, tenant); err != nil {
		log.Error(err, "Failed to cleanup orphaned ResourceSummaries")
		// Don't return error, just log it
	}

	log.Info("Completed scanning tenant resources")
	return nil
}

// scanKubernetesResources scans common Kubernetes resources
func (s *ResourceScanner) scanKubernetesResources(ctx context.Context, tenant *platformv1alpha1.Tenant) error {
	log := logf.FromContext(ctx).WithValues("tenant", tenant.Name)

	for _, ns := range tenant.Spec.Namespaces {
		// Scan Deployments
		deployList := &appsv1.DeploymentList{}
		if err := s.List(ctx, deployList, client.InNamespace(ns)); err != nil {
			log.Error(err, "Failed to list Deployments", "namespace", ns)
			return err
		}
		for _, deploy := range deployList.Items {
			if err := s.createOrUpdateResourceSummary(ctx, tenant, &deploy, platformv1alpha1.ResourceCategoryKubernetes); err != nil {
				log.Error(err, "Failed to create/update ResourceSummary for Deployment", "name", deploy.Name)
			}
		}

		// Scan StatefulSets
		stsListt := &appsv1.StatefulSetList{}
		if err := s.List(ctx, stsListt, client.InNamespace(ns)); err != nil {
			log.Error(err, "Failed to list StatefulSets", "namespace", ns)
			return err
		}
		for _, sts := range stsListt.Items {
			if err := s.createOrUpdateResourceSummary(ctx, tenant, &sts, platformv1alpha1.ResourceCategoryKubernetes); err != nil {
				log.Error(err, "Failed to create/update ResourceSummary for StatefulSet", "name", sts.Name)
			}
		}

		// Scan DaemonSets
		dsList := &appsv1.DaemonSetList{}
		if err := s.List(ctx, dsList, client.InNamespace(ns)); err != nil {
			log.Error(err, "Failed to list DaemonSets", "namespace", ns)
			return err
		}
		for _, ds := range dsList.Items {
			if err := s.createOrUpdateResourceSummary(ctx, tenant, &ds, platformv1alpha1.ResourceCategoryKubernetes); err != nil {
				log.Error(err, "Failed to create/update ResourceSummary for DaemonSet", "name", ds.Name)
			}
		}

		// Scan Jobs
		jobList := &batchv1.JobList{}
		if err := s.List(ctx, jobList, client.InNamespace(ns)); err != nil {
			log.Error(err, "Failed to list Jobs", "namespace", ns)
			return err
		}
		for _, job := range jobList.Items {
			if err := s.createOrUpdateResourceSummary(ctx, tenant, &job, platformv1alpha1.ResourceCategoryKubernetes); err != nil {
				log.Error(err, "Failed to create/update ResourceSummary for Job", "name", job.Name)
			}
		}

		// Scan CronJobs
		cronJobList := &batchv1.CronJobList{}
		if err := s.List(ctx, cronJobList, client.InNamespace(ns)); err != nil {
			log.Error(err, "Failed to list CronJobs", "namespace", ns)
			return err
		}
		for _, cronJob := range cronJobList.Items {
			if err := s.createOrUpdateResourceSummary(ctx, tenant, &cronJob, platformv1alpha1.ResourceCategoryKubernetes); err != nil {
				log.Error(err, "Failed to create/update ResourceSummary for CronJob", "name", cronJob.Name)
			}
		}
	}

	return nil
}

// scanCrossplaneResources scans Crossplane resources
func (s *ResourceScanner) scanCrossplaneResources(ctx context.Context, tenant *platformv1alpha1.Tenant) error {
	log := logf.FromContext(ctx).WithValues("tenant", tenant.Name)

	// Define the GVKs for Crossplane resources we want to track
	gvks := []schema.GroupVersionKind{
		// IAM resources
		{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "Role"},
		{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "Policy"},
		{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "RolePolicyAttachment"},
		{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "User"},
		{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "Group"},
		// Storage and data
		{Group: "s3.aws.upbound.io", Version: "v1beta1", Kind: "Bucket"},
		{Group: "dynamodb.aws.upbound.io", Version: "v1beta1", Kind: "Table"},
		// Crossplane core
		{Group: "apiextensions.crossplane.io", Version: "v1", Kind: "CompositeResourceDefinition"},
		{Group: "apiextensions.crossplane.io", Version: "v1", Kind: "Composition"},
		{Group: "pkg.crossplane.io", Version: "v1", Kind: "Provider"},
	}

	for _, gvk := range gvks {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)

		if err := s.List(ctx, list); err != nil {
			log.Error(err, "Failed to list Crossplane resources", "gvk", gvk.String())
			// Don't return error, just log and continue
			continue
		}

		for _, item := range list.Items {
			// Same ownership rule as the Crossplane watcher, so both views agree.
			if !belongsToTenant(&item, tenant) {
				continue
			}
			category := platformv1alpha1.ResourceCategoryCrossplane
			// Check if it's an IAM resource
			if strings.Contains(gvk.Group, "iam") {
				category = platformv1alpha1.ResourceCategoryIAM
			}
			if err := s.createOrUpdateResourceSummary(ctx, tenant, &item, category); err != nil {
				log.Error(err, "Failed to create/update ResourceSummary for Crossplane resource", "name", item.GetName())
			}
		}
	}

	return nil
}

// scanArgoResources scans ArgoCD Application resources
func (s *ResourceScanner) scanArgoResources(ctx context.Context, tenant *platformv1alpha1.Tenant) error {
	log := logf.FromContext(ctx).WithValues("tenant", tenant.Name)

	appList := &unstructured.UnstructuredList{}
	appList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationList",
	})

	listOpts := []client.ListOption{
		client.InNamespace("argocd"), // ArgoCD apps are typically in the argocd namespace
	}

	if err := s.List(ctx, appList, listOpts...); err != nil {
		log.Error(err, "Failed to list ArgoCD applications")
		return err
	}

	for _, app := range appList.Items {
		// Filter by tenant's ArgoProjects if specified
		if len(tenant.Spec.ArgoProjects) > 0 {
			project, _, _ := unstructured.NestedString(app.Object, "spec", "project")
			isTenantApp := false
			for _, tenantProject := range tenant.Spec.ArgoProjects {
				if project == tenantProject {
					isTenantApp = true
					break
				}
			}
			if !isTenantApp {
				continue
			}
		} else {
			// If no ArgoProjects specified, try to match by label
			if tenant.Spec.LabelSelector != nil {
				selector := labels.SelectorFromSet(tenant.Spec.LabelSelector.MatchLabels)
				if !selector.Matches(labels.Set(app.GetLabels())) {
					continue
				}
			} else {
				// Fallback to tenant name label if no explicit selector
				if app.GetLabels()["platform.io/tenant"] != tenant.Name {
					continue
				}
			}
		}

		if err := s.createOrUpdateResourceSummary(ctx, tenant, &app, platformv1alpha1.ResourceCategoryArgoCD); err != nil {
			log.Error(err, "Failed to create/update ResourceSummary for ArgoCD app", "name", app.GetName())
		}
	}

	return nil
}

// createOrUpdateResourceSummary creates or updates a ResourceSummary CR for a given resource
func (s *ResourceScanner) createOrUpdateResourceSummary(ctx context.Context, tenant *platformv1alpha1.Tenant, obj client.Object, category platformv1alpha1.ResourceCategory) error {
	log := logf.FromContext(ctx)

	// Typed objects from List come back with an empty TypeMeta, so resolve
	// the GVK from the scheme instead of trusting obj.GetObjectKind().
	gvk, err := s.GroupVersionKindFor(obj)
	if err != nil {
		return fmt.Errorf("failed to resolve GVK for %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
	}

	// Generate a unique name for the ResourceSummary
	// Format: <tenant>-<kind>-<namespace>-<name>
	summaryName := generateResourceSummaryName(tenant.Name, gvk, obj)

	resourceSummary := &platformv1alpha1.ResourceSummary{
		ObjectMeta: metav1.ObjectMeta{
			Name: summaryName,
			Labels: map[string]string{
				"platform.io/tenant":   tenant.Name,
				"platform.io/category": string(category),
			},
		},
	}

	// Create or update the spec
	_, err = controllerutil.CreateOrUpdate(ctx, s.Client, resourceSummary, func() error {
		// Populate spec
		resourceSummary.Spec = platformv1alpha1.ResourceSummarySpec{
			Group:     gvk.Group,
			Version:   gvk.Version,
			Kind:      gvk.Kind,
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
			TenantRef: tenant.Name,
			Category:  category,
		}

		// Determine provider if Crossplane resource
		if category == platformv1alpha1.ResourceCategoryCrossplane || category == platformv1alpha1.ResourceCategoryIAM {
			resourceSummary.Spec.Provider = extractProvider(gvk.Group)
		}

		return nil
	})

	if err != nil {
		log.Error(err, "Failed to create or update ResourceSummary", "name", summaryName)
		return err
	}

	// Update status separately (status subresource requires separate update)
	resourceSummary.Status = s.normalizeResourceStatus(obj, category)
	resourceSummary.Status.LastSeen = &metav1.Time{Time: time.Now()}

	if err := s.Status().Update(ctx, resourceSummary); err != nil {
		log.Error(err, "Failed to update ResourceSummary status", "name", summaryName)
		return err
	}

	return nil
}

// normalizeResourceStatus normalizes the status of a resource into a ResourceSummaryStatus
func (s *ResourceScanner) normalizeResourceStatus(obj client.Object, category platformv1alpha1.ResourceCategory) platformv1alpha1.ResourceSummaryStatus {
	status := platformv1alpha1.ResourceSummaryStatus{
		Age: time.Since(obj.GetCreationTimestamp().Time).Round(time.Second).String(),
	}

	// Extract owner chain
	status.OwnerChain = extractOwnerChain(obj)

	// Normalize based on category and type
	switch category {
	case platformv1alpha1.ResourceCategoryKubernetes:
		status.State, status.Message, status.Conditions = s.normalizeKubernetesStatus(obj)
	case platformv1alpha1.ResourceCategoryCrossplane, platformv1alpha1.ResourceCategoryIAM:
		status.State, status.Message, status.Conditions = s.normalizeCrossplaneStatus(obj)
	case platformv1alpha1.ResourceCategoryArgoCD:
		status.State, status.Message, status.Conditions = s.normalizeArgoStatus(obj)
	default:
		status.State = platformv1alpha1.ResourceStateUnknown
		status.Message = "Unknown category"
	}

	return status
}

// normalizeKubernetesStatus normalizes status for Kubernetes resources
func (s *ResourceScanner) normalizeKubernetesStatus(obj client.Object) (platformv1alpha1.ResourceState, string, []platformv1alpha1.ConditionSummary) {
	switch v := obj.(type) {
	case *appsv1.Deployment:
		return normalizeDeploymentStatus(v)
	case *appsv1.StatefulSet:
		return normalizeStatefulSetStatus(v)
	case *appsv1.DaemonSet:
		return normalizeDaemonSetStatus(v)
	case *batchv1.Job:
		return normalizeJobStatus(v)
	case *batchv1.CronJob:
		return normalizeCronJobStatus(v)
	default:
		return platformv1alpha1.ResourceStateUnknown, "Unknown resource type", nil
	}
}

// normalizeCrossplaneStatus normalizes status for Crossplane resources
func (s *ResourceScanner) normalizeCrossplaneStatus(obj client.Object) (platformv1alpha1.ResourceState, string, []platformv1alpha1.ConditionSummary) {
	// Check for crossplane.io/paused annotation first
	annotations := obj.GetAnnotations()
	if annotations != nil {
		if paused, exists := annotations[crossplanePausedAnnotation]; exists && paused == annotationValueTrue {
			return platformv1alpha1.ResourceStatePaused, "Resource reconciliation is paused", nil
		}
	}

	// For unstructured Crossplane resources, check the status.conditions field
	if u, ok := obj.(*unstructured.Unstructured); ok {
		conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
		if err != nil || !found {
			return platformv1alpha1.ResourceStateUnknown, "No status conditions", nil
		}

		conditionSummaries := []platformv1alpha1.ConditionSummary{}
		isReady := false
		isFailed := false
		isPaused := false
		message := ""

		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}

			condType, _ := condMap["type"].(string)
			condStatus, _ := condMap["status"].(string)
			condReason, _ := condMap["reason"].(string)
			condMessage, _ := condMap["message"].(string)

			// Create condition summary
			condSummary := platformv1alpha1.ConditionSummary{
				Type:    condType,
				Status:  condStatus,
				Reason:  condReason,
				Message: condMessage,
			}

			// Parse lastTransitionTime if available
			if lastTransitionStr, ok := condMap["lastTransitionTime"].(string); ok {
				if t, err := time.Parse(time.RFC3339, lastTransitionStr); err == nil {
					condSummary.LastTransitionTime = &metav1.Time{Time: t}
				}
			}

			conditionSummaries = append(conditionSummaries, condSummary)

			// Check for ReconcilePaused reason (Crossplane paused resources)
			if condReason == "ReconcilePaused" {
				isPaused = true
				message = "Reconciliation paused"
			}

			// Check for Ready condition
			if condType == "Ready" || condType == conditionTypeSynced {
				switch condStatus {
				case conditionStatusTrue:
					isReady = true
					if message == "" {
						message = condMessage
					}
				case conditionStatusFalse:
					if condReason != "ReconcilePaused" {
						isFailed = true
						if message == "" {
							message = condMessage
						}
					}
				}
			}
		}

		// Determine state - paused takes precedence
		if isPaused {
			return platformv1alpha1.ResourceStatePaused, message, conditionSummaries
		} else if isReady {
			return platformv1alpha1.ResourceStateReady, message, conditionSummaries
		} else if isFailed {
			return platformv1alpha1.ResourceStateFailed, message, conditionSummaries
		} else {
			return platformv1alpha1.ResourceStateWaiting, message, conditionSummaries
		}
	}

	return platformv1alpha1.ResourceStateUnknown, "Unable to parse status", nil
}

// normalizeArgoStatus normalizes status for ArgoCD Application resources
func (s *ResourceScanner) normalizeArgoStatus(obj client.Object) (platformv1alpha1.ResourceState, string, []platformv1alpha1.ConditionSummary) {
	app, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return platformv1alpha1.ResourceStateUnknown, "Not an ArgoCD Application", nil
	}

	// Extract sync and health status
	syncStatus, _, _ := unstructured.NestedString(app.Object, "status", "sync", "status")
	healthStatus, _, _ := unstructured.NestedString(app.Object, "status", "health", "status")

	// Determine state based on sync and health status
	var state platformv1alpha1.ResourceState
	message := fmt.Sprintf("Sync: %s, Health: %s", syncStatus, healthStatus)

	// Map ArgoCD health to our state
	switch healthStatus {
	case "Healthy":
		if syncStatus == argoSyncStatusSynced {
			state = platformv1alpha1.ResourceStateReady
		} else {
			state = platformv1alpha1.ResourceStateWaiting
		}
	case "Degraded":
		state = platformv1alpha1.ResourceStateFailed
	case "Progressing":
		state = platformv1alpha1.ResourceStateWaiting
	case "Suspended":
		state = platformv1alpha1.ResourceStatePaused
	default:
		state = platformv1alpha1.ResourceStateUnknown
	}

	// Extract conditions from ArgoCD app
	conditions := []platformv1alpha1.ConditionSummary{}
	conditionsRaw, found, _ := unstructured.NestedSlice(app.Object, "status", "conditions")
	if found {
		for _, condRaw := range conditionsRaw {
			condMap, ok := condRaw.(map[string]interface{})
			if !ok {
				continue
			}
			condType, _ := condMap["type"].(string)
			condMessage, _ := condMap["message"].(string)
			conditions = append(conditions, platformv1alpha1.ConditionSummary{
				Type:    condType,
				Message: condMessage,
			})
		}
	}

	return state, message, conditions
}

// Helper functions for Kubernetes resource normalization

func normalizeDeploymentStatus(deploy *appsv1.Deployment) (platformv1alpha1.ResourceState, string, []platformv1alpha1.ConditionSummary) {
	conditions := []platformv1alpha1.ConditionSummary{}
	for _, cond := range deploy.Status.Conditions {
		conditions = append(conditions, platformv1alpha1.ConditionSummary{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastTransitionTime: &cond.LastTransitionTime,
		})
	}

	if deploy.Spec.Paused {
		return platformv1alpha1.ResourceStatePaused, "Deployment is paused", conditions
	}

	desiredReplicas := int32(1)
	if deploy.Spec.Replicas != nil {
		desiredReplicas = *deploy.Spec.Replicas
	}

	if deploy.Status.AvailableReplicas == desiredReplicas && deploy.Status.UpdatedReplicas == desiredReplicas {
		return platformv1alpha1.ResourceStateReady, fmt.Sprintf("%d/%d replicas ready", deploy.Status.AvailableReplicas, desiredReplicas), conditions
	}

	// Check for failure conditions
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentReplicaFailure && cond.Status == corev1.ConditionTrue {
			return platformv1alpha1.ResourceStateFailed, cond.Message, conditions
		}
	}

	return platformv1alpha1.ResourceStateWaiting, fmt.Sprintf("%d/%d replicas ready", deploy.Status.AvailableReplicas, desiredReplicas), conditions
}

func normalizeStatefulSetStatus(sts *appsv1.StatefulSet) (platformv1alpha1.ResourceState, string, []platformv1alpha1.ConditionSummary) {
	conditions := []platformv1alpha1.ConditionSummary{}
	for _, cond := range sts.Status.Conditions {
		conditions = append(conditions, platformv1alpha1.ConditionSummary{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastTransitionTime: &cond.LastTransitionTime,
		})
	}

	desiredReplicas := int32(1)
	if sts.Spec.Replicas != nil {
		desiredReplicas = *sts.Spec.Replicas
	}

	if sts.Status.ReadyReplicas == desiredReplicas && sts.Status.UpdatedReplicas == desiredReplicas {
		return platformv1alpha1.ResourceStateReady, fmt.Sprintf("%d/%d replicas ready", sts.Status.ReadyReplicas, desiredReplicas), conditions
	}

	return platformv1alpha1.ResourceStateWaiting, fmt.Sprintf("%d/%d replicas ready", sts.Status.ReadyReplicas, desiredReplicas), conditions
}

func normalizeDaemonSetStatus(ds *appsv1.DaemonSet) (platformv1alpha1.ResourceState, string, []platformv1alpha1.ConditionSummary) {
	conditions := []platformv1alpha1.ConditionSummary{}
	for _, cond := range ds.Status.Conditions {
		conditions = append(conditions, platformv1alpha1.ConditionSummary{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastTransitionTime: &cond.LastTransitionTime,
		})
	}

	if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled && ds.Status.UpdatedNumberScheduled == ds.Status.DesiredNumberScheduled {
		return platformv1alpha1.ResourceStateReady, fmt.Sprintf("%d/%d pods ready", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled), conditions
	}

	return platformv1alpha1.ResourceStateWaiting, fmt.Sprintf("%d/%d pods ready", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled), conditions
}

func normalizeJobStatus(job *batchv1.Job) (platformv1alpha1.ResourceState, string, []platformv1alpha1.ConditionSummary) {
	conditions := []platformv1alpha1.ConditionSummary{}
	for _, cond := range job.Status.Conditions {
		conditions = append(conditions, platformv1alpha1.ConditionSummary{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastTransitionTime: &cond.LastTransitionTime,
		})
	}

	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
			return platformv1alpha1.ResourceStateReady, "Job completed successfully", conditions
		}
		if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
			return platformv1alpha1.ResourceStateFailed, cond.Message, conditions
		}
	}

	return platformv1alpha1.ResourceStateWaiting, fmt.Sprintf("%d/%d pods succeeded", job.Status.Succeeded, job.Status.Active+job.Status.Succeeded+job.Status.Failed), conditions
}

func normalizeCronJobStatus(cronJob *batchv1.CronJob) (platformv1alpha1.ResourceState, string, []platformv1alpha1.ConditionSummary) {
	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		return platformv1alpha1.ResourceStatePaused, "CronJob is suspended", nil
	}

	message := "CronJob is scheduled"
	if cronJob.Status.LastScheduleTime != nil {
		message = fmt.Sprintf("Last scheduled: %s", cronJob.Status.LastScheduleTime.Format(time.RFC3339))
	}

	return platformv1alpha1.ResourceStateReady, message, nil
}

// extractOwnerChain extracts the owner chain from a resource
func extractOwnerChain(obj client.Object) []platformv1alpha1.OwnerRef {
	ownerChain := []platformv1alpha1.OwnerRef{}
	for _, owner := range obj.GetOwnerReferences() {
		ownerChain = append(ownerChain, platformv1alpha1.OwnerRef{
			APIVersion: owner.APIVersion,
			Kind:       owner.Kind,
			Name:       owner.Name,
			UID:        string(owner.UID),
		})
	}
	return ownerChain
}

// extractProvider extracts the provider name from a Crossplane resource group
func extractProvider(group string) string {
	// Example: "iam.aws.upbound.io" -> "aws"
	parts := strings.Split(group, ".")
	if len(parts) >= 2 {
		return parts[1] // Return "aws", "gcp", "azure", etc.
	}
	return ""
}

// generateResourceSummaryName generates a unique name for a ResourceSummary
func generateResourceSummaryName(tenantName string, gvk schema.GroupVersionKind, obj client.Object) string {
	kind := strings.ToLower(gvk.Kind)
	namespace := obj.GetNamespace()
	name := obj.GetName()

	if namespace == "" {
		// Cluster-scoped resource
		return fmt.Sprintf("%s-%s-%s", tenantName, kind, name)
	}
	// Namespaced resource
	return fmt.Sprintf("%s-%s-%s-%s", tenantName, kind, namespace, name)
}

// cleanupOrphanedResourceSummaries removes ResourceSummaries for resources that no longer exist
func (s *ResourceScanner) cleanupOrphanedResourceSummaries(ctx context.Context, tenant *platformv1alpha1.Tenant) error {
	log := logf.FromContext(ctx).WithValues("tenant", tenant.Name)

	// List all ResourceSummaries for this tenant
	summaryList := &platformv1alpha1.ResourceSummaryList{}
	listOpts := []client.ListOption{
		client.MatchingLabels{
			"platform.io/tenant": tenant.Name,
		},
	}

	if err := s.List(ctx, summaryList, listOpts...); err != nil {
		return fmt.Errorf("failed to list ResourceSummaries: %w", err)
	}

	deletedCount := 0
	for _, summary := range summaryList.Items {
		// Check if the underlying resource still exists
		exists, err := s.resourceExists(ctx, &summary)
		if err != nil {
			log.Error(err, "Failed to check if resource exists", "summary", summary.Name)
			continue
		}

		if !exists {
			// Resource no longer exists, delete the ResourceSummary
			if err := s.Delete(ctx, &summary); err != nil {
				log.Error(err, "Failed to delete orphaned ResourceSummary", "summary", summary.Name)
				continue
			}
			log.Info("Deleted orphaned ResourceSummary", "summary", summary.Name, "resource", summary.Spec.Name)
			deletedCount++
		}
	}

	if deletedCount > 0 {
		log.Info("Cleanup completed", "orphanedResourcesDeleted", deletedCount)
	}

	return nil
}

// resourceExists checks if the underlying resource still exists for a ResourceSummary
func (s *ResourceScanner) resourceExists(ctx context.Context, summary *platformv1alpha1.ResourceSummary) (bool, error) {
	// Build the GVK from the ResourceSummary spec
	gvk := schema.GroupVersionKind{
		Group:   summary.Spec.Group,
		Version: summary.Spec.Version,
		Kind:    summary.Spec.Kind,
	}

	// Create an unstructured object to query
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	// Build the object key
	key := client.ObjectKey{
		Name: summary.Spec.Name,
	}
	if summary.Spec.Namespace != "" {
		key.Namespace = summary.Spec.Namespace
	}

	// Try to get the resource
	err := s.Get(ctx, key, obj)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Resource not found, it no longer exists
			return false, nil
		}
		// Other error occurred
		return false, err
	}

	// Resource exists
	return true, nil
}
