package task

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/trace/noop"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	. "github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

var _ = Describe("Task Controller", func() {
	Context("'' -> Initializing", func() {
		ctx := context.Background()
		It("moves to Initializing and sets a span context", func() {
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			task := testTask.Setup(ctx)
			defer testTask.Teardown(ctx)

			By("reconciling the task")
			reconciler, _ := reconciler()
			reconciler.Tracer = noop.NewTracerProvider().Tracer("test")

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			By("checking the reconciler result")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			By("checking the task status")
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseInitializing))
			Expect(task.Status.SpanContext).NotTo(BeNil())
			Expect(task.Status.SpanContext.TraceID).NotTo(BeEmpty())
			Expect(task.Status.SpanContext.SpanID).NotTo(BeEmpty())
		})
	})
	Context("Initializing -> Error", func() {
		It("moves to error if the agent is not found", func() {
			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhaseInitializing,
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhasePending))
			Expect(task.Status.StatusDetail).To(ContainSubstring("Waiting for Agent to exist"))
			ExpectRecorder(recorder).ToEmitEventContaining("Waiting")
		})
	})
	Context("Initializing -> Pending", func() {
		It("moves to pending if upstream agent does not exist", func() {
			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhaseInitializing,
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhasePending))
			Expect(task.Status.StatusDetail).To(ContainSubstring("Waiting for Agent to exist"))
			ExpectRecorder(recorder).ToEmitEventContaining("Waiting")
		})
		It("moves to pending if upstream agent is not ready", func() {
			_ = testAgent.SetupWithStatus(ctx, acp.AgentStatus{
				Ready: false,
			})
			defer testAgent.Teardown(ctx)

			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhaseInitializing,
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhasePending))
			Expect(task.Status.StatusDetail).To(ContainSubstring("Waiting for agent \"test-agent\" to become ready"))
			ExpectRecorder(recorder).ToEmitEventContaining("Waiting for agent")
		})
	})
	Context("Initializing -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if there is a userMessage + agentRef", func() {
			testAgent.SetupWithStatus(ctx, acp.AgentStatus{
				Status: "Ready",
				Ready:  true,
			})
			defer testAgent.Teardown(ctx)

			testTask2 := &TestTask{
				name:        "test-task-2",
				agentName:   testAgent.name,
				userMessage: "test-user-message",
			}
			task := testTask2.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhaseInitializing,
			})
			defer testTask2.Teardown(ctx)

			By("reconciling the task")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask2.name, Namespace: "default"},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("ensuring the context window is set correctly")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask2.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseReadyForLLM))
			Expect(task.Status.ContextWindow).To(HaveLen(2))
			Expect(task.Status.ContextWindow[0].Role).To(Equal("system"))
			Expect(task.Status.ContextWindow[0].Content).To(ContainSubstring(testAgent.system))
			Expect(task.Status.ContextWindow[1].Role).To(Equal("user"))
			Expect(task.Status.ContextWindow[1].Content).To(ContainSubstring("test-user-message"))
			ExpectRecorder(recorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})
	Context("Pending -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if upstream dependencies are ready", func() {
			testAgent.SetupWithStatus(ctx, acp.AgentStatus{
				Status: "Ready",
				Ready:  true,
			})
			defer testAgent.Teardown(ctx)

			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhasePending,
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("ensuring the context window is set correctly")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseReadyForLLM))
			Expect(task.Status.StatusDetail).To(ContainSubstring("Ready to send to LLM"))
			Expect(task.Status.ContextWindow).To(HaveLen(2))
			Expect(task.Status.ContextWindow[0].Role).To(Equal("system"))
			Expect(task.Status.ContextWindow[0].Content).To(ContainSubstring(testAgent.system))
			Expect(task.Status.ContextWindow[1].Role).To(Equal("user"))
			Expect(task.Status.ContextWindow[1].Content).To(ContainSubstring(testTask.userMessage))
			ExpectRecorder(recorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})
	Context("ReadyForLLM -> LLMFinalAnswer", func() {
		It("moves to LLMFinalAnswer after getting a response from the LLM", func() {
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhaseReadyForLLM,
				ContextWindow: []acp.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.userMessage,
					},
				},
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockLLMClient{
				Response: &acp.Message{
					Role:    "assistant",
					Content: "The moon is a natural satellite of the Earth and lacks any formal government or capital.",
				},
			}
			reconciler.newLLMClient = func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
				return mockLLMClient, nil
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("ensuring the task status is updated with the llm final answer")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseFinalAnswer))
			Expect(task.Status.StatusDetail).To(ContainSubstring("LLM final response received"))
			Expect(task.Status.Output).To(Equal("The moon is a natural satellite of the Earth and lacks any formal government or capital."))
			Expect(task.Status.ContextWindow).To(HaveLen(3))
			Expect(task.Status.ContextWindow[2].Role).To(Equal("assistant"))
			Expect(task.Status.ContextWindow[2].Content).To(ContainSubstring("The moon is a natural satellite of the Earth and lacks any formal government or capital."))
			ExpectRecorder(recorder).ToEmitEventContaining("SendingContextWindowToLLM", "LLMFinalAnswer")

			By("ensuring the llm client was called correctly")
			Expect(mockLLMClient.Calls).To(HaveLen(1))
			Expect(mockLLMClient.Calls[0].Messages).To(HaveLen(2))
			Expect(mockLLMClient.Calls[0].Messages[0].Role).To(Equal("system"))
			Expect(mockLLMClient.Calls[0].Messages[0].Content).To(ContainSubstring(testAgent.system))
			Expect(mockLLMClient.Calls[0].Messages[1].Role).To(Equal("user"))
			Expect(mockLLMClient.Calls[0].Messages[1].Content).To(ContainSubstring(testTask.userMessage))
		})
	})
	Context("ReadyForLLM -> Error", func() {
		It("moves to Error state but not Failed phase on general error", func() {
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhaseReadyForLLM,
				ContextWindow: []acp.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.userMessage,
					},
				},
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task with a mock LLM client that returns an error")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockLLMClient{
				Error: fmt.Errorf("connection timeout"),
			}
			reconciler.newLLMClient = func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
				return mockLLMClient, nil
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).To(HaveOccurred())

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Status).To(Equal(acp.TaskStatusTypeError))
			// Phase shouldn't be Failed for general errors
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseReadyForLLM))
			Expect(task.Status.Error).To(Equal("connection timeout"))
			ExpectRecorder(recorder).ToEmitEventContaining("LLMRequestFailed")
		})

		It("moves to Error state AND Failed phase on 4xx error", func() {
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhaseReadyForLLM,
				ContextWindow: []acp.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.userMessage,
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

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).To(HaveOccurred())

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Status).To(Equal(acp.TaskStatusTypeError))
			// Phase should be Failed for 4xx errors
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseFailed))
			Expect(task.Status.Error).To(ContainSubstring("LLM request failed with status 400"))
			ExpectRecorder(recorder).ToEmitEventContaining("LLMRequestFailed4xx")
		})
	})
	Context("Error -> ErrorBackoff", func() {
		XIt("moves to ErrorBackoff if the error is retryable", func() {})
	})
	Context("Error -> Error", func() {
		XIt("Stays in Error if the error is not retryable", func() {})
	})
	Context("ErrorBackoff -> ReadyForLLM", func() {
		XIt("moves to ReadyForLLM after the backoff period", func() {})
	})
	Context("ReadyForLLM -> ToolCallsPending", func() {
		It("moves to ToolCallsPending if the LLM returns tool calls", func() {
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhaseReadyForLLM,
			})
			defer testTask.Teardown(ctx)

			By("reconciling the task")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockLLMClient{
				Response: &acp.Message{
					Role: "assistant",
					ToolCalls: []acp.ToolCall{
						{
							ID:       "1",
							Function: acp.ToolCallFunction{Name: "fetch__fetch", Arguments: `{"url": "https://api.example.com/data"}`},
						},
					},
				},
			}
			reconciler.newLLMClient = func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
				return mockLLMClient, nil
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("ensuring the task status is updated with the tool calls pending")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseToolCallsPending))
			Expect(task.Status.StatusDetail).To(ContainSubstring("LLM response received, tool calls pending"))
			ExpectRecorder(recorder).ToEmitEventContaining("SendingContextWindowToLLM", "ToolCallsPending")

			By("ensuring the tool call was created")
			toolCalls := &acp.TaskRunToolCallList{}
			Expect(k8sClient.List(ctx, toolCalls, client.InNamespace("default"))).To(Succeed())
			Expect(toolCalls.Items).To(HaveLen(1))
			Expect(toolCalls.Items[0].Spec.ToolRef.Name).To(Equal("fetch__fetch"))
			Expect(toolCalls.Items[0].Spec.Arguments).To(Equal(`{"url": "https://api.example.com/data"}`))

			By("cleaning up the tool call")
			Expect(k8sClient.Delete(ctx, &toolCalls.Items[0])).To(Succeed())
		})
	})
	Context("ToolCallsPending -> Error", func() {
		XIt("moves to Error if its in ToolCallsPending but no tool calls are found", func() {
			// todo
		})
	})
	Context("ToolCallsPending -> ToolCallsPending", func() {
		It("Stays in ToolCallsPending if the tool calls are not completed", func() {
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase:             acp.TaskPhaseToolCallsPending,
				ToolCallRequestID: "test123",
			})
			defer testTask.Teardown(ctx)

			testTaskRunToolCall.SetupWithStatus(ctx, acp.TaskRunToolCallStatus{
				Phase: acp.TaskRunToolCallPhasePending,
			})
			defer testTaskRunToolCall.Teardown(ctx)

			By("reconciling the task")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseToolCallsPending))
		})
	})
	Context("ToolCallsPending -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if all tool calls are completed", func() {
			_, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			By("setting up the task with a tool call pending")
			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase:             acp.TaskPhaseToolCallsPending,
				ToolCallRequestID: "test123",
				ContextWindow: []acp.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.userMessage,
					},
					{
						Role: "assistant",
						ToolCalls: []acp.ToolCall{
							{
								ID: "1",
								Function: acp.ToolCallFunction{
									Name:      "fetch__fetch",
									Arguments: `{"url": "https://api.example.com/data"}`,
								},
							},
						},
					},
				},
			})
			defer testTask.Teardown(ctx)

			testTaskRunToolCall.SetupWithStatus(ctx, acp.TaskRunToolCallStatus{
				Status: acp.TaskRunToolCallStatusTypeSucceeded,
				Phase:  acp.TaskRunToolCallPhaseSucceeded,
				Result: `{"data": "test-data"}`,
			})
			defer testTaskRunToolCall.Teardown(ctx)

			testTaskRunToolCallTwo.SetupWithStatus(ctx, acp.TaskRunToolCallStatus{
				Status: acp.TaskRunToolCallStatusTypeSucceeded,
				Phase:  acp.TaskRunToolCallPhaseToolCallRejected,
				Result: `human contact channel rejected this tool call with the following response: "I'm out here just testing things okay, try again later."`,
			})
			defer testTaskRunToolCallTwo.Teardown(ctx)

			By("reconciling the task")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTask.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("checking the task status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTask.name, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseReadyForLLM))
			Expect(task.Status.StatusDetail).To(ContainSubstring("All tool calls completed, ready to send tool results to LLM"))
			ExpectRecorder(recorder).ToEmitEventContaining("AllToolCallsCompleted")

			// todo expect the context window has the tool call result appended
			Expect(task.Status.ContextWindow).To(HaveLen(5))
			Expect(task.Status.ContextWindow[3].Role).To(Equal("tool"))
			Expect(task.Status.ContextWindow[3].Content).To(ContainSubstring("test-data"))
			Expect(task.Status.ContextWindow[4].Role).To(Equal("tool"))
			Expect(task.Status.ContextWindow[4].Content).To(ContainSubstring("I'm out here just testing things okay, try again later."))
		})
	})
	Context("LLMFinalAnswer -> LLMFinalAnswer", func() {
		It("stays in LLMFinalAnswer", func() {})
	})
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		// todo(dex) i think this is not needed anymore - check version history to restore it
		XIt("should progress through phases correctly", func() {})

		// todo(dex) i think this is not needed anymore - check version history to restore it
		XIt("should clear error field when entering ready state", func() {})

		// todo(dex) i think this is not needed anymore - check version history to restore it
		XIt("should pass tools correctly to LLM and handle tool calls", func() {})

		// todo(dex) i think this is not needed anymore - check version history to restore it
		XIt("should keep the task run in the ToolCallsPending state when tool call is pending", func() {})

		// todo dex should fix this but trying to get something merged in asap
		XIt("should correctly handle multi-message conversations with the LLM", func() {
			uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
			testTaskName := fmt.Sprintf("multi-message-%s", uniqueSuffix)

			By("setting up the task with an existing conversation history")
			task := testTask.SetupWithStatus(ctx, acp.TaskStatus{
				Phase: acp.TaskPhaseReadyForLLM,
				ContextWindow: []acp.Message{
					{
						Role:    "system",
						Content: "you are a testing assistant",
					},
					{
						Role:    "user",
						Content: "what is 2 + 2?",
					},
					{
						Role:    "assistant",
						Content: "2 + 2 = 4",
					},
					{
						Role:    "user",
						Content: "what is 4 + 4?",
					},
				},
			})
			defer testTask.Teardown(ctx)

			By("creating a mock LLM client that validates context window messages are passed correctly")
			mockClient := &llmclient.MockLLMClient{
				Response: &acp.Message{
					Role:    "assistant",
					Content: "4 + 4 = 8",
				},
				ValidateContextWindow: func(contextWindow []acp.Message) error {
					Expect(contextWindow).To(HaveLen(4), "All 4 messages should be sent to the LLM")

					// Verify all messages are present in the correct order
					Expect(contextWindow[0].Role).To(Equal("system"))
					Expect(contextWindow[0].Content).To(Equal("you are a testing assistant"))

					Expect(contextWindow[1].Role).To(Equal("user"))
					Expect(contextWindow[1].Content).To(Equal("what is 2 + 2?"))

					Expect(contextWindow[2].Role).To(Equal("assistant"))
					Expect(contextWindow[2].Content).To(Equal("2 + 2 = 4"))

					Expect(contextWindow[3].Role).To(Equal("user"))
					Expect(contextWindow[3].Content).To(Equal("what is 4 + 4?"))

					return nil
				},
			}

			By("reconciling the task")
			reconciler, _ := reconciler()
			reconciler.newLLMClient = func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
				return mockClient, nil
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testTaskName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking that the task moved to FinalAnswer phase")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskName, Namespace: "default"}, task)).To(Succeed())
			Expect(task.Status.Phase).To(Equal(acp.TaskPhaseFinalAnswer))

			By("checking that the new assistant response was appended to the context window")
			Expect(task.Status.ContextWindow).To(HaveLen(5))
			lastMessage := task.Status.ContextWindow[4]
			Expect(lastMessage.Role).To(Equal("assistant"))
			Expect(lastMessage.Content).To(Equal("4 + 4 = 8"))
		})
	})
})
