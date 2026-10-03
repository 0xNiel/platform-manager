/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
)

var _ = Describe("ArgoWatcher", func() {
	var (
		ctx     context.Context
		watcher *ArgoWatcher
		scheme  *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(platformv1alpha1.AddToScheme(scheme)).To(Succeed())
	})

	Describe("getSyncStatus", func() {
		Context("when app is synced", func() {
			It("should return Synced", func() {
				app := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"sync": map[string]interface{}{
								"status": "Synced",
							},
						},
					},
				}

				watcher = &ArgoWatcher{}
				status := watcher.getSyncStatus(app)
				Expect(status).To(Equal("Synced"))
			})
		})

		Context("when app is out of sync", func() {
			It("should return OutOfSync", func() {
				app := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"sync": map[string]interface{}{
								"status": "OutOfSync",
							},
						},
					},
				}

				watcher = &ArgoWatcher{}
				status := watcher.getSyncStatus(app)
				Expect(status).To(Equal("OutOfSync"))
			})
		})

		Context("when sync status is missing", func() {
			It("should return Unknown", func() {
				app := &unstructured.Unstructured{
					Object: map[string]interface{}{},
				}

				watcher = &ArgoWatcher{}
				status := watcher.getSyncStatus(app)
				Expect(status).To(Equal("Unknown"))
			})
		})
	})

	Describe("getHealthStatus", func() {
		Context("when app is healthy", func() {
			It("should return Healthy", func() {
				app := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"health": map[string]interface{}{
								"status": "Healthy",
							},
						},
					},
				}

				watcher = &ArgoWatcher{}
				status := watcher.getHealthStatus(app)
				Expect(status).To(Equal("Healthy"))
			})
		})

		Context("when app is degraded", func() {
			It("should return Degraded", func() {
				app := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"health": map[string]interface{}{
								"status": "Degraded",
							},
						},
					},
				}

				watcher = &ArgoWatcher{}
				status := watcher.getHealthStatus(app)
				Expect(status).To(Equal("Degraded"))
			})
		})
	})

	Describe("belongsToTenant", func() {
		var tenant *platformv1alpha1.Tenant

		BeforeEach(func() {
			tenant = &platformv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-tenant",
				},
				Spec: platformv1alpha1.TenantSpec{
					DisplayName:  "Test Tenant",
					Namespaces:   []string{"tenant-ns"},
					ArgoProjects: []string{"tenant-project"},
				},
			}
		})

		Context("when app has matching tenant label", func() {
			It("should return true", func() {
				app := &unstructured.Unstructured{}
				app.SetLabels(map[string]string{
					"platform.io/tenant": "test-tenant",
				})

				watcher = &ArgoWatcher{}
				belongs := watcher.belongsToTenant(app, tenant)
				Expect(belongs).To(BeTrue())
			})
		})

		Context("when app project matches tenant project", func() {
			It("should return true", func() {
				app := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": map[string]interface{}{
							"project": "tenant-project",
						},
					},
				}

				watcher = &ArgoWatcher{}
				belongs := watcher.belongsToTenant(app, tenant)
				Expect(belongs).To(BeTrue())
			})
		})

		Context("when app destination namespace matches tenant namespace", func() {
			It("should return true", func() {
				app := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": map[string]interface{}{
							"destination": map[string]interface{}{
								"namespace": "tenant-ns",
							},
						},
					},
				}

				watcher = &ArgoWatcher{}
				belongs := watcher.belongsToTenant(app, tenant)
				Expect(belongs).To(BeTrue())
			})
		})

		Context("when app doesn't belong to tenant", func() {
			It("should return false", func() {
				app := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": map[string]interface{}{
							"project": "other-project",
							"destination": map[string]interface{}{
								"namespace": "other-ns",
							},
						},
					},
				}

				watcher = &ArgoWatcher{}
				belongs := watcher.belongsToTenant(app, tenant)
				Expect(belongs).To(BeFalse())
			})
		})
	})

	Describe("ScanArgoApplications", func() {
		Context("with no ArgoCD applications", func() {
			It("should return zero counts", func() {
				client := fake.NewClientBuilder().WithScheme(scheme).Build()
				watcher = NewArgoWatcher(client)

				tenant := &platformv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-tenant",
					},
					Spec: platformv1alpha1.TenantSpec{
						Namespaces: []string{"test-namespace"},
					},
				}

				summary, err := watcher.ScanArgoApplications(ctx, tenant)
				Expect(err).NotTo(HaveOccurred())
				Expect(summary.TotalApps).To(Equal(0))
				Expect(summary.Synced).To(Equal(0))
				Expect(summary.Healthy).To(Equal(0))
			})
		})
	})

	Describe("extractTenantName", func() {
		Context("with various display names", func() {
			It("should extract first word in lowercase", func() {
				Expect(extractTenantName("Alpha - ML Platform")).To(Equal("alpha"))
				Expect(extractTenantName("Beta - Data Pipeline")).To(Equal("beta"))
				Expect(extractTenantName("Gamma")).To(Equal("gamma"))
				Expect(extractTenantName("PROD-Infrastructure")).To(Equal("prod"))
			})
		})
	})
})
