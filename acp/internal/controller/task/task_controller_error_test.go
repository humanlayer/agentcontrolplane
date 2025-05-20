package task

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	. "github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

var _ = Describe("Task Controller Error Handling", func() {
	Context("ReadyForLLM -> Error -> Failed", func() {
		It("moves to Error state AND Failed phase on HTTP 4xx error", func() {
			// Set up the objects needed for the test
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			task := testTask.SetupWithStatus(ctx, k8sClient, acp.TaskStatus{
				Phase: acp.TaskPhaseReadyForLLM, // Start from ReadyForLLM
				SpanContext: &acp.SpanContext{ // Add a dummy span context
					TraceID: "0af7651916cd43dd8448eb211c80319c",
					SpanID:  "b7ad6b7169203331",
				},
				ContextWindow: []acp.Message{
					{
						Role:    "system",
						Content: testAgent.SystemPrompt,
					},
					{
						Role:    "user",
						Content: testTask.UserMessage,
					},
				},
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task with a mock LLM client that returns a 400 error")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockLLMClient{
				Error: &llmclient.LLMRequestError{
					StatusCode: 400,
					Message:    "invalid request: model not found",
					Err:        fmt.Errorf("LLM API request failed"),
				},
			}
			reconciler.newLLMClient = func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
				return mockLLMClient, nil
			}

			// First reconcile should update to SendContextWindowToLLM
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.Name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Check the task status after first reconcile
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.Name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseSendContextWindowToLLM))

			// Second reconcile (should handle SendContextWindowToLLM -> Failed)
			result, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.Name, Namespace: "default"},
			})
			// Expect NO error returned because the status is updated to Failed (terminal)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeZero())

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.Name, Namespace: "default"}, task)).To(Succeed())

			// Check status is Error and phase is Failed for 4xx errors
			Expect(task.Status.Status).To(Equal(acp.TaskStatusTypeError))
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseFailed))
			Expect(task.Status.Error).To(ContainSubstring("LLM request failed with status 400"))
			ExpectRecorder(recorder).ToEmitEventContaining("LLMRequestFailed4xx")
		})

		It("moves to Error state but NOT Failed phase on general (non-4xx) error", func() {
			// Set up the objects needed for the test
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			task := testTask.SetupWithStatus(ctx, k8sClient, acp.TaskStatus{
				Phase: acp.TaskPhaseReadyForLLM, // Start from ReadyForLLM
				SpanContext: &acp.SpanContext{
					TraceID: "0af7651916cd43dd8448eb211c80319c",
					SpanID:  "b7ad6b7169203331",
				},
				ContextWindow: []acp.Message{
					{
						Role:    "system",
						Content: testAgent.SystemPrompt,
					},
					{
						Role:    "user",
						Content: testTask.UserMessage,
					},
				},
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task with a mock LLM client that returns a general error")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockLLMClient{
				Error: fmt.Errorf("connection timeout"),
			}
			reconciler.newLLMClient = func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
				return mockLLMClient, nil
			}

			// First reconcile should update to SendContextWindowToLLM
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.Name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Check the task status after first reconcile
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.Name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseSendContextWindowToLLM))

			// Second reconcile (should handle SendContextWindowToLLM -> Error)
			result, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.Name, Namespace: "default"},
			})
			Expect(err).To(HaveOccurred()) // Expect the error to be returned for requeue

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.Name, Namespace: "default"}, task)).To(Succeed())

			// Status should be Error, but phase should remain SendContextWindowToLLM
			Expect(task.Status.Status).To(Equal(acp.TaskStatusTypeError))
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseSendContextWindowToLLM))
			Expect(task.Status.Error).To(Equal("connection timeout"))
			ExpectRecorder(recorder).ToEmitEventContaining("LLMRequestFailed")
		})

		It("moves to Error state AND Failed phase on explicit LLMRequestError with 429 status code", func() {
			// Set up the objects needed for the test
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			task := testTask.SetupWithStatus(ctx, k8sClient, acp.TaskStatus{
				Phase: acp.TaskPhaseReadyForLLM, // Start from ReadyForLLM
				SpanContext: &acp.SpanContext{
					TraceID: "0af7651916cd43dd8448eb211c80319c",
					SpanID:  "b7ad6b7169203331",
				},
				ContextWindow: []acp.Message{
					{
						Role:    "system",
						Content: testAgent.SystemPrompt,
					},
					{
						Role:    "user",
						Content: testTask.UserMessage,
					},
				},
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task with a mock LLM client that returns a 429 error")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockLLMClient{
				Error: &llmclient.LLMRequestError{
					StatusCode: 429,
					Message:    "rate limit exceeded",
					Err:        fmt.Errorf("LLM API rate limit exceeded"),
				},
			}
			reconciler.newLLMClient = func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
				return mockLLMClient, nil
			}

			// First reconcile should update to SendContextWindowToLLM
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.Name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Check the task status after first reconcile
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.Name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseSendContextWindowToLLM))

			// Second reconcile with the error
			result, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.Name, Namespace: "default"},
			})

			// The controller will actually return nil error since it handles 4xx errors internally
			Expect(err).NotTo(HaveOccurred())

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.Name, Namespace: "default"}, task)).To(Succeed())

			// Based on the controller code, all HTTP 400-499 errors are treated as terminal client errors
			Expect(task.Status.Status).To(Equal(acp.TaskStatusTypeError))
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseFailed))
			Expect(task.Status.Error).To(ContainSubstring("LLM request failed with status 429"))
			ExpectRecorder(recorder).ToEmitEventContaining("LLMRequestFailed4xx")
		})
	})
})
