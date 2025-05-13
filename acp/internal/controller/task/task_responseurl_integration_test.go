package task

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/trace/noop"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MockLLMClient for testing
type MockLLMClient struct {
	SendRequestResponse *acp.Message
	SendRequestError    error
}

func (m *MockLLMClient) SendRequest(ctx context.Context, messages []acp.Message, tools []llmclient.Tool) (*acp.Message, error) {
	return m.SendRequestResponse, m.SendRequestError
}

// Creates a TaskReconciler with a custom LLM client factory
func reconcilerWithMockLLM(newLLMClient func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error)) (*TaskReconciler, *record.FakeRecorder) {
	recorder := record.NewFakeRecorder(10)
	tracer := noop.NewTracerProvider().Tracer("test")

	r := &TaskReconciler{
		Client:       k8sClient,
		Scheme:       k8sClient.Scheme(),
		recorder:     recorder,
		newLLMClient: newLLMClient,
		Tracer:       tracer,
	}
	return r, recorder
}

var _ = Describe("Task Controller with ResponseUrl", func() {
	Context("when Task has responseUrl", func() {
		var (
			server          *httptest.Server
			requestReceived chan struct{}
			receivedRequest humanlayerapi.HumanContactInput
			receivedMutex   sync.Mutex
			mockLLMClient   *MockLLMClient
		)

		BeforeEach(func() {
			// Set up the secret, LLM, and agent
			_, _, _, teardown := setupSuiteObjects(ctx)
			DeferCleanup(teardown)

			// Set up the mock LLM client to return a final answer
			mockLLMClient = &MockLLMClient{
				SendRequestResponse: &acp.Message{
					Content: "This is the final answer",
				},
			}

			// Set up the test server to receive the HTTP request
			requestReceived = make(chan struct{})
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Decode the request body
				decoder := json.NewDecoder(r.Body)
				var req humanlayerapi.HumanContactInput
				Expect(decoder.Decode(&req)).To(Succeed())

				// Store the request for later verification
				receivedMutex.Lock()
				receivedRequest = req
				receivedMutex.Unlock()

				// Send a success response
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"success"}`))

				// Notify that request was received
				close(requestReceived)
			}))
			DeferCleanup(server.Close)
		})

		It("sends the final result to the responseUrl", func() {
			By("creating a task with responseUrl")
			// Create a task with responseUrl
			customTask := &acp.Task{
				ObjectMeta: v1.ObjectMeta{
					Name:      "task-with-responseurl",
					Namespace: "default",
				},
				Spec: acp.TaskSpec{
					AgentRef: acp.LocalObjectReference{
						Name: testAgent.Name,
					},
					UserMessage: "What is the capital of France?",
					ResponseUrl: server.URL,
				},
			}
			Expect(k8sClient.Create(ctx, customTask)).To(Succeed())
			task := customTask
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, task)).To(Succeed())
			})

			// Create a mock LLM client factory
			mockLLMClientFn := func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
				return mockLLMClient, nil
			}

			// Get reconciler with mock LLM client
			By("creating reconciler with mock LLM client")
			reconciler, _ := reconcilerWithMockLLM(mockLLMClientFn)

			By("reconciling the task to initialize it")
			// First reconcile (should initialize the task)
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: task.Name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Get the updated task
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseInitializing))

			By("reconciling the task to prepare for LLM")
			// Second reconcile (should validate agent and prepare for LLM)
			result, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: task.Name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Get the updated task
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseReadyForLLM))

			By("reconciling the task to get final answer")
			// Third reconcile (should send to LLM and get final answer)
			result, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: task.Name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())

			// Get the updated task
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseFinalAnswer))
			Expect(task.Status.Output).To(Equal("This is the final answer"))

			By("waiting for HTTP request to be received")
			// Wait for the HTTP request to be made
			select {
			case <-requestReceived:
				// Request was received, continue with assertions
			case <-time.After(5 * time.Second):
				Fail("Timed out waiting for responseUrl request")
			}

			By("verifying request content")
			// Verify the request content
			receivedMutex.Lock()
			defer receivedMutex.Unlock()
			Expect(receivedRequest.Spec.Msg).To(Equal("This is the final answer"))
		})
	})
})
