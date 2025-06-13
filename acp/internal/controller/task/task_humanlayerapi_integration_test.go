package task

import (
	"context"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayer"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/trace/noop"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type MockLLMClient struct {
	SendRequestResponse *acp.Message
	SendRequestError    error
}

func (m *MockLLMClient) SendRequest(ctx context.Context, messages []acp.Message, tools []llmclient.Tool) (*acp.Message, error) {
	return m.SendRequestResponse, m.SendRequestError
}

// Mock HumanLayer Client
type MockHumanLayerClientFactory struct {
	client *MockHumanLayerClient
}

type MockHumanLayerClient struct {
	apiKey      string
	runID       string
	callID      string
	baseURL     string
	requests    []string
	responses   []*humanlayerapi.HumanContactOutput
	statusCodes []int
	errors      []error
	callCount   int
}

func (f *MockHumanLayerClientFactory) NewHumanLayerClient() humanlayer.HumanLayerClientWrapper {
	return f.client
}

func (f *MockHumanLayerClientFactory) NewClient(baseURL string) (humanlayer.HumanLayerClientWrapper, error) {
	f.client.baseURL = baseURL
	return f.client, nil
}

func (c *MockHumanLayerClient) SetSlackConfig(slackConfig *acp.SlackChannelConfig) {}
func (c *MockHumanLayerClient) SetEmailConfig(emailConfig *acp.EmailChannelConfig) {}
func (c *MockHumanLayerClient) SetFunctionCallSpec(functionName string, args map[string]interface{}) {
}

func (c *MockHumanLayerClient) SetCallID(callID string) {
	c.callID = callID
}

func (c *MockHumanLayerClient) SetRunID(runID string) {
	c.runID = runID
}

func (c *MockHumanLayerClient) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

func (c *MockHumanLayerClient) SetThreadID(threadID string) {
	// Mock implementation
}

func (c *MockHumanLayerClient) RequestApproval(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
	return nil, 200, nil
}

func (c *MockHumanLayerClient) RequestHumanContact(ctx context.Context, userMsg string) (*humanlayerapi.HumanContactOutput, int, error) {
	c.requests = append(c.requests, userMsg)

	if c.callCount < len(c.responses) {
		response := c.responses[c.callCount]
		statusCode := c.statusCodes[c.callCount]
		err := c.errors[c.callCount]
		c.callCount++
		return response, statusCode, err
	}

	// Default response
	output := humanlayerapi.NewHumanContactOutput("test-run", c.callID, *humanlayerapi.NewHumanContactSpecOutput("Test result"))
	return output, 200, nil
}

func (c *MockHumanLayerClient) GetFunctionCallStatus(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
	return nil, 200, nil
}

func (c *MockHumanLayerClient) GetHumanContactStatus(ctx context.Context) (*humanlayerapi.HumanContactOutput, int, error) {
	return nil, 200, nil
}

func reconcilerWithMockFactories(createFunc func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error), humanLayerFactory HumanLayerClientFactory) (*TaskReconciler, *record.FakeRecorder) {
	recorder := record.NewFakeRecorder(10)
	tracer := noop.NewTracerProvider().Tracer("test")
	return &TaskReconciler{
		Client:                  k8sClient,
		Scheme:                  k8sClient.Scheme(),
		recorder:                recorder,
		llmClientFactory:        &mockLLMClientFactory{createFunc: createFunc},
		humanLayerClientFactory: humanLayerFactory,
		toolAdapter:             &defaultToolAdapter{},
		Tracer:                  tracer,
	}, recorder
}

var _ = Describe("Task Controller with HumanLayer API", func() {
	Context("using ChannelTokenFrom with secret reference", func() {
		var (
			mockLLMClient         *MockLLMClient
			mockHumanLayerClient  *MockHumanLayerClient
			mockHumanLayerFactory *MockHumanLayerClientFactory
		)

		BeforeEach(func() {
			_, _, _, teardown := setupSuiteObjects(ctx)
			DeferCleanup(teardown)

			mockLLMClient = &MockLLMClient{
				SendRequestResponse: &acp.Message{Content: "Test result"},
			}

			// Create a very simple mock HumanLayer client that just stores the API key
			mockHumanLayerClient = &MockHumanLayerClient{
				responses:   []*humanlayerapi.HumanContactOutput{humanlayerapi.NewHumanContactOutput("test-run", "test-call", *humanlayerapi.NewHumanContactSpecOutput("Test result"))},
				statusCodes: []int{200},
				errors:      []error{nil},
			}
			mockHumanLayerFactory = &MockHumanLayerClientFactory{
				client: mockHumanLayerClient,
			}
		})

		It("retrieves channel token from secret and uses it as API key", func() {
			// Create a secret containing the token
			secret := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{Name: "test-channel-token", Namespace: "default"},
				Data: map[string][]byte{
					"token": []byte("hl_testtoken"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			DeferCleanup(func() { Expect(k8sClient.Delete(ctx, secret)).To(Succeed()) })

			task := &acp.Task{
				ObjectMeta: v1.ObjectMeta{Name: "test-task", Namespace: "default"},
				Spec: acp.TaskSpec{
					AgentRef:    acp.LocalObjectReference{Name: testAgent.Name},
					UserMessage: "Test message",
					BaseURL:     "https://api.example.com",
					ChannelTokenFrom: &acp.SecretKeyRef{
						Name: "test-channel-token",
						Key:  "token",
					},
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())
			DeferCleanup(func() { Expect(k8sClient.Delete(ctx, task)).To(Succeed()) })

			mockLLMClientFn := func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
				return mockLLMClient, nil
			}
			reconciler, _ := reconcilerWithMockFactories(mockLLMClientFn, mockHumanLayerFactory)

			for i := 0; i < 3; i++ {
				result, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: task.Name, Namespace: "default"},
				})
				Expect(err).NotTo(HaveOccurred())
				if i < 2 {
					Expect(result.Requeue).To(BeTrue())
				}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: "default"}, task)).To(Succeed())
			}

			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseFinalAnswer))
			Expect(task.Status.Output).To(Equal("Test result"))

			// Verify that the token from the secret was correctly used as the API key
			Expect(mockHumanLayerClient.baseURL).To(Equal("https://api.example.com"))
			Expect(mockHumanLayerClient.apiKey).To(Equal("hl_testtoken"))
			Expect(mockHumanLayerClient.runID).To(Equal(testAgent.Name))
		})
	})
})
