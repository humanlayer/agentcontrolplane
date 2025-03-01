package controller

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
)

var _ = Describe("TaskRun Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-taskrun"
		const taskName = "test-task"
		const agentName = "test-agent"
		const taskRunName = "test-taskrun"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			// Create a test secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"api-key": []byte("test-api-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			// Create a test LLM
			llm := &kubechain.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-llm",
					Namespace: "default",
				},
				Spec: kubechain.LLMSpec{
					Provider: "openai",
					APIKeyFrom: kubechain.APIKeySource{
						SecretKeyRef: kubechain.SecretKeyRef{
							Name: "test-secret",
							Key:  "api-key",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, llm)).To(Succeed())

			// Mark LLM as ready
			llm.Status.Ready = true
			llm.Status.Status = "Ready"
			llm.Status.StatusDetail = "Ready for testing"
			Expect(k8sClient.Status().Update(ctx, llm)).To(Succeed())

			tool := &kubechain.Tool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "add",
					Namespace: "default",
				},
				Spec: kubechain.ToolSpec{
					Name:        "add",
					Description: "add two numbers",
					Execute: kubechain.ToolExecute{
						Builtin: &kubechain.BuiltinToolSpec{
							Name: "add",
						},
					},
					Parameters: runtime.RawExtension{
						Raw: []byte(`{
							"type": "object",
							"properties": {
								"a": {
									"type": "number"
								},
								"b": {
									"type": "number"
								}
							},
							"required": ["a", "b"]
						}`),
					},
				},
			}
			Expect(k8sClient.Create(ctx, tool)).To(Succeed())

			// Create a test Agent
			agent := &kubechain.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      agentName,
					Namespace: "default",
				},
				Spec: kubechain.AgentSpec{
					LLMRef: kubechain.LocalObjectReference{
						Name: "test-llm",
					},
					System: "you are a testing assistant",
					Tools: []kubechain.LocalObjectReference{
						{
							Name: "add",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())

			// Mark Agent as ready
			agent.Status.Ready = true
			agent.Status.Status = "Ready"
			agent.Status.StatusDetail = "Ready for testing"
			agent.Status.ValidTools = []kubechain.ResolvedTool{
				{
					Kind: "Tool",
					Name: "add",
				},
			}
			Expect(k8sClient.Status().Update(ctx, agent)).To(Succeed())

			// Create a test Task
			task := &kubechain.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskName,
					Namespace: "default",
				},
				Spec: kubechain.TaskSpec{
					AgentRef: kubechain.LocalObjectReference{
						Name: agentName,
					},
					Message: "What state is San Francisco in?",
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			// Mark Task as ready
			task.Status.Ready = true
			task.Status.Status = "Ready"
			task.Status.StatusDetail = "Agent validated successfully"
			Expect(k8sClient.Status().Update(ctx, task)).To(Succeed())
		})

		AfterEach(func() {
			// Cleanup test resources
			By("Cleanup the test secret")
			secret := &corev1.Secret{}
			var err error // Declare err at the start
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}

			By("Cleanup the test LLM")
			llm := &kubechain.LLM{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-llm", Namespace: "default"}, llm)
			if err == nil {
				Expect(k8sClient.Delete(ctx, llm)).To(Succeed())
			}

			By("Cleanup the test Tool")
			tool := &kubechain.Tool{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "add", Namespace: "default"}, tool)
			if err == nil {
				Expect(k8sClient.Delete(ctx, tool)).To(Succeed())
			}

			By("Cleanup the test Agent")
			agent := &kubechain.Agent{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: agentName, Namespace: "default"}, agent)
			if err == nil {
				Expect(k8sClient.Delete(ctx, agent)).To(Succeed())
			}

			By("Cleanup the test Task")
			task := &kubechain.Task{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskName, Namespace: "default"}, task)
			if err == nil {
				Expect(k8sClient.Delete(ctx, task)).To(Succeed())
			}

			By("Cleanup the test TaskRun")
			taskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, taskRun)
			if err == nil {
				Expect(k8sClient.Delete(ctx, taskRun)).To(Succeed())
			}
		})

		It("should progress through the taskrun lifecycle for a simple task with no tools", func() {
			By("creating the taskrun")
			taskRun := &kubechain.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("reconciling the taskrun")
			mockClient := &llmclient.MockRawOpenAIClient{}

			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
				newLLMClient: func(apiKey string) (llmclient.OpenAIClient, error) {
					return mockClient, nil
				},
			}

			// First reconciliation - should set ReadyForLLM phase
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("checking taskrun moves to pending after first reconciliation")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhasePending))
			Expect(updatedTaskRun.Status.Status).To(Equal("Pending"))
			Expect(updatedTaskRun.Status.Ready).To(BeFalse())

			By("reconciling a second time")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking taskrun moves to ready after second reconciliation")
			updatedTaskRun = &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal("Ready"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Ready to send to LLM"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))

			By("checking initial taskrun status")
			updatedTaskRun = &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal("Ready"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Ready to send to LLM"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))

			By("checking that validation success event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "ValidationSucceeded")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find validation success event")

			By("reconciling the taskrun again")
			mockClient.Response = &kubechain.Message{
				Role:    "assistant",
				Content: "San Francisco is in California",
			}
			// Second reconciliation - should send to LLM and get response
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking post-LLM taskrun status")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal("Ready"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("LLM final response received"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFinalAnswer))
			Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(3))

			By("checking that LLM event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "LLMFinalAnswer")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find LLM final answer event")
		})

		It("should clear error field when entering ready state", func() {
			By("creating a taskrun with an error")
			taskRun := &kubechain.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: taskName,
					},
				},
				Status: kubechain.TaskRunStatus{
					Error: "previous error that should be cleared",
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("reconciling the taskrun")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the taskrun status")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal("Ready"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Ready to send to LLM"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(updatedTaskRun.Status.Error).To(BeEmpty(), "Error field should be cleared")
		})

		It("should fail when task doesn't exist", func() {
			By("creating the taskrun with non-existent task")
			taskRun := &kubechain.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: "nonexistent-task",
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("reconciling the taskrun")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())

			By("checking the taskrun status")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeFalse())
			Expect(updatedTaskRun.Status.Status).To(Equal("Error"))
			Expect(updatedTaskRun.Status.StatusDetail).To(ContainSubstring("failed to get Task"))
			Expect(updatedTaskRun.Status.Error).To(ContainSubstring("failed to get Task"))

			By("checking that a failure event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "ValidationFailed")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find failure event")
		})

		It("should set pending status when task exists but is not ready", func() {
			By("creating a task that is not ready")
			unreadyTask := &kubechain.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unready-task",
					Namespace: "default",
				},
				Spec: kubechain.TaskSpec{
					AgentRef: kubechain.LocalObjectReference{
						Name: agentName,
					},
					Message: "Test input",
				},
				Status: kubechain.TaskStatus{
					Ready: false,
				},
			}
			Expect(k8sClient.Create(ctx, unreadyTask)).To(Succeed())

			By("creating the taskrun referencing the unready task")
			taskRun := &kubechain.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: "unready-task",
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("reconciling the taskrun")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second))

			By("checking the taskrun status")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeFalse())
			Expect(updatedTaskRun.Status.Status).To(Equal("Pending"))
			Expect(updatedTaskRun.Status.StatusDetail).To(ContainSubstring("Waiting for task"))
			Expect(updatedTaskRun.Status.Error).To(BeEmpty())

			By("checking that a waiting event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "Waiting")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find waiting event")
		})

		// 	It("should pass tools correctly to OpenAI and handle tool calls", func() {
		// 		By("creating the taskrun")
		// 		taskRun := &v1alpha1.TaskRun{
		// 			ObjectMeta: metav1.ObjectMeta{
		// 				Name:      taskRunName,
		// 				Namespace: "default",
		// 			},
		// 			Spec: v1alpha1.TaskRunSpec{
		// 				TaskRef: v1alpha1.LocalObjectReference{
		// 					Name: taskName,
		// 				},
		// 			},
		// 		}
		// 		Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

		// 		By("creating a mock OpenAI client that validates tools and returns tool calls")
		// 		mockClient := &llmclient.MockOpenAIClient{sistant},
		// 			Response: &openai.ChatCompletionMessage{
		// 				ToolCalls: []openai.ChatCompletionMessageToolCall{
		// 					{
		// 						ID:   "call_123",
		// 						Type: openai.ChatCompletionMessageToolCallTypeFunction,
		// 						Function: openai.ChatCompletionMessageToolCallFunction{
		// 							Name:      "add",
		// 							Arguments: `{"a": 1, "b": 2}`,
		// 						},
		// 					},
		// 				},
		// 			},
		// 			ValidateTools: func(tools []openai.ChatCompletionToolParam) error {
		// 				Expect(tools).To(HaveLen(1))
		// 				Expect(tools[0].Type.Value).To(Equal(openai.ChatCompletionToolTypeFunction))
		// 				Expect(tools[0].Function.Value.Name.Value).To(Equal("add"))
		// 				Expect(tools[0].Function.Value.Description.Value).To(Equal("add two numbers"))
		// 				// Verify parameters were passed correctly
		// 				Expect(tools[0].Function.Value.Parameters.Value).To(Equal(openai.FunctionParameters{
		// 					"type": "object",
		// 					"properties": map[string]interface{}{
		// 						"a": map[string]interface{}{
		// 							"type": "number",
		// 						},
		// 						"b": map[string]interface{}{
		// 							"type": "number",
		// 						},
		// 					},
		// 					"required": []interface{}{"a", "b"},
		// 				}))
		// 				return nil
		// 			},
		// 		}

		// 		By("reconciling the taskrun")
		// 		eventRecorder := record.NewFakeRecorder(10)
		// 		reconciler := &TaskRunReconciler{
		// 			Client:   k8sClient,
		// 			Scheme:   k8sClient.Scheme(),
		// 			recorder: eventRecorder,
		// 			newLLMClient: func(apiKey string) (llmclient.OpenAIClient, error) {
		// 				return mockClient, nil
		// 			},
		// 		}

		// 		// First reconciliation - should set ReadyForLLM phase
		// 		_, err := reconciler.Reconcile(ctx, reconcile.Request{
		// 			NamespacedName: types.NamespacedName{
		// 				Name:      taskRunName,
		// 				Namespace: "default",
		// 			},
		// 		})
		// 		Expect(err).NotTo(HaveOccurred())

		// 		By("checking initial taskrun status")
		// 		updatedTaskRun := &v1alpha1.TaskRun{}
		// 		err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
		// 		Expect(err).NotTo(HaveOccurred())
		// 		Expect(updatedTaskRun.Status.Phase).To(Equal(v1alpha1.TaskRunPhaseReadyForLLM))
		// 		Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(2)) // System + User message

		// 		By("reconciling the taskrun again")
		// 		// Second reconciliation - should send to LLM and get tool calls
		// 		_, err = reconciler.Reconcile(ctx, reconcile.Request{
		// 			NamespacedName: types.NamespacedName{
		// 				Name:      taskRunName,
		// 				Namespace: "default",
		// 			},
		// 		})
		// 		Expect(err).NotTo(HaveOccurred())

		// 		By("checking that the taskrun status was updated correctly")
		// 		err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
		// 		Expect(err).NotTo(HaveOccurred())
		// 		Expect(updatedTaskRun.Status.Phase).To(Equal(v1alpha1.TaskRunPhaseToolCallsPending))
		// 		Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(3)) // System + User message + Assistant message with tool calls
		// 		Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls).To(HaveLen(1))
		// 		Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls[0].Function.Name).To(Equal("add"))
		// 		Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls[0].Function.Arguments).To(Equal(`{"a": 1, "b": 2}`))

		// 		By("checking that a TaskRunToolCall was created")
		// 		var taskRunToolCalls v1alpha1.TaskRunToolCallList
		// 		err = k8sClient.List(ctx, &taskRunToolCalls, client.InNamespace("default"))
		// 		Expect(err).NotTo(HaveOccurred())
		// 		Expect(taskRunToolCalls.Items).To(HaveLen(1))
		// 		trtc := taskRunToolCalls.Items[0]
		// 		Expect(trtc.Spec.ToolRef.Name).To(Equal("add"))
		// 		Expect(trtc.Spec.Arguments).To(Equal(`{"a": 1, "b": 2}`))
		// 	})
		// })
	})
})
