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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

var _ = Describe("LLM Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		llm := &kubechainv1alpha1.LLM{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind LLM")
			err := k8sClient.Get(ctx, typeNamespacedName, llm)
			if err != nil && errors.IsNotFound(err) {
				resource := &kubechainv1alpha1.LLM{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: kubechainv1alpha1.LLMSpec{
						Provider: "openai",
						APIKeyFrom: kubechainv1alpha1.APIKeySource{
							SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
								Name: "test-secret",
								Key:  "api-key",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &kubechainv1alpha1.LLM{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance LLM")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &LLMReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeFalse())
			Expect(updatedLLM.Status.Message).To(ContainSubstring("failed to get secret"))
		})
	})
})
