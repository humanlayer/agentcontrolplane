package taskrun

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
	. "github.com/humanlayer/smallchain/kubechain/test/utils"
)

var _ = Describe("TaskRun Controller", func() {
	Context("'' -> Initializing", func() {
		ctx := context.Background()
		It("moves to Initializing and sets a span context", func() {
			testTask.Setup(ctx)
			defer testTask.Teardown(ctx)

			taskRun := testTaskRun.Setup(ctx)
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			By("checking the reconciler result")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			By("checking the taskrun status")
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseInitializing))
			Expect(taskRun.Status.SpanContext).NotTo(BeNil())
			Expect(taskRun.Status.SpanContext.TraceID).NotTo(BeEmpty())
			Expect(taskRun.Status.SpanContext.SpanID).NotTo(BeEmpty())
		})
	})
	Context("Initializing -> Error", func() {
		It("moves to error if the task is not found", func() {
			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseInitializing,
			})
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			// todo dont error if not found, don't requeue
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFailed))
			Expect(taskRun.Status.Error).To(Equal("Task \"test-task\" not found"))
		})
	})
	Context("Initializing -> Pending", func() {
		It("moves to pending if upstream task is not ready", func() {
			_ = testTask.SetupWithStatus(ctx, kubechain.TaskStatus{
				Status: kubechain.TaskStatusPending,
			})
			defer testTask.Teardown(ctx)

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseInitializing,
			})
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhasePending))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("Waiting for task \"test-task\" to become ready"))
			ExpectRecorder(recorder).ToEmitEventContaining("TaskNotReady")
		})
	})
	Context("Initializing -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if the task is ready", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseInitializing,
			})
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("ensuring the context window is set correctly")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("Ready to send to LLM"))
			Expect(taskRun.Status.ContextWindow).To(HaveLen(2))
			Expect(taskRun.Status.ContextWindow[0].Role).To(Equal("system"))
			Expect(taskRun.Status.ContextWindow[0].Content).To(ContainSubstring(testAgent.system))
			Expect(taskRun.Status.ContextWindow[1].Role).To(Equal("user"))
			Expect(taskRun.Status.ContextWindow[1].Content).To(ContainSubstring(testTask.message))
			ExpectRecorder(recorder).ToEmitEventContaining("ValidationSucceeded")
		})

		XIt("moves to ReadyForLLM if the taskRun is missing a taskRef but has an agentRef and userMessage ", func() {})
	})
	Context("Pending -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if upstream dependencies are ready", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhasePending,
			})
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("ensuring the context window is set correctly")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("Ready to send to LLM"))
			Expect(taskRun.Status.ContextWindow).To(HaveLen(2))
			Expect(taskRun.Status.ContextWindow[0].Role).To(Equal("system"))
			Expect(taskRun.Status.ContextWindow[0].Content).To(ContainSubstring(testAgent.system))
			Expect(taskRun.Status.ContextWindow[1].Role).To(Equal("user"))
			Expect(taskRun.Status.ContextWindow[1].Content).To(ContainSubstring(testTask.message))
			ExpectRecorder(recorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})
	Context("ReadyForLLM -> LLMFinalAnswer", func() {
		It("moves to LLMFinalAnswer after getting a response from the LLM", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseReadyForLLM,
				ContextWindow: []kubechain.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.message,
					},
				},
			})
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockRawOpenAIClient{
				Response: &v1alpha1.Message{
					Role:    "assistant",
					Content: "The moon is a natural satellite of the Earth and lacks any formal government or capital.",
				},
			}
			reconciler.newLLMClient = func(apiKey string) (llmclient.OpenAIClient, error) {
				return mockLLMClient, nil
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("ensuring the taskrun status is updated with the llm final answer")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFinalAnswer))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("LLM final response received"))
			Expect(taskRun.Status.Output).To(Equal("The moon is a natural satellite of the Earth and lacks any formal government or capital."))
			Expect(taskRun.Status.ContextWindow).To(HaveLen(3))
			Expect(taskRun.Status.ContextWindow[2].Role).To(Equal("assistant"))
			Expect(taskRun.Status.ContextWindow[2].Content).To(ContainSubstring("The moon is a natural satellite of the Earth and lacks any formal government or capital."))
			ExpectRecorder(recorder).ToEmitEventContaining("SendingContextWindowToLLM", "LLMFinalAnswer")

			By("ensuring the llm client was called correctly")
			Expect(mockLLMClient.Calls).To(HaveLen(1))
			Expect(mockLLMClient.Calls[0].Messages).To(HaveLen(2))
			Expect(mockLLMClient.Calls[0].Messages[0].Role).To(Equal("system"))
			Expect(mockLLMClient.Calls[0].Messages[0].Content).To(ContainSubstring(testAgent.system))
			Expect(mockLLMClient.Calls[0].Messages[1].Role).To(Equal("user"))
			Expect(mockLLMClient.Calls[0].Messages[1].Content).To(ContainSubstring(testTask.message))
		})
	})
	Context("ReadyForLLM -> Error", func() {
		It("moves to Error state but not Failed phase on general error", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseReadyForLLM,
				ContextWindow: []kubechain.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.message,
					},
				},
			})
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun with a mock LLM client that returns an error")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockRawOpenAIClient{
				Error: fmt.Errorf("connection timeout"),
			}
			reconciler.newLLMClient = func(apiKey string) (llmclient.OpenAIClient, error) {
				return mockLLMClient, nil
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).To(HaveOccurred())

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Status).To(Equal(kubechain.TaskRunStatusStatusError))
			// Phase shouldn't be Failed for general errors
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(taskRun.Status.Error).To(Equal("connection timeout"))
			ExpectRecorder(recorder).ToEmitEventContaining("LLMRequestFailed")
		})

		It("moves to Error state AND Failed phase on 4xx error", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseReadyForLLM,
				ContextWindow: []kubechain.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.message,
					},
				},
			})
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun with a mock LLM client that returns a 400 error")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockRawOpenAIClient{
				Error: &llmclient.LLMRequestError{
					StatusCode: 400,
					Message:    "invalid request: model not found",
					Err:        fmt.Errorf("OpenAI API request failed"),
				},
			}
			reconciler.newLLMClient = func(apiKey string) (llmclient.OpenAIClient, error) {
				return mockLLMClient, nil
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).To(HaveOccurred())

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Status).To(Equal(kubechain.TaskRunStatusStatusError))
			// Phase should be Failed for 4xx errors
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFailed))
			Expect(taskRun.Status.Error).To(ContainSubstring("LLM request failed with status 400"))
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
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseReadyForLLM,
			})
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockRawOpenAIClient{
				Response: &v1alpha1.Message{
					Role: "assistant",
					ToolCalls: []v1alpha1.ToolCall{
						{
							ID:       "1",
							Function: v1alpha1.ToolCallFunction{Name: "fetch__fetch", Arguments: `{"url": "https://api.example.com/data"}`},
						},
					},
				},
			}
			reconciler.newLLMClient = func(apiKey string) (llmclient.OpenAIClient, error) {
				return mockLLMClient, nil
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("ensuring the taskrun status is updated with the tool calls pending")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseToolCallsPending))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("LLM response received, tool calls pending"))
			ExpectRecorder(recorder).ToEmitEventContaining("SendingContextWindowToLLM", "ToolCallsPending")

			By("ensuring the tool call was created")
			toolCalls := &kubechain.TaskRunToolCallList{}
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
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase:             kubechain.TaskRunPhaseToolCallsPending,
				ToolCallRequestID: "test123",
			})
			defer testTaskRun.Teardown(ctx)

			testTaskRunToolCall.SetupWithStatus(ctx, kubechain.TaskRunToolCallStatus{
				Phase: kubechain.TaskRunToolCallPhasePending,
			})
			defer testTaskRunToolCall.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseToolCallsPending))
		})
	})
	Context("ToolCallsPending -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if all tool calls are completed", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			By("setting up the taskrun with a tool call pending")
			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase:             kubechain.TaskRunPhaseToolCallsPending,
				ToolCallRequestID: "test123",
				ContextWindow: []kubechain.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.message,
					},
					{
						Role: "assistant",
						ToolCalls: []kubechain.ToolCall{
							{
								ID: "1",
								Function: kubechain.ToolCallFunction{
									Name:      "fetch__fetch",
									Arguments: `{"url": "https://api.example.com/data"}`,
								},
							},
						},
					},
				},
			})
			defer testTaskRun.Teardown(ctx)

			testTaskRunToolCall.SetupWithStatus(ctx, kubechain.TaskRunToolCallStatus{
				Phase:  kubechain.TaskRunToolCallPhaseSucceeded,
				Result: `{"data": "test-data"}`,
			})
			defer testTaskRunToolCall.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("All tool calls completed, ready to send tool results to LLM"))
			ExpectRecorder(recorder).ToEmitEventContaining("AllToolCallsCompleted")

			// todo expect the context window has the tool call result appended
			Expect(taskRun.Status.ContextWindow).To(HaveLen(4))
			Expect(taskRun.Status.ContextWindow[3].Role).To(Equal("tool"))
			Expect(taskRun.Status.ContextWindow[3].Content).To(ContainSubstring("test-data"))
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
		XIt("should pass tools correctly to OpenAI and handle tool calls", func() {})

		// todo(dex) i think this is not needed anymore - check version history to restore it
		XIt("should keep the task run in the ToolCallsPending state when tool call is pending", func() {})

		// todo dex should fix this but trying to get something merged in asap
		XIt("should correctly handle multi-message conversations with the LLM", func() {
			uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
			testTaskRunName := fmt.Sprintf("multi-message-%s", uniqueSuffix)

			By("setting up the taskrun with an existing conversation history")
			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseReadyForLLM,
				ContextWindow: []kubechain.Message{
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
			defer testTaskRun.Teardown(ctx)

			By("creating a mock OpenAI client that validates context window messages are passed correctly")
			mockClient := &llmclient.MockRawOpenAIClient{
				Response: &kubechain.Message{
					Role:    "assistant",
					Content: "4 + 4 = 8",
				},
				ValidateContextWindow: func(contextWindow []kubechain.Message) error {
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

			By("reconciling the taskrun")
			reconciler, _ := reconciler()
			reconciler.newLLMClient = func(apiKey string) (llmclient.OpenAIClient, error) {
				return mockClient, nil
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testTaskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking that the taskrun moved to FinalAnswer phase")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRunName, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFinalAnswer))

			By("checking that the new assistant response was appended to the context window")
			Expect(taskRun.Status.ContextWindow).To(HaveLen(5))
			lastMessage := taskRun.Status.ContextWindow[4]
			Expect(lastMessage.Role).To(Equal("assistant"))
			Expect(lastMessage.Content).To(Equal("4 + 4 = 8"))
		})

		// todo(dex) i think this is not needed anymore - check version history to restore it
		XIt("should transition to ReadyForLLM when all tool calls are complete", func() {})
	})
})
