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

var _ = Describe("CrossplaneWatcher", func() {
	var (
		ctx     context.Context
		watcher *CrossplaneWatcher
		scheme  *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(platformv1alpha1.AddToScheme(scheme)).To(Succeed())
	})

	Describe("normalizeResourceState", func() {
		Context("when resource is paused", func() {
			It("should return Paused state", func() {
				resource := &unstructured.Unstructured{}
				resource.SetAnnotations(map[string]string{
					"crossplane.io/paused": "true",
				})

				watcher = &CrossplaneWatcher{}
				state := watcher.normalizeResourceState(resource)
				Expect(state).To(Equal(CrossplaneStatePaused))
			})
		})

		Context("when resource has Ready=True condition", func() {
			It("should return Ready state", func() {
				resource := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"conditions": []interface{}{
								map[string]interface{}{
									"type":   "Ready",
									"status": "True",
								},
							},
						},
					},
				}

				watcher = &CrossplaneWatcher{}
				state := watcher.normalizeResourceState(resource)
				Expect(state).To(Equal(CrossplaneStateReady))
			})
		})

		Context("when resource has Ready=False with ReconcileError", func() {
			It("should return Failed state", func() {
				resource := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"conditions": []interface{}{
								map[string]interface{}{
									"type":   "Ready",
									"status": "False",
									"reason": "ReconcileError",
								},
							},
						},
					},
				}

				watcher = &CrossplaneWatcher{}
				state := watcher.normalizeResourceState(resource)
				Expect(state).To(Equal(CrossplaneStateFailed))
			})
		})

		Context("when resource has Ready=False without error reason", func() {
			It("should return Waiting state", func() {
				resource := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"conditions": []interface{}{
								map[string]interface{}{
									"type":   "Ready",
									"status": "False",
									"reason": "Pending",
								},
							},
						},
					},
				}

				watcher = &CrossplaneWatcher{}
				state := watcher.normalizeResourceState(resource)
				Expect(state).To(Equal(CrossplaneStateWaiting))
			})
		})

		Context("when resource has no conditions", func() {
			It("should return Unknown state", func() {
				resource := &unstructured.Unstructured{
					Object: map[string]interface{}{},
				}

				watcher = &CrossplaneWatcher{}
				state := watcher.normalizeResourceState(resource)
				Expect(state).To(Equal(CrossplaneStateUnknown))
			})
		})
	})

	Describe("ScanCrossplaneResources", func() {
		Context("with no Crossplane resources", func() {
			It("should return zero counts", func() {
				client := fake.NewClientBuilder().WithScheme(scheme).Build()
				watcher = NewCrossplaneWatcher(client)

				tenant := &platformv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-tenant",
					},
					Spec: platformv1alpha1.TenantSpec{
						Namespaces: []string{"test-namespace"},
					},
				}

				counts, err := watcher.ScanCrossplaneResources(ctx, tenant)
				Expect(err).NotTo(HaveOccurred())
				Expect(counts.Total).To(Equal(0))
				Expect(counts.Ready).To(Equal(0))
				Expect(counts.Failed).To(Equal(0))
			})
		})
	})

	Describe("ScanIAMResources", func() {
		Context("with no IAM resources", func() {
			It("should return zero counts", func() {
				client := fake.NewClientBuilder().WithScheme(scheme).Build()
				watcher = NewCrossplaneWatcher(client)

				tenant := &platformv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-tenant",
					},
					Spec: platformv1alpha1.TenantSpec{
						Namespaces: []string{"test-namespace"},
					},
				}

				summary, err := watcher.ScanIAMResources(ctx, tenant)
				Expect(err).NotTo(HaveOccurred())
				Expect(summary.TotalRoles).To(Equal(0))
				Expect(summary.TotalPolicies).To(Equal(0))
			})
		})
	})
})
