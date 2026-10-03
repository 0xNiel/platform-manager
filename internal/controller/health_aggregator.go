/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
)

// HealthAggregator aggregates health from various sources
type HealthAggregator struct {
	client            client.Client
	crossplaneWatcher *CrossplaneWatcher
	argoWatcher       *ArgoWatcher
	iamScanner        *IAMDriftScanner
}

// NewHealthAggregator creates a new HealthAggregator
func NewHealthAggregator(c client.Client) *HealthAggregator {
	return &HealthAggregator{
		client:            c,
		crossplaneWatcher: NewCrossplaneWatcher(c),
		argoWatcher:       NewArgoWatcher(c),
		iamScanner:        nil, // Will be set later via SetIAMScanner
	}
}

// SetIAMScanner sets the IAM drift scanner
func (a *HealthAggregator) SetIAMScanner(scanner *IAMDriftScanner) {
	a.iamScanner = scanner
}

// AggregateHealth aggregates health information for a tenant
func (a *HealthAggregator) AggregateHealth(ctx context.Context, tenant *platformv1alpha1.Tenant) (*platformv1alpha1.TenantHealthStatus, error) {
	log := logf.FromContext(ctx).WithName("health-aggregator")

	status := &platformv1alpha1.TenantHealthStatus{
		OverallHealth: platformv1alpha1.HealthLevelUnknown,
	}

	// Scan Crossplane resources
	crossplaneCounts, err := a.crossplaneWatcher.ScanCrossplaneResources(ctx, tenant)
	if err != nil {
		log.Error(err, "Failed to scan Crossplane resources")
	}
	status.CrossplaneResources = crossplaneCounts

	// Scan Kubernetes resources
	k8sCounts, err := a.scanKubernetesResources(ctx, tenant)
	if err != nil {
		log.Error(err, "Failed to scan Kubernetes resources")
	}
	status.KubernetesResources = k8sCounts

	// Get IAM drift from scanner if available
	if a.iamScanner != nil {
		tenantDrift, ok := a.iamScanner.GetTenantSummary(tenant.Name)
		if ok {
			// Convert IAM scanner summary to TenantHealth IAMDriftSummary
			lastChecked := metav1.NewTime(tenantDrift.LastChecked)
			status.IAMDrift = platformv1alpha1.IAMDriftSummary{
				TotalRoles:        tenantDrift.TotalRoles,
				TotalPolicies:     tenantDrift.TotalPolicies,
				RolesWithDrift:    tenantDrift.RolesWithDrift,
				PoliciesWithDrift: tenantDrift.PoliciesWithDrift,
				ExtraPrivileges:   tenantDrift.CriticalDrifts, // Critical drifts are extra privileges
				MissingPrivileges: tenantDrift.HighDrifts,     // High drifts are missing privileges
				LastChecked:       &lastChecked,
			}
		} else {
			// Fall back to basic counting if drift scanner hasn't run yet
			iamSummary, err := a.crossplaneWatcher.ScanIAMResources(ctx, tenant)
			if err != nil {
				log.Error(err, "Failed to scan IAM resources")
			}
			status.IAMDrift = iamSummary
		}
	} else {
		// Fall back to basic counting if no scanner available
		iamSummary, err := a.crossplaneWatcher.ScanIAMResources(ctx, tenant)
		if err != nil {
			log.Error(err, "Failed to scan IAM resources")
		}
		status.IAMDrift = iamSummary
	}

	// Scan ArgoCD applications
	argoSummary, err := a.argoWatcher.ScanArgoApplications(ctx, tenant)
	if err != nil {
		log.Error(err, "Failed to scan ArgoCD applications")
	}
	status.Argo = argoSummary

	// Determine overall health
	status.OverallHealth = a.calculateOverallHealth(status)

	// Aggregate top issues
	status.TopIssues = a.collectTopIssues(ctx, status, crossplaneCounts, k8sCounts)

	log.Info("Aggregated health for tenant",
		"tenant", tenant.Name,
		"overallHealth", status.OverallHealth,
		"crossplaneTotal", crossplaneCounts.Total,
		"k8sTotal", k8sCounts.Total)

	return status, nil
}

// scanKubernetesResources scans standard Kubernetes resources in tenant namespaces
func (a *HealthAggregator) scanKubernetesResources(ctx context.Context, tenant *platformv1alpha1.Tenant) (platformv1alpha1.ResourceStateCounts, error) {
	log := logf.FromContext(ctx).WithName("health-aggregator")

	counts := platformv1alpha1.ResourceStateCounts{}

	// Scan each namespace
	for _, namespace := range tenant.Spec.Namespaces {
		// Scan Deployments
		deployments := &appsv1.DeploymentList{}
		if err := a.client.List(ctx, deployments, client.InNamespace(namespace)); err != nil {
			log.V(1).Info("Failed to list deployments", "namespace", namespace, "error", err)
		} else {
			for _, dep := range deployments.Items {
				state := a.normalizeDeploymentState(&dep)
				a.incrementCount(&counts, state)
			}
		}

		// Scan StatefulSets
		statefulsets := &appsv1.StatefulSetList{}
		if err := a.client.List(ctx, statefulsets, client.InNamespace(namespace)); err != nil {
			log.V(1).Info("Failed to list statefulsets", "namespace", namespace, "error", err)
		} else {
			for _, sts := range statefulsets.Items {
				state := a.normalizeStatefulSetState(&sts)
				a.incrementCount(&counts, state)
			}
		}

		// Scan DaemonSets
		daemonsets := &appsv1.DaemonSetList{}
		if err := a.client.List(ctx, daemonsets, client.InNamespace(namespace)); err != nil {
			log.V(1).Info("Failed to list daemonsets", "namespace", namespace, "error", err)
		} else {
			for _, ds := range daemonsets.Items {
				state := a.normalizeDaemonSetState(&ds)
				a.incrementCount(&counts, state)
			}
		}

		// Scan Pods (to catch standalone pods and identify failed states)
		pods := &corev1.PodList{}
		if err := a.client.List(ctx, pods, client.InNamespace(namespace)); err != nil {
			log.V(1).Info("Failed to list pods", "namespace", namespace, "error", err)
		} else {
			// Count pods without owner references (standalone pods)
			for _, pod := range pods.Items {
				if len(pod.OwnerReferences) == 0 {
					state := a.normalizePodState(&pod)
					a.incrementCount(&counts, state)
				}
			}
		}
	}

	return counts, nil
}

// normalizeDeploymentState determines the state of a Deployment
func (a *HealthAggregator) normalizeDeploymentState(dep *appsv1.Deployment) CrossplaneResourceState {
	if dep.Spec.Replicas == nil {
		return CrossplaneStateUnknown
	}

	desiredReplicas := *dep.Spec.Replicas
	readyReplicas := dep.Status.ReadyReplicas

	// Check for paused annotation
	if paused, ok := dep.Annotations["paused"]; ok && paused == "true" {
		return CrossplaneStatePaused
	}

	// Check if all replicas are ready
	if readyReplicas == desiredReplicas && desiredReplicas > 0 {
		return CrossplaneStateReady
	}

	// Check if progressing
	if dep.Status.AvailableReplicas > 0 && readyReplicas < desiredReplicas {
		return CrossplaneStateWaiting
	}

	// Check for failures
	if readyReplicas == 0 && desiredReplicas > 0 {
		// Check conditions for failure reasons
		for _, cond := range dep.Status.Conditions {
			if cond.Type == appsv1.DeploymentProgressing && cond.Status == corev1.ConditionFalse {
				return CrossplaneStateFailed
			}
		}
		return CrossplaneStateWaiting
	}

	return CrossplaneStateUnknown
}

// normalizeStatefulSetState determines the state of a StatefulSet
func (a *HealthAggregator) normalizeStatefulSetState(sts *appsv1.StatefulSet) CrossplaneResourceState {
	if sts.Spec.Replicas == nil {
		return CrossplaneStateUnknown
	}

	desiredReplicas := *sts.Spec.Replicas
	readyReplicas := sts.Status.ReadyReplicas

	if readyReplicas == desiredReplicas && desiredReplicas > 0 {
		return CrossplaneStateReady
	}

	if readyReplicas > 0 && readyReplicas < desiredReplicas {
		return CrossplaneStateWaiting
	}

	if readyReplicas == 0 && desiredReplicas > 0 {
		return CrossplaneStateFailed
	}

	return CrossplaneStateUnknown
}

// normalizeDaemonSetState determines the state of a DaemonSet
func (a *HealthAggregator) normalizeDaemonSetState(ds *appsv1.DaemonSet) CrossplaneResourceState {
	desiredScheduled := ds.Status.DesiredNumberScheduled
	numberReady := ds.Status.NumberReady

	if numberReady == desiredScheduled && desiredScheduled > 0 {
		return CrossplaneStateReady
	}

	if numberReady > 0 && numberReady < desiredScheduled {
		return CrossplaneStateWaiting
	}

	if numberReady == 0 && desiredScheduled > 0 {
		return CrossplaneStateFailed
	}

	return CrossplaneStateUnknown
}

// normalizePodState determines the state of a Pod
func (a *HealthAggregator) normalizePodState(pod *corev1.Pod) CrossplaneResourceState {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		// Check if all containers are ready
		allReady := true
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				allReady = false
				break
			}
		}
		if allReady {
			return CrossplaneStateReady
		}
		return CrossplaneStateWaiting

	case corev1.PodSucceeded:
		return CrossplaneStateReady

	case corev1.PodPending:
		return CrossplaneStateWaiting

	case corev1.PodFailed:
		return CrossplaneStateFailed

	default:
		return CrossplaneStateUnknown
	}
}

// incrementCount increments the appropriate counter based on state
func (a *HealthAggregator) incrementCount(counts *platformv1alpha1.ResourceStateCounts, state CrossplaneResourceState) {
	switch state {
	case CrossplaneStateReady:
		counts.Ready++
	case CrossplaneStateFailed:
		counts.Failed++
	case CrossplaneStateWaiting:
		counts.Waiting++
	case CrossplaneStatePaused:
		counts.Paused++
	default:
		counts.Unknown++
	}
	counts.Total++
}

// calculateOverallHealth determines the overall health level
func (a *HealthAggregator) calculateOverallHealth(status *platformv1alpha1.TenantHealthStatus) platformv1alpha1.HealthLevel {
	totalResources := status.CrossplaneResources.Total + status.KubernetesResources.Total
	if totalResources == 0 {
		return platformv1alpha1.HealthLevelUnknown
	}

	totalFailed := status.CrossplaneResources.Failed + status.KubernetesResources.Failed

	// Critical if more than 50% failed or any ArgoCD app degraded
	failureRate := float64(totalFailed) / float64(totalResources)
	if failureRate > 0.5 || status.Argo.Degraded > 0 {
		return platformv1alpha1.HealthLevelCritical
	}

	// Degraded if any failures or out-of-sync apps
	if totalFailed > 0 || status.Argo.OutOfSync > 0 {
		return platformv1alpha1.HealthLevelDegraded
	}

	// Healthy if majority are ready
	totalReady := status.CrossplaneResources.Ready + status.KubernetesResources.Ready
	if float64(totalReady)/float64(totalResources) > 0.8 {
		return platformv1alpha1.HealthLevelHealthy
	}

	return platformv1alpha1.HealthLevelDegraded
}

// collectTopIssues collects the top issues for the tenant
func (a *HealthAggregator) collectTopIssues(ctx context.Context, status *platformv1alpha1.TenantHealthStatus, crossplaneCounts, k8sCounts platformv1alpha1.ResourceStateCounts) []platformv1alpha1.Issue {
	issues := []platformv1alpha1.Issue{}

	// Issue for failed Crossplane resources
	if crossplaneCounts.Failed > 0 {
		issues = append(issues, platformv1alpha1.Issue{
			Severity:    platformv1alpha1.IssueSeverityCritical,
			Message:     "Crossplane resources in failed state",
			ResourceRef: "",
			Timestamp:   platformv1alpha1.Now(),
		})
	}

	// Issue for failed Kubernetes resources
	if k8sCounts.Failed > 0 {
		issues = append(issues, platformv1alpha1.Issue{
			Severity:    platformv1alpha1.IssueSeverityHigh,
			Message:     "Kubernetes workloads in failed state",
			ResourceRef: "",
			Timestamp:   platformv1alpha1.Now(),
		})
	}

	// Issue for out-of-sync ArgoCD apps
	if status.Argo.OutOfSync > 0 {
		issues = append(issues, platformv1alpha1.Issue{
			Severity:    platformv1alpha1.IssueSeverityWarning,
			Message:     "ArgoCD applications out of sync",
			ResourceRef: "",
			Timestamp:   platformv1alpha1.Now(),
		})
	}

	// Limit to top 5
	if len(issues) > 5 {
		issues = issues[:5]
	}

	return issues
}
