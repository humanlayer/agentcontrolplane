package toolcall

import (
	"context"
	"fmt"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayer"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Unit tests are integrated with the existing test suite in toolcall_controller_test.go

type mockHumanLayerClient struct {
	setAPIKeyCalled bool
	setRunIDCalled  bool
	setCallIDCalled bool
	setEmailCalled  bool
	requestCalled   bool
	lastAPIKey      string
	lastRunID       string
	lastCallID      string
	lastMessage     string
	shouldFail      bool
	callIDToReturn  string
}

func (m *mockHumanLayerClient) SetSlackConfig(slackConfig *acp.SlackChannelConfig) {}
func (m *mockHumanLayerClient) SetEmailConfig(emailConfig *acp.EmailChannelConfig) {
	m.setEmailCalled = true
}
func (m *mockHumanLayerClient) SetFunctionCallSpec(functionName string, args map[string]interface{}) {
}
func (m *mockHumanLayerClient) SetCallID(callID string) {
	m.setCallIDCalled = true
	m.lastCallID = callID
}
func (m *mockHumanLayerClient) SetRunID(runID string) {
	m.setRunIDCalled = true
	m.lastRunID = runID
}
func (m *mockHumanLayerClient) SetAPIKey(apiKey string) {
	m.setAPIKeyCalled = true
	m.lastAPIKey = apiKey
}
func (m *mockHumanLayerClient) SetThreadID(threadID string) {
	// Mock implementation
}
func (m *mockHumanLayerClient) RequestApproval(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
	return nil, 200, nil
}
func (m *mockHumanLayerClient) RequestHumanContact(ctx context.Context, userMsg string) (*humanlayerapi.HumanContactOutput, int, error) {
	m.requestCalled = true
	m.lastMessage = userMsg
	if m.shouldFail {
		return nil, 400, fmt.Errorf("bad request")
	}
	output := humanlayerapi.NewHumanContactOutput("test-run", m.callIDToReturn, *humanlayerapi.NewHumanContactSpecOutput(userMsg))
	return output, 200, nil
}
func (m *mockHumanLayerClient) GetFunctionCallStatus(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
	return nil, 200, nil
}
func (m *mockHumanLayerClient) GetHumanContactStatus(ctx context.Context) (*humanlayerapi.HumanContactOutput, int, error) {
	return nil, 200, nil
}

type mockHumanLayerFactory struct {
	client *mockHumanLayerClient
}

func (m *mockHumanLayerFactory) NewHumanLayerClient() humanlayer.HumanLayerClientWrapper {
	return m.client
}

var _ = Describe("ToolExecutor Unit Tests", func() {
	var (
		ctx         context.Context
		fakeClient  client.Client
		executor    *ToolExecutor
		mockHL      *mockHumanLayerClient
		mockFactory *mockHumanLayerFactory
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(acp.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		// Create test resources
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"HUMANLAYER_API_KEY": []byte("test-api-key"),
			},
		}

		contactChannel := &acp.ContactChannel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-channel",
				Namespace: "default",
			},
			Spec: acp.ContactChannelSpec{
				Type: acp.ContactChannelTypeEmail,
				Email: &acp.EmailChannelConfig{
					Address:          "test@example.com",
					ContextAboutUser: "Test user",
				},
				APIKeyFrom: acp.APIKeySource{
					SecretKeyRef: acp.SecretKeyRef{
						Name: "test-secret",
						Key:  "HUMANLAYER_API_KEY",
					},
				},
			},
		}

		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret, contactChannel).
			Build()

		mockHL = &mockHumanLayerClient{
			callIDToReturn: "test-call-id-123",
		}
		mockFactory = &mockHumanLayerFactory{client: mockHL}
		executor = NewToolExecutor(fakeClient, nil, mockFactory)
	})

	Describe("executeHumanContact", func() {
		var toolCall *acp.ToolCall

		BeforeEach(func() {
			toolCall = &acp.ToolCall{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-toolcall",
					Namespace: "default",
				},
				Spec: acp.ToolCallSpec{
					ToolCallID: "test-call-id",
					ToolRef: acp.LocalObjectReference{
						Name: "test-channel__human_contact_email",
					},
					Arguments: `{"message": "What is the fastest animal?"}`,
					ToolType:  acp.ToolTypeHumanContact,
				},
			}
		})

		It("should extract message from arguments and set all required fields", func() {
			args := map[string]interface{}{
				"message": "What is the fastest animal?",
			}

			result, err := executor.executeHumanContact(ctx, toolCall, args)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("Human contact requested, call ID: test-call-id-123"))

			// Verify all required methods were called
			Expect(mockHL.setAPIKeyCalled).To(BeTrue())
			Expect(mockHL.setRunIDCalled).To(BeTrue())
			Expect(mockHL.setCallIDCalled).To(BeTrue())
			Expect(mockHL.setEmailCalled).To(BeTrue())
			Expect(mockHL.requestCalled).To(BeTrue())

			// Verify correct values were passed
			Expect(mockHL.lastAPIKey).To(Equal("test-api-key"))
			Expect(mockHL.lastRunID).To(Equal("test-toolcall"))
			Expect(mockHL.lastCallID).To(Equal("test-call-id"))
			Expect(mockHL.lastMessage).To(Equal("What is the fastest animal?"))
		})

		It("should fail when message argument is missing", func() {
			args := map[string]interface{}{
				"url": "https://example.com", // Wrong argument
			}

			result, err := executor.executeHumanContact(ctx, toolCall, args)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing or invalid 'message' argument"))
			Expect(result).To(BeEmpty())
			Expect(mockHL.requestCalled).To(BeFalse())
		})

		It("should fail when message argument is not a string", func() {
			args := map[string]interface{}{
				"message": 12345, // Wrong type
			}

			result, err := executor.executeHumanContact(ctx, toolCall, args)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing or invalid 'message' argument"))
			Expect(result).To(BeEmpty())
			Expect(mockHL.requestCalled).To(BeFalse())
		})

		It("should propagate HumanLayer API errors", func() {
			mockHL.shouldFail = true
			args := map[string]interface{}{
				"message": "What is the fastest animal?",
			}

			result, err := executor.executeHumanContact(ctx, toolCall, args)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("human contact request failed"))
			Expect(err.Error()).To(ContainSubstring("bad request"))
			Expect(result).To(BeEmpty())
			Expect(mockHL.requestCalled).To(BeTrue())
		})

		It("should fail when contact channel is not found", func() {
			toolCall.Spec.ToolRef.Name = "nonexistent-channel__human_contact_email"
			args := map[string]interface{}{
				"message": "What is the fastest animal?",
			}

			result, err := executor.executeHumanContact(ctx, toolCall, args)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get contact channel"))
			Expect(result).To(BeEmpty())
			Expect(mockHL.requestCalled).To(BeFalse())
		})

		It("should handle channel extraction correctly", func() {
			// Test with different channel name formats
			testCases := []struct {
				toolRefName     string
				expectedChannel string
			}{
				{"test-channel__human_contact_email", "test-channel"},
				{"my-awesome-channel__some_tool", "my-awesome-channel"},
				{"simple", "simple"}, // No __ separator
			}

			for _, tc := range testCases {
				toolCall.Spec.ToolRef.Name = tc.toolRefName

				// Only test the first case since only test-channel exists in our fake client
				if tc.expectedChannel == "test-channel" {
					args := map[string]interface{}{
						"message": "Test message",
					}

					result, err := executor.executeHumanContact(ctx, toolCall, args)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(ContainSubstring("Human contact requested"))
				}
			}
		})
	})

	Describe("getAPIKey", func() {
		It("should retrieve API key from secret", func() {
			contactChannel := &acp.ContactChannel{
				Spec: acp.ContactChannelSpec{
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: "test-secret",
							Key:  "HUMANLAYER_API_KEY",
						},
					},
				},
			}

			apiKey, err := executor.getAPIKey(ctx, contactChannel, "default")

			Expect(err).NotTo(HaveOccurred())
			Expect(apiKey).To(Equal("test-api-key"))
		})

		It("should fail when secret is not found", func() {
			contactChannel := &acp.ContactChannel{
				Spec: acp.ContactChannelSpec{
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: "nonexistent-secret",
							Key:  "HUMANLAYER_API_KEY",
						},
					},
				},
			}

			apiKey, err := executor.getAPIKey(ctx, contactChannel, "default")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get API key secret"))
			Expect(apiKey).To(BeEmpty())
		})

		It("should fail when API key is not found in secret", func() {
			contactChannel := &acp.ContactChannel{
				Spec: acp.ContactChannelSpec{
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: "test-secret",
							Key:  "NONEXISTENT_KEY",
						},
					},
				},
			}

			apiKey, err := executor.getAPIKey(ctx, contactChannel, "default")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API key not found in secret"))
			Expect(apiKey).To(BeEmpty())
		})
	})
})
