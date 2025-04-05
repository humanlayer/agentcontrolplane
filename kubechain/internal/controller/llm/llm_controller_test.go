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

package llm

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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/test/utils"
)

// LLMTestFixture provides helper methods for testing LLM reconciliation
type LLMTestFixture struct {
	namespace     string
	secretName    string
	resourceName  string
	secretKey     string
	apiKeyContent string
	provider      string
	baseURL       string
}

// NewLLMTestFixture creates a new test fixture
func NewLLMTestFixture(provider, resourceName, secretName, secretKey, apiKeyContent, baseURL string) *LLMTestFixture {
	return &LLMTestFixture{
		namespace:     "default",
		resourceName:  resourceName,
		secretName:    secretName,
		secretKey:     secretKey,
		apiKeyContent: apiKeyContent,
		provider:      provider,
		baseURL:       baseURL,
	}
}

// Setup creates a Secret and LLM resource with basic configuration
func (f *LLMTestFixture) Setup(ctx context.Context, k8sClient client.Client) (*kubechainv1alpha1.LLM, *corev1.Secret, error) {
	// Create secret first
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.secretName,
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			f.secretKey: []byte(f.apiKeyContent),
		},
	}
	if err := k8sClient.Create(ctx, secret); err != nil {
		return nil, nil, err
	}

	// Create LLM resource with provider-specific configuration
	llm := &kubechainv1alpha1.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.resourceName,
			Namespace: f.namespace,
		},
		Spec: kubechainv1alpha1.LLMSpec{
			Provider: f.provider,
			APIKeyFrom: &kubechainv1alpha1.APIKeySource{
				SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
					Name: f.secretName,
					Key:  f.secretKey,
				},
			},
			Parameters: kubechainv1alpha1.BaseConfig{
				BaseURL: f.baseURL,
				Model:   "test-model",
			},
		},
	}

	// Add provider-specific configuration
	f.addProviderConfig(llm)

	if err := k8sClient.Create(ctx, llm); err != nil {
		return nil, secret, err
	}

	return llm, secret, nil
}

// addProviderConfig adds provider-specific configuration to the LLM resource
func (f *LLMTestFixture) addProviderConfig(llm *kubechainv1alpha1.LLM) {
	switch f.provider {
	case "openai":
		llm.Spec.OpenAI = &kubechainv1alpha1.OpenAIConfig{
			Organization: "test-org",
			APIType:      "OPEN_AI",
		}
	case "anthropic":
		llm.Spec.Anthropic = &kubechainv1alpha1.AnthropicConfig{
			AnthropicBetaHeader: "test-beta-header",
		}
	case "mistral":
		maxRetries := 3
		timeout := 30
		randomSeed := 42
		llm.Spec.Mistral = &kubechainv1alpha1.MistralConfig{
			MaxRetries: &maxRetries,
			Timeout:    &timeout,
			RandomSeed: &randomSeed,
		}
	case "google":
		llm.Spec.Google = &kubechainv1alpha1.GoogleConfig{
			CloudProject:  "test-project",
			CloudLocation: "us-central1",
		}
	case "vertex":
		llm.Spec.Vertex = &kubechainv1alpha1.VertexConfig{
			CloudProject:  "test-project",
			CloudLocation: "us-central1",
		}
	}
}

// SetupWithoutAPIKey creates an LLM resource without APIKeyFrom
func (f *LLMTestFixture) SetupWithoutAPIKey(ctx context.Context, k8sClient client.Client) (*kubechainv1alpha1.LLM, error) {
	// Create LLM resource without APIKeyFrom
	llm := &kubechainv1alpha1.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.resourceName,
			Namespace: f.namespace,
		},
		Spec: kubechainv1alpha1.LLMSpec{
			Provider: f.provider,
			Parameters: kubechainv1alpha1.BaseConfig{
				BaseURL: f.baseURL,
				Model:   "test-model",
			},
		},
	}

	// Add provider-specific configuration
	f.addProviderConfig(llm)

	if err := k8sClient.Create(ctx, llm); err != nil {
		return nil, err
	}

	return llm, nil
}

// SetupWithoutProviderConfig creates an LLM resource without provider-specific configuration
func (f *LLMTestFixture) SetupWithoutProviderConfig(ctx context.Context, k8sClient client.Client) (*kubechainv1alpha1.LLM, error) {
	// Create secret first
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.secretName,
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			f.secretKey: []byte(f.apiKeyContent),
		},
	}
	if err := k8sClient.Create(ctx, secret); err != nil {
		return nil, err
	}

	// Create LLM resource without provider-specific configuration
	llm := &kubechainv1alpha1.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.resourceName,
			Namespace: f.namespace,
		},
		Spec: kubechainv1alpha1.LLMSpec{
			Provider: f.provider,
			APIKeyFrom: &kubechainv1alpha1.APIKeySource{
				SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
					Name: f.secretName,
					Key:  f.secretKey,
				},
			},
			Parameters: kubechainv1alpha1.BaseConfig{
				BaseURL: f.baseURL,
				Model:   "test-model",
			},
			// ProviderConfig intentionally left with defaults
		},
	}

	if err := k8sClient.Create(ctx, llm); err != nil {
		return nil, err
	}

	return llm, nil
}

// Cleanup deletes the created resources
func (f *LLMTestFixture) Cleanup(ctx context.Context, k8sClient client.Client) error {
	// Delete LLM resource
	llm := &kubechainv1alpha1.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.resourceName,
			Namespace: f.namespace,
		},
	}
	if err := k8sClient.Delete(ctx, llm); client.IgnoreNotFound(err) != nil {
		return err
	}

	// Delete secret if it exists
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.secretName,
			Namespace: f.namespace,
		},
	}
	if err := k8sClient.Delete(ctx, secret); client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
}

// getReconciler creates a reconciler for testing
func getReconciler() (*LLMReconciler, *record.FakeRecorder) {
	eventRecorder := record.NewFakeRecorder(10)
	reconciler := &LLMReconciler{
		Client:   k8sClient,
		Scheme:   k8sClient.Scheme(),
		recorder: eventRecorder,
	}
	return reconciler, eventRecorder
}

var _ = Describe("LLM Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const secretName = "test-secret"
		const secretKey = "api-key"

		ctx := context.Background()
		var mockServer *httptest.Server

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			// Set up a mock server that returns success for API validation
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Always return success for our tests
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				// Return appropriate responses based on the provider being tested
				_, err := w.Write([]byte(`{"id":"test-id","choices":[{"message":{"content":"test"}}]}`))
				if err != nil {
					http.Error(w, "Error writing response", http.StatusInternalServerError)
					return
				}
			}))
		})

		AfterEach(func() {
			mockServer.Close()

			By("Cleaning up test resources")
			cleanup := &LLMTestFixture{
				namespace:    "default",
				resourceName: resourceName,
				secretName:   secretName,
			}
			_ = cleanup.Cleanup(ctx, k8sClient)
		})

		It("should successfully validate OpenAI configuration", func() {
			By("Creating test resources for OpenAI")
			fixture := NewLLMTestFixture(
				"openai",
				resourceName,
				secretName,
				secretKey,
				"test-key",
				mockServer.URL,
			)

			_, _, err := fixture.Setup(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			reconciler, eventRecorder := getReconciler()

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeTrue())
			Expect(updatedLLM.Status.Status).To(Equal("Ready"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("validated successfully"))

			By("Checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})

		XIt("should successfully validate Anthropic configuration - requires more complex mocking", func() {
			By("Creating test resources for Anthropic")
			fixture := NewLLMTestFixture(
				"anthropic",
				resourceName,
				secretName,
				secretKey,
				"test-key",
				mockServer.URL,
			)

			_, _, err := fixture.Setup(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			reconciler, eventRecorder := getReconciler()

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeTrue())
			Expect(updatedLLM.Status.Status).To(Equal("Ready"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("validated successfully"))

			By("Checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})

		XIt("should successfully validate Mistral configuration - requires more complex mocking", func() {
			By("Creating test resources for Mistral")
			fixture := NewLLMTestFixture(
				"mistral",
				resourceName,
				secretName,
				secretKey,
				"test-key",
				mockServer.URL,
			)

			_, _, err := fixture.Setup(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			reconciler, eventRecorder := getReconciler()

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeTrue())
			Expect(updatedLLM.Status.Status).To(Equal("Ready"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("validated successfully"))

			By("Checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})

		XIt("should successfully validate Google configuration - requires more complex mocking", func() {
			By("Creating test resources for Google")
			fixture := NewLLMTestFixture(
				"google",
				resourceName,
				secretName,
				secretKey,
				"test-key",
				mockServer.URL,
			)

			_, _, err := fixture.Setup(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			reconciler, eventRecorder := getReconciler()

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeTrue())
			Expect(updatedLLM.Status.Status).To(Equal("Ready"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("validated successfully"))

			By("Checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})

		XIt("should successfully validate Vertex configuration - requires more complex mocking", func() {
			By("Creating test resources for Vertex")
			fixture := NewLLMTestFixture(
				"vertex",
				resourceName,
				secretName,
				secretKey,
				"test-key",
				mockServer.URL,
			)

			_, _, err := fixture.Setup(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			reconciler, eventRecorder := getReconciler()

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeTrue())
			Expect(updatedLLM.Status.Status).To(Equal("Ready"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("validated successfully"))

			By("Checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})

		It("should fail reconciliation with non-existent secret", func() {
			By("Creating the LLM resource with non-existent secret")
			resource := &kubechainv1alpha1.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.LLMSpec{
					Provider: "openai",
					APIKeyFrom: &kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
							Name: "nonexistent-secret",
							Key:  secretKey,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Reconciling the resource")
			reconciler, eventRecorder := getReconciler()

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeFalse())
			Expect(updatedLLM.Status.Status).To(Equal("Error"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("failed to get secret"))

			By("checking that a failure event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationFailed")
		})

		It("should fail when APIKeyFrom is nil for providers that require it", func() {
			By("Creating LLM resource without APIKeyFrom for OpenAI")
			fixture := NewLLMTestFixture(
				"openai",
				resourceName,
				secretName,
				secretKey,
				"test-key",
				mockServer.URL,
			)

			_, err := fixture.SetupWithoutAPIKey(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the resource")
			reconciler, eventRecorder := getReconciler()

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeFalse())
			Expect(updatedLLM.Status.Status).To(Equal("Error"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("apiKeyFrom is required"))

			By("checking that a failure event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationFailed")
		})

		It("should fail when VertexConfig is missing for Vertex provider", func() {
			By("Creating LLM resource without VertexConfig")
			fixture := NewLLMTestFixture(
				"vertex",
				resourceName,
				secretName,
				secretKey,
				"test-key",
				mockServer.URL,
			)

			// Create the fixture but don't add provider-specific config
			llm, err := fixture.SetupWithoutProviderConfig(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(llm.Spec.Vertex).To(BeNil())

			By("Reconciling the resource")
			reconciler, eventRecorder := getReconciler()

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeFalse())
			Expect(updatedLLM.Status.Status).To(Equal("Error"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("vertex configuration is required"))

			By("checking that a failure event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationFailed")
		})

		// Test common configuration options
		It("should properly apply BaseConfig options", func() {
			By("Creating test resources with comprehensive BaseConfig")

			// Create secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("test-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			// Create LLM with comprehensive BaseConfig
			maxTokens := 100
			topK := 40

			resource := &kubechainv1alpha1.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.LLMSpec{
					Provider: "openai",
					APIKeyFrom: &kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					Parameters: kubechainv1alpha1.BaseConfig{
						BaseURL:          mockServer.URL,
						Model:            "gpt-4",
						Temperature:      "0.7",
						MaxTokens:        &maxTokens,
						TopP:             "0.95",
						TopK:             &topK,
						FrequencyPenalty: "0.5",
						PresencePenalty:  "0.5",
					},
					OpenAI: &kubechainv1alpha1.OpenAIConfig{
						Organization: "test-org",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Reconciling the resource")
			reconciler, eventRecorder := getReconciler()

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeTrue())
			Expect(updatedLLM.Status.Status).To(Equal("Ready"))

			By("checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})
})
