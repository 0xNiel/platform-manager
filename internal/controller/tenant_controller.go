/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/platform-manager/platform-manager/api/v1alpha1"
)

const (
	tenantFinalizer = "platform.platform.io/finalizer"

	// Condition types
	ConditionTypeReady       = "Ready"
	ConditionTypeHealthReady = "HealthReady"

	// Requeue intervals
	defaultRequeueInterval = 30 * time.Second
	errorRequeueInterval   = 10 * time.Second
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.platform.io,resources=tenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.platform.io,resources=tenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.platform.io,resources=tenants/finalizers,verbs=update
// +kubebuilder:rbac:groups=platform.platform.io,resources=tenanthealth,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.platform.io,resources=tenanthealth/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Tenant instance
	tenant := &platformv1alpha1.Tenant{}
	if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
		if errors.IsNotFound(err) {
			// Tenant was deleted, nothing to do
			log.Info("Tenant resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Tenant")
		return ctrl.Result{}, err
	}

	// Handle finalizer for cleanup
	if tenant.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add finalizer if not present
		if !controllerutil.ContainsFinalizer(tenant, tenantFinalizer) {
			controllerutil.AddFinalizer(tenant, tenantFinalizer)
			if err := r.Update(ctx, tenant); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Tenant is being deleted
		if controllerutil.ContainsFinalizer(tenant, tenantFinalizer) {
			// Perform cleanup - delete associated TenantHealth
			if err := r.cleanupTenantHealth(ctx, tenant); err != nil {
				log.Error(err, "Failed to cleanup TenantHealth")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(tenant, tenantFinalizer)
			if err := r.Update(ctx, tenant); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Update tenant phase to Active if not set
	if tenant.Status.Phase == "" {
		tenant.Status.Phase = platformv1alpha1.TenantPhaseActive
	}

	// Count namespaces belonging to this tenant
	namespaceCount, err := r.countTenantNamespaces(ctx, tenant)
	if err != nil {
		log.Error(err, "Failed to count tenant namespaces")
		// Don't fail the reconciliation, just log
	}
	tenant.Status.NamespaceCount = namespaceCount

	// Ensure TenantHealth exists for this tenant
	tenantHealth, err := r.ensureTenantHealth(ctx, tenant)
	if err != nil {
		log.Error(err, "Failed to ensure TenantHealth")
		meta.SetStatusCondition(&tenant.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeHealthReady,
			Status:             metav1.ConditionFalse,
			Reason:             "HealthCreationFailed",
			Message:            fmt.Sprintf("Failed to create TenantHealth: %v", err),
			LastTransitionTime: metav1.Now(),
		})
	} else {
		tenant.Status.HealthRef = tenantHealth.Name
		meta.SetStatusCondition(&tenant.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeHealthReady,
			Status:             metav1.ConditionTrue,
			Reason:             "HealthReady",
			Message:            "TenantHealth resource is ready",
			LastTransitionTime: metav1.Now(),
		})
	}

	// Set Ready condition
	meta.SetStatusCondition(&tenant.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "ReconcileSuccess",
		Message:            "Tenant reconciled successfully",
		LastTransitionTime: metav1.Now(),
	})

	// Update last reconciled time
	now := metav1.Now()
	tenant.Status.LastReconciled = &now
	tenant.Status.ObservedGeneration = tenant.Generation

	// Update status
	if err := r.Status().Update(ctx, tenant); err != nil {
		log.Error(err, "Failed to update Tenant status")
		return ctrl.Result{RequeueAfter: errorRequeueInterval}, err
	}

	log.Info("Successfully reconciled Tenant",
		"tenant", tenant.Name,
		"namespaces", namespaceCount,
		"phase", tenant.Status.Phase)

	// Requeue to periodically refresh health data
	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
}

// countTenantNamespaces counts the number of namespaces belonging to this tenant
func (r *TenantReconciler) countTenantNamespaces(ctx context.Context, tenant *platformv1alpha1.Tenant) (int, error) {
	count := 0

	// Count explicitly listed namespaces
	for _, ns := range tenant.Spec.Namespaces {
		namespace := &corev1.Namespace{}
		if err := r.Get(ctx, client.ObjectKey{Name: ns}, namespace); err == nil {
			count++
		}
	}

	// If label selector is specified, also count matching namespaces
	if tenant.Spec.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(tenant.Spec.LabelSelector)
		if err != nil {
			return count, err
		}

		namespaceList := &corev1.NamespaceList{}
		if err := r.List(ctx, namespaceList, &client.ListOptions{
			LabelSelector: selector,
		}); err != nil {
			return count, err
		}

		// Count unique namespaces (avoid double counting)
		existingNs := make(map[string]bool)
		for _, ns := range tenant.Spec.Namespaces {
			existingNs[ns] = true
		}
		for _, ns := range namespaceList.Items {
			if !existingNs[ns.Name] {
				count++
			}
		}
	}

	return count, nil
}

// ensureTenantHealth creates or updates the TenantHealth resource for this tenant
func (r *TenantReconciler) ensureTenantHealth(ctx context.Context, tenant *platformv1alpha1.Tenant) (*platformv1alpha1.TenantHealth, error) {
	log := logf.FromContext(ctx)

	healthName := fmt.Sprintf("%s-health", tenant.Name)
	tenantHealth := &platformv1alpha1.TenantHealth{}

	err := r.Get(ctx, client.ObjectKey{Name: healthName}, tenantHealth)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new TenantHealth
			tenantHealth = &platformv1alpha1.TenantHealth{
				ObjectMeta: metav1.ObjectMeta{
					Name: healthName,
					Labels: labels.Set{
						"platform.io/tenant": tenant.Name,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         platformv1alpha1.GroupVersion.String(),
							Kind:               "Tenant",
							Name:               tenant.Name,
							UID:                tenant.UID,
							Controller:         ptrBool(true),
							BlockOwnerDeletion: ptrBool(true),
						},
					},
				},
				Spec: platformv1alpha1.TenantHealthSpec{
					TenantRef: tenant.Name,
				},
			}

			if err := r.Create(ctx, tenantHealth); err != nil {
				return nil, err
			}
			log.Info("Created TenantHealth", "name", healthName)
		} else {
			return nil, err
		}
	}

	// Initialize status if empty
	if tenantHealth.Status.OverallHealth == "" {
		tenantHealth.Status.OverallHealth = platformv1alpha1.HealthLevelUnknown
		now := metav1.Now()
		tenantHealth.Status.LastUpdated = &now

		if err := r.Status().Update(ctx, tenantHealth); err != nil {
			log.Error(err, "Failed to update TenantHealth status")
			// Don't fail, we'll update it on the next reconcile
		}
	}

	return tenantHealth, nil
}

// cleanupTenantHealth deletes the TenantHealth associated with the tenant
func (r *TenantReconciler) cleanupTenantHealth(ctx context.Context, tenant *platformv1alpha1.Tenant) error {
	log := logf.FromContext(ctx)

	healthName := fmt.Sprintf("%s-health", tenant.Name)
	tenantHealth := &platformv1alpha1.TenantHealth{}

	err := r.Get(ctx, client.ObjectKey{Name: healthName}, tenantHealth)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Already deleted
		}
		return err
	}

	if err := r.Delete(ctx, tenantHealth); err != nil {
		return err
	}

	log.Info("Deleted TenantHealth", "name", healthName)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Tenant{}).
		Owns(&platformv1alpha1.TenantHealth{}).
		Named("tenant").
		Complete(r)
}

// ptrBool returns a pointer to a bool
func ptrBool(b bool) *bool {
	return &b
}
