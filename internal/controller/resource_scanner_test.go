package controller

import (
	"context"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
)

var _ = Describe("ResourceScanner", func() {
	var (
		scanner   *ResourceScanner
		tenant    *platformv1alpha1.Tenant
		namespace *corev1.Namespace
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		scanner = NewResourceScanner(k8sClient)

		// Create a unique namespace for this test
		namespaceName := "test-scanner-" + randStringRunes(5)
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create a test tenant
		tenant = &platformv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-scanner-tenant",
			},
			Spec: platformv1alpha1.TenantSpec{
				DisplayName: "Test Scanner Tenant",
				Namespaces:  []string{namespaceName},
			},
		}
		Expect(k8sClient.Create(ctx, tenant)).To(Succeed())

		DeferCleanup(func() {
			Expect(k8sClient.Delete(ctx, tenant)).To(Succeed())
			Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		})
	})

	Context("ScanKubernetesResources", func() {
		It("should create ResourceSummary for Deployments", func() {
			// Create a test deployment
			deploy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: namespace.Name,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptrInt32(2),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "nginx:alpine",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, deploy)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, deploy)).To(Succeed())
			})

			// Scan only Kubernetes resources (not full tenant scan to avoid Crossplane/Argo errors)
			Expect(scanner.scanKubernetesResources(ctx, tenant)).To(Succeed())

			// Verify ResourceSummary was created
			summaryName := generateResourceSummaryName(tenant.Name, deploy)
			summary := &platformv1alpha1.ResourceSummary{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: summaryName}, summary)
			}).Should(Succeed())

			Expect(summary.Spec.Kind).To(Equal("Deployment"))
			Expect(summary.Spec.Name).To(Equal("test-deployment"))
			Expect(summary.Spec.Namespace).To(Equal(namespace.Name))
			Expect(summary.Spec.TenantRef).To(Equal(tenant.Name))
			Expect(summary.Spec.Category).To(Equal(platformv1alpha1.ResourceCategoryKubernetes))
		})

		It("should create ResourceSummary for Jobs", func() {
			// Create a test job
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: namespace.Name,
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "nginx:alpine",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, job)).To(Succeed())
			})

			// Scan only Kubernetes resources
			Expect(scanner.scanKubernetesResources(ctx, tenant)).To(Succeed())

			// Verify ResourceSummary was created
			summaryName := generateResourceSummaryName(tenant.Name, job)
			summary := &platformv1alpha1.ResourceSummary{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: summaryName}, summary)
			}).Should(Succeed())

			Expect(summary.Spec.Kind).To(Equal("Job"))
			Expect(summary.Spec.Name).To(Equal("test-job"))
		})
	})

	// Skip ArgoCD tests as CRDs are not available in test environment
	Context("ScanArgoResources", func() {
		// These tests require ArgoCD CRDs which are not installed in the test environment
		// In production, the scanner handles missing CRDs gracefully
		PIt("should create ResourceSummary for ArgoCD Applications")
	})

	Context("NormalizeResourceStatus", func() {
		It("should normalize Deployment status correctly", func() {
			deploy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-deploy",
					Namespace:         namespace.Name,
					CreationTimestamp: metav1.Now(),
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptrInt32(2),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          2,
					AvailableReplicas: 2,
					UpdatedReplicas:   2,
				},
			}

			state, message, conditions := scanner.normalizeKubernetesStatus(deploy)
			Expect(state).To(Equal(platformv1alpha1.ResourceStateReady))
			Expect(message).To(ContainSubstring("2/2 replicas ready"))
			Expect(conditions).NotTo(BeNil())
		})

		It("should detect failed Deployment", func() {
			deploy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-deploy",
					Namespace:         namespace.Name,
					CreationTimestamp: metav1.Now(),
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptrInt32(2),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          1,
					AvailableReplicas: 0,
					UpdatedReplicas:   1,
					Conditions: []appsv1.DeploymentCondition{
						{
							Type:    appsv1.DeploymentReplicaFailure,
							Status:  corev1.ConditionTrue,
							Reason:  "FailedCreate",
							Message: "Failed to create pod",
						},
					},
				},
			}

			state, _, _ := scanner.normalizeKubernetesStatus(deploy)
			Expect(state).To(Equal(platformv1alpha1.ResourceStateFailed))
		})

		It("should detect paused Deployment", func() {
			deploy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-deploy",
					Namespace:         namespace.Name,
					CreationTimestamp: metav1.Now(),
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptrInt32(2),
					Paused:   true,
				},
			}

			state, message, _ := scanner.normalizeKubernetesStatus(deploy)
			Expect(state).To(Equal(platformv1alpha1.ResourceStatePaused))
			Expect(message).To(ContainSubstring("paused"))
		})
	})
})

// Helper function
func ptrInt32(i int32) *int32 {
	return &i
}

// randStringRunes generates a random string of length n
func randStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
