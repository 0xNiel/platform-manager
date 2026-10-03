/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
)

var _ = Describe("HealthAggregator", func() {
	var (
		ctx        context.Context
		aggregator *HealthAggregator
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(platformv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(appsv1.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
	})

	Describe("normalizeDeploymentState", func() {
		Context("when all replicas are ready", func() {
			It("should return Ready state", func() {
				deployment := &appsv1.Deployment{
					Spec: appsv1.DeploymentSpec{
						Replicas: ptr.To(int32(3)),
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas: 3,
					},
				}

				aggregator = &HealthAggregator{}
				state := aggregator.normalizeDeploymentState(deployment)
				Expect(state).To(Equal(CrossplaneStateReady))
			})
		})

		Context("when some replicas are ready", func() {
			It("should return Waiting state", func() {
				deployment := &appsv1.Deployment{
					Spec: appsv1.DeploymentSpec{
						Replicas: ptr.To(int32(3)),
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas:     1,
						AvailableReplicas: 1,
					},
				}

				aggregator = &HealthAggregator{}
				state := aggregator.normalizeDeploymentState(deployment)
				Expect(state).To(Equal(CrossplaneStateWaiting))
			})
		})

		Context("when no replicas are ready and deployment is not progressing", func() {
			It("should return Failed state", func() {
				deployment := &appsv1.Deployment{
					Spec: appsv1.DeploymentSpec{
						Replicas: ptr.To(int32(3)),
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas: 0,
						Conditions: []appsv1.DeploymentCondition{
							{
								Type:   appsv1.DeploymentProgressing,
								Status: corev1.ConditionFalse,
							},
						},
					},
				}

				aggregator = &HealthAggregator{}
				state := aggregator.normalizeDeploymentState(deployment)
				Expect(state).To(Equal(CrossplaneStateFailed))
			})
		})

		Context("when deployment is paused", func() {
			It("should return Paused state", func() {
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"paused": "true",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: ptr.To(int32(3)),
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas: 2,
					},
				}

				aggregator = &HealthAggregator{}
				state := aggregator.normalizeDeploymentState(deployment)
				Expect(state).To(Equal(CrossplaneStatePaused))
			})
		})
	})

	Describe("normalizePodState", func() {
		Context("when pod is running and all containers are ready", func() {
			It("should return Ready state", func() {
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{Ready: true},
							{Ready: true},
						},
					},
				}

				aggregator = &HealthAggregator{}
				state := aggregator.normalizePodState(pod)
				Expect(state).To(Equal(CrossplaneStateReady))
			})
		})

		Context("when pod is running but some containers are not ready", func() {
			It("should return Waiting state", func() {
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{Ready: true},
							{Ready: false},
						},
					},
				}

				aggregator = &HealthAggregator{}
				state := aggregator.normalizePodState(pod)
				Expect(state).To(Equal(CrossplaneStateWaiting))
			})
		})

		Context("when pod is pending", func() {
			It("should return Waiting state", func() {
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				}

				aggregator = &HealthAggregator{}
				state := aggregator.normalizePodState(pod)
				Expect(state).To(Equal(CrossplaneStateWaiting))
			})
		})

		Context("when pod has failed", func() {
			It("should return Failed state", func() {
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodFailed,
					},
				}

				aggregator = &HealthAggregator{}
				state := aggregator.normalizePodState(pod)
				Expect(state).To(Equal(CrossplaneStateFailed))
			})
		})

		Context("when pod has succeeded", func() {
			It("should return Ready state", func() {
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodSucceeded,
					},
				}

				aggregator = &HealthAggregator{}
				state := aggregator.normalizePodState(pod)
				Expect(state).To(Equal(CrossplaneStateReady))
			})
		})
	})

	Describe("calculateOverallHealth", func() {
		Context("when all resources are ready", func() {
			It("should return Healthy", func() {
				status := &platformv1alpha1.TenantHealthStatus{
					CrossplaneResources: platformv1alpha1.ResourceStateCounts{
						Total: 5,
						Ready: 5,
					},
					KubernetesResources: platformv1alpha1.ResourceStateCounts{
						Total: 3,
						Ready: 3,
					},
				}

				aggregator = &HealthAggregator{}
				health := aggregator.calculateOverallHealth(status)
				Expect(health).To(Equal(platformv1alpha1.HealthLevelHealthy))
			})
		})

		Context("when some resources have failed", func() {
			It("should return Degraded", func() {
				status := &platformv1alpha1.TenantHealthStatus{
					CrossplaneResources: platformv1alpha1.ResourceStateCounts{
						Total:  5,
						Ready:  4,
						Failed: 1,
					},
					KubernetesResources: platformv1alpha1.ResourceStateCounts{
						Total: 3,
						Ready: 3,
					},
				}

				aggregator = &HealthAggregator{}
				health := aggregator.calculateOverallHealth(status)
				Expect(health).To(Equal(platformv1alpha1.HealthLevelDegraded))
			})
		})

		Context("when more than 50% of resources have failed", func() {
			It("should return Critical", func() {
				status := &platformv1alpha1.TenantHealthStatus{
					CrossplaneResources: platformv1alpha1.ResourceStateCounts{
						Total:  10,
						Ready:  2,
						Failed: 8,
					},
				}

				aggregator = &HealthAggregator{}
				health := aggregator.calculateOverallHealth(status)
				Expect(health).To(Equal(platformv1alpha1.HealthLevelCritical))
			})
		})

		Context("when ArgoCD apps are degraded", func() {
			It("should return Critical", func() {
				status := &platformv1alpha1.TenantHealthStatus{
					CrossplaneResources: platformv1alpha1.ResourceStateCounts{
						Total: 5,
						Ready: 5,
					},
					Argo: platformv1alpha1.ArgoSummary{
						TotalApps: 2,
						Degraded:  1,
					},
				}

				aggregator = &HealthAggregator{}
				health := aggregator.calculateOverallHealth(status)
				Expect(health).To(Equal(platformv1alpha1.HealthLevelCritical))
			})
		})

		Context("when there are no resources", func() {
			It("should return Unknown", func() {
				status := &platformv1alpha1.TenantHealthStatus{}

				aggregator = &HealthAggregator{}
				health := aggregator.calculateOverallHealth(status)
				Expect(health).To(Equal(platformv1alpha1.HealthLevelUnknown))
			})
		})
	})

	Describe("AggregateHealth", func() {
		Context("with real Kubernetes resources", func() {
			It("should scan and aggregate health correctly", func() {
				// Create a deployment in the test namespace
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "test-tenant-ns",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: ptr.To(int32(2)),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "test"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": "test"},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{Name: "test", Image: "nginx"},
								},
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas: 2,
					},
				}

				client := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(deployment).
					Build()

				aggregator = NewHealthAggregator(client)

				tenant := &platformv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-tenant",
					},
					Spec: platformv1alpha1.TenantSpec{
						Namespaces: []string{"test-tenant-ns"},
					},
				}

				status, err := aggregator.AggregateHealth(ctx, tenant)
				Expect(err).NotTo(HaveOccurred())
				Expect(status.KubernetesResources.Total).To(Equal(1))
				Expect(status.KubernetesResources.Ready).To(Equal(1))
				Expect(status.OverallHealth).To(Equal(platformv1alpha1.HealthLevelHealthy))
			})
		})
	})
})
