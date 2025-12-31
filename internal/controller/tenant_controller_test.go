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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/platform-manager/platform-manager/api/v1alpha1"
)

var _ = Describe("Tenant Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When reconciling a Tenant resource", func() {
		var (
			ctx        context.Context
			tenantName string
			tenant     *platformv1alpha1.Tenant
			namespace  *corev1.Namespace
			reconciler *TenantReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			tenantName = "test-tenant-" + randString(8)

			// Create a namespace for the tenant
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: tenantName + "-ns",
				},
			}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: namespace.Name}, namespace)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
			}

			// Create the Tenant
			tenant = &platformv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name: tenantName,
				},
				Spec: platformv1alpha1.TenantSpec{
					DisplayName: "Test Tenant",
					Namespaces:  []string{tenantName + "-ns"},
				},
			}
			Expect(k8sClient.Create(ctx, tenant)).To(Succeed())

			// Create reconciler with health aggregator
			reconciler = &TenantReconciler{
				Client:           k8sClient,
				Scheme:           k8sClient.Scheme(),
				HealthAggregator: NewHealthAggregator(k8sClient),
			}
		})

		AfterEach(func() {
			// Cleanup - try to delete if they still exist
			if tenant != nil {
				err := k8sClient.Delete(ctx, tenant)
				if err != nil && !errors.IsNotFound(err) {
					// Tenant might already be deleted in some tests
					GinkgoWriter.Printf("Warning: Failed to delete tenant: %v\n", err)
				}
			}
			if namespace != nil {
				// Delete namespace and wait briefly
				err := k8sClient.Delete(ctx, namespace)
				if err != nil && !errors.IsNotFound(err) {
					GinkgoWriter.Printf("Warning: Failed to delete namespace: %v\n", err)
				}
				// Wait a moment for namespace to start deleting
				time.Sleep(100 * time.Millisecond)
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: tenantName,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the Tenant status is updated")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: tenantName}, tenant)
				if err != nil {
					return false
				}
				return tenant.Status.Phase == platformv1alpha1.TenantPhaseActive
			}, timeout, interval).Should(BeTrue())
		})

		It("should create a TenantHealth resource", func() {
			By("Reconciling the Tenant")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: tenantName,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that TenantHealth was created")
			tenantHealth := &platformv1alpha1.TenantHealth{}
			healthName := tenantName + "-health"
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: healthName}, tenantHealth)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Verifying TenantHealth has correct spec")
			Expect(tenantHealth.Spec.TenantRef).To(Equal(tenantName))

			By("Verifying TenantHealth has been populated with status")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: healthName}, tenantHealth)
				if err != nil {
					return false
				}
				return tenantHealth.Status.OverallHealth != ""
			}, timeout, interval).Should(BeTrue())
		})

		It("should update TenantHealth status with aggregated data", func() {
			By("Reconciling the Tenant")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: tenantName,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Getting the TenantHealth")
			tenantHealth := &platformv1alpha1.TenantHealth{}
			healthName := tenantName + "-health"
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: healthName}, tenantHealth)
			}, timeout, interval).Should(Succeed())

			By("Verifying health status fields are present")
			Expect(tenantHealth.Status.OverallHealth).NotTo(BeEmpty())
			Expect(tenantHealth.Status.LastUpdated).NotTo(BeNil())
		})

		It("should add finalizer to Tenant", func() {
			By("Reconciling the Tenant")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: tenantName,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that finalizer was added")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: tenantName}, tenant)
				if err != nil {
					return false
				}
				return len(tenant.Finalizers) > 0
			}, timeout, interval).Should(BeTrue())
		})

		It("should cleanup TenantHealth when Tenant is deleted", func() {
			By("Reconciling the Tenant first")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: tenantName,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying TenantHealth exists")
			tenantHealth := &platformv1alpha1.TenantHealth{}
			healthName := tenantName + "-health"
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: healthName}, tenantHealth)
			}, timeout, interval).Should(Succeed())

			By("Deleting the Tenant")
			Expect(k8sClient.Delete(ctx, tenant)).To(Succeed())

			By("Reconciling the deletion")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: tenantName,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying TenantHealth is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: healthName}, tenantHealth)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			// Set tenant to nil so AfterEach doesn't try to delete again
			tenant = nil
		})
	})
})

// randString generates a random string of given length
func randString(n int) string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%1000000000)[:n]
}
