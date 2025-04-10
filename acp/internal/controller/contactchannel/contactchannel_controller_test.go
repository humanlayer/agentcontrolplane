/*
Copyright 2025 the Kubechain Authors.

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

package contactchannel

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

var _ = Describe("ContactChannel Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-contactchannel"
		const secretName = "test-contactchannel-secret"
		const secretKey = "api-key"

		ctx := context.Background()
		var mockHumanLayerServer *httptest.Server

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			// Set up mock HumanLayer server
			mockHumanLayerServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") == "Bearer valid-humanlayer-key" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					// Return mock project info
					_, _ = w.Write([]byte(`{"id": "test-project-id", "name": "Test Project"}`))
				} else {
					w.WriteHeader(http.StatusUnauthorized)
				}
			}))

			// Override constants for testing
			humanLayerAPIURL = mockHumanLayerServer.URL
		})

		AfterEach(func() {
			mockHumanLayerServer.Close()

			By("Cleanup the test secret")
			secret := &corev1.Secret{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, secret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}

			By("Cleanup the ContactChannel resource")
			resource := &acp.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should successfully validate a Slack channel with valid config", func() {
			By("Creating a secret with valid API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("valid-humanlayer-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a ContactChannel resource for Slack")
			channel := &acp.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: acp.ContactChannelSpec{
					Type: "slack",
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					Slack: &acp.SlackChannelConfig{
						ChannelOrUserID:           "C12345678",
						ContextAboutChannelOrUser: "A test channel",
					},
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &acp.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeTrue())
			Expect(updatedChannel.Status.Status).To(Equal(statusReady))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("validated successfully"))
		})

		It("should successfully validate an Email channel with valid config", func() {
			By("Creating a secret with valid API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("valid-humanlayer-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a ContactChannel resource for Email")
			channel := &acp.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: acp.ContactChannelSpec{
					Type: "email",
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					Email: &acp.EmailChannelConfig{
						Address:          "test@example.com",
						ContextAboutUser: "Test user",
						Subject:          "Test notification",
					},
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &acp.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeTrue())
			Expect(updatedChannel.Status.Status).To(Equal(statusReady))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("validated successfully"))
		})

		It("should fail validation with invalid configuration", func() {
			By("Creating a secret with valid API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("valid-humanlayer-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a ContactChannel resource with missing config")
			channel := &acp.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: acp.ContactChannelSpec{
					Type: "slack",
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					// Missing SlackConfig
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &acp.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeFalse())
			Expect(updatedChannel.Status.Status).To(Equal(statusError))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("slackConfig"))
		})

		It("should fail validation with invalid API key", func() {
			By("Creating a secret with invalid API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("invalid-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a ContactChannel resource")
			channel := &acp.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: acp.ContactChannelSpec{
					Type: "slack",
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					Slack: &acp.SlackChannelConfig{
						ChannelOrUserID: "C12345678",
					},
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &acp.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeFalse())
			Expect(updatedChannel.Status.Status).To(Equal(statusError))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("invalid"))
		})

		It("should fail validation with non-existent secret", func() {
			By("Creating a ContactChannel resource with non-existent secret")
			channel := &acp.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: acp.ContactChannelSpec{
					Type: "slack",
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: "nonexistent-secret",
							Key:  secretKey,
						},
					},
					Slack: &acp.SlackChannelConfig{
						ChannelOrUserID: "C12345678",
					},
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &acp.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeFalse())
			Expect(updatedChannel.Status.Status).To(Equal(statusError))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("failed to get secret"))
		})

		It("should fail validation with invalid email address", func() {
			By("Creating a secret with valid API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("valid-humanlayer-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a ContactChannel resource with invalid email")
			channel := &acp.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: acp.ContactChannelSpec{
					Type: "email",
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					Email: &acp.EmailChannelConfig{
						// Use an email that passes regex pattern but fails RFC5322 validation
						Address: "test@example..com",
					},
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &acp.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeFalse())
			Expect(updatedChannel.Status.Status).To(Equal(statusError))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("invalid email"))
		})
	})
})
