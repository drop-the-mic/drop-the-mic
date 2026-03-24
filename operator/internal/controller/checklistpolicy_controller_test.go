/*
Copyright 2026.

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
	"github.com/drop-the-mic/operator/internal/state"
)

var _ = Describe("ChecklistPolicy Controller", func() {
	const (
		policyName      = "test-policy"
		policyNamespace = "default"
		timeout         = time.Second * 10
		interval        = time.Millisecond * 250
	)

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: policyName, Namespace: policyNamespace}

	Context("When creating a new ChecklistPolicy", func() {
		var policy *dtmv1alpha1.ChecklistPolicy

		BeforeEach(func() {
			// Create the LLM secret first
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-llm-secret",
					Namespace: policyNamespace,
				},
				Data: map[string][]byte{
					"api-key": []byte("test-api-key"),
				},
			}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, &corev1.Secret{})
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			}

			policy = &dtmv1alpha1.ChecklistPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      policyName,
					Namespace: policyNamespace,
				},
				Spec: dtmv1alpha1.ChecklistPolicySpec{
					Schedule: dtmv1alpha1.ScheduleConfig{
						FullScan:     "0 */6 * * *",
						FailedRescan: "*/30 * * * *",
					},
					LLM: dtmv1alpha1.LLMConfig{
						Provider: "claude",
						SecretRef: dtmv1alpha1.SecretReference{
							Name: "test-llm-secret",
							Key:  "api-key",
						},
					},
					Checks: []dtmv1alpha1.CheckItem{
						{
							ID:          "pod-health",
							Description: "Verify all pods in default namespace are running",
							Severity:    "critical",
						},
						{
							ID:          "node-ready",
							Description: "Verify all nodes are in Ready state",
							Severity:    "warning",
						},
					},
					TargetNamespaces: []string{"default"},
				},
			}

			err = k8sClient.Get(ctx, namespacedName, &dtmv1alpha1.ChecklistPolicy{})
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, policy)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &dtmv1alpha1.ChecklistPolicy{}
			err := k8sClient.Get(ctx, namespacedName, resource)
			if err == nil {
				// Remove finalizer before deleting
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}
		})

		It("should add a finalizer on first reconcile", func() {
			reconciler := &ChecklistPolicyReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				KubeClient: fake.NewSimpleClientset(),
				Store:      state.NewStore(5),
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			var updated dtmv1alpha1.ChecklistPolicy
			Expect(k8sClient.Get(ctx, namespacedName, &updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(finalizerName))
		})

		It("should create the policy with correct spec", func() {
			var fetched dtmv1alpha1.ChecklistPolicy
			Expect(k8sClient.Get(ctx, namespacedName, &fetched)).To(Succeed())

			Expect(fetched.Spec.LLM.Provider).To(Equal("claude"))
			Expect(fetched.Spec.Schedule.FullScan).To(Equal("0 */6 * * *"))
			Expect(fetched.Spec.Checks).To(HaveLen(2))
			Expect(fetched.Spec.Checks[0].ID).To(Equal("pod-health"))
			Expect(fetched.Spec.Checks[1].ID).To(Equal("node-ready"))
			Expect(fetched.Spec.TargetNamespaces).To(Equal([]string{"default"}))
		})
	})

	Context("When handling the run-now annotation", func() {
		BeforeEach(func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "run-now-secret", Namespace: policyNamespace},
				Data:       map[string][]byte{"api-key": []byte("key")},
			}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, &corev1.Secret{})
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			}

			policy := &dtmv1alpha1.ChecklistPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "run-now-policy",
					Namespace: policyNamespace,
					Annotations: map[string]string{
						runNowAnnotation: "2026-03-24T15:00:00Z",
					},
				},
				Spec: dtmv1alpha1.ChecklistPolicySpec{
					Schedule: dtmv1alpha1.ScheduleConfig{FullScan: "0 0 * * *"},
					LLM: dtmv1alpha1.LLMConfig{
						Provider:  "claude",
						SecretRef: dtmv1alpha1.SecretReference{Name: "run-now-secret"},
					},
					Checks: []dtmv1alpha1.CheckItem{
						{ID: "c1", Description: "test", Severity: "info"},
					},
				},
			}

			nn := types.NamespacedName{Name: "run-now-policy", Namespace: policyNamespace}
			err = k8sClient.Get(ctx, nn, &dtmv1alpha1.ChecklistPolicy{})
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, policy)).To(Succeed())
			}
		})

		AfterEach(func() {
			nn := types.NamespacedName{Name: "run-now-policy", Namespace: policyNamespace}
			resource := &dtmv1alpha1.ChecklistPolicy{}
			err := k8sClient.Get(ctx, nn, resource)
			if err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}
		})

		It("should remove the run-now annotation after reconcile", func() {
			nn := types.NamespacedName{Name: "run-now-policy", Namespace: policyNamespace}

			reconciler := &ChecklistPolicyReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				KubeClient: fake.NewSimpleClientset(),
				Store:      state.NewStore(5),
			}

			// First reconcile adds finalizer
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile sets up scheduler
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			// Third reconcile handles run-now (scheduler exists, annotation present)
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			var updated dtmv1alpha1.ChecklistPolicy
			Expect(k8sClient.Get(ctx, nn, &updated)).To(Succeed())
			_, hasAnnotation := updated.Annotations[runNowAnnotation]
			Expect(hasAnnotation).To(BeFalse(), "run-now annotation should be removed")
		})
	})

	Context("When the policy is deleted", func() {
		It("should handle not-found gracefully", func() {
			reconciler := &ChecklistPolicyReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				KubeClient: fake.NewSimpleClientset(),
				Store:      state.NewStore(5),
			}

			nn := types.NamespacedName{Name: "nonexistent-policy", Namespace: "default"}
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
