package toolcall

import (
	"fmt"
	"time"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayer"
	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("TaskRunToolCall Controller", func() {
	// Tests for DelegateToAgent tool
	Context("Ready:AwaitingSubAgent -> Error:Failed (DelegateToAgent Tool, Task Failed)", func() {
		It("transitions to Error:Failed when sub-agent task fails", func() {
			By("creating a ToolCall for delegate to agent already in AwaitingSubAgent phase")
			toolCall := &TestToolCall{
				name:      "test-delegate-tool-call-failed",
				toolName:  "delegate_to_agent__test-sub-agent",
				arguments: `{"message": "Please analyze this data"}`,
				toolType:  acp.ToolTypeDelegateToAgent,
			}

			// Set up with Ready:AwaitingSubAgent status
			tc := toolCall.SetupWithStatus(ctx, acp.ToolCallStatus{
				Status:       acp.ToolCallStatusTypeReady,
				Phase:        acp.ToolCallPhaseAwaitingSubAgent,
				StatusDetail: "Delegating to sub-agent test-sub-agent",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:  fakeSpanContext,
			})
			defer toolCall.Teardown(ctx)

			By("creating a failed child Task")
			childTask := &acp.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-child-task-failed",
					Namespace: "default",
					Labels: map[string]string{
						"acp.humanlayer.dev/parent-toolcall": tc.Name,
					},
				},
				Spec: acp.TaskSpec{
					AgentRef: acp.LocalObjectReference{
						Name: "test-sub-agent",
					},
					UserMessage: "Please analyze this data",
				},
				Status: acp.TaskStatus{
					Status:         acp.TaskStatusTypeError,
					Phase:          acp.TaskPhaseFailed,
					Error:          "Failed to analyze data: insufficient permissions",
					StartTime:      &metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
					CompletionTime: &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			}

			Expect(k8sClient.Create(ctx, childTask)).To(Succeed())
			Expect(k8sClient.Status().Update(ctx, childTask)).To(Succeed())
			defer func() { k8sClient.Delete(ctx, childTask) }()

			By("reconciling the ToolCall")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

			By("checking the ToolCall is updated to Error state")
			updatedTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			}, updatedTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTC.Status.Phase).To(Equal(acp.ToolCallPhaseFailed))
			Expect(updatedTC.Status.Status).To(Equal(acp.ToolCallStatusTypeError))
			Expect(updatedTC.Status.Error).To(Equal("Failed to analyze data: insufficient permissions"))
			Expect(updatedTC.Status.Result).To(ContainSubstring("Sub-agent task failed"))
			Expect(updatedTC.Status.CompletionTime).NotTo(BeNil())

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("SubAgentFailed")
		})
	})

	Context("Ready:AwaitingSubAgent -> Succeeded:Succeeded (DelegateToAgent Tool, Task Completed)", func() {
		It("transitions to Succeeded:Succeeded when sub-agent task completes", func() {
			By("creating a ToolCall for delegate to agent already in AwaitingSubAgent phase")
			toolCall := &TestToolCall{
				name:      "test-delegate-tool-call-completed",
				toolName:  "delegate_to_agent__test-sub-agent",
				arguments: `{"message": "Please analyze this data"}`,
				toolType:  acp.ToolTypeDelegateToAgent,
			}

			// Set up with Ready:AwaitingSubAgent status
			tc := toolCall.SetupWithStatus(ctx, acp.ToolCallStatus{
				Status:       acp.ToolCallStatusTypeReady,
				Phase:        acp.ToolCallPhaseAwaitingSubAgent,
				StatusDetail: "Delegating to sub-agent test-sub-agent",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:  fakeSpanContext,
			})
			defer toolCall.Teardown(ctx)

			By("creating a completed child Task")
			childTask := &acp.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-child-task",
					Namespace: "default",
					Labels: map[string]string{
						"acp.humanlayer.dev/parent-toolcall": tc.Name,
					},
				},
				Spec: acp.TaskSpec{
					AgentRef: acp.LocalObjectReference{
						Name: "test-sub-agent",
					},
					UserMessage: "Please analyze this data",
				},
				Status: acp.TaskStatus{
					Status:         acp.TaskStatusType,
					Phase:          acp.TaskPhaseFinalAnswer,
					Output:         "Analysis completed: The data shows significant patterns.",
					StartTime:      &metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
					CompletionTime: &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			}

			Expect(k8sClient.Create(ctx, childTask)).To(Succeed())
			Expect(k8sClient.Status().Update(ctx, childTask)).To(Succeed())
			defer func() { k8sClient.Delete(ctx, childTask) }()

			By("reconciling the ToolCall")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

			By("checking the ToolCall is updated to Succeeded state")
			updatedTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			}, updatedTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTC.Status.Phase).To(Equal(acp.ToolCallPhaseSucceeded))
			Expect(updatedTC.Status.Status).To(Equal(acp.ToolCallStatusTypeSucceeded))
			Expect(updatedTC.Status.Result).To(Equal("Analysis completed: The data shows significant patterns."))
			Expect(updatedTC.Status.CompletionTime).NotTo(BeNil())

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("SubAgentCompleted")
		})
	})

	Context("Ready:Pending -> Ready:AwaitingSubAgent (DelegateToAgent Tool)", func() {
		It("transitions to Ready:AwaitingSubAgent and creates a child Task", func() {
			By("creating a ToolCall for delegate to agent")
			toolCall := &TestToolCall{
				name:      "test-delegate-tool-call",
				toolName:  "delegate_to_agent__test-sub-agent",
				arguments: `{"message": "Please analyze this data"}`,
				toolType:  acp.ToolTypeDelegateToAgent,
			}

			// Set up with Ready:Pending status
			tc := toolCall.SetupWithStatus(ctx, acp.ToolCallStatus{
				Status:       acp.ToolCallStatusTypeReady,
				Phase:        acp.ToolCallPhasePending,
				StatusDetail: "Ready to execute",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:  fakeSpanContext,
			})
			defer toolCall.Teardown(ctx)

			By("reconciling the ToolCall")
			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			// Should requeue to check on child task status
			Expect(result.RequeueAfter).To(Equal(5 * time.Second))

			By("checking the ToolCall is updated to AwaitingSubAgent phase")
			updatedTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			}, updatedTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTC.Status.Phase).To(Equal(acp.ToolCallPhaseAwaitingSubAgent))
			Expect(updatedTC.Status.Status).To(Equal(acp.ToolCallStatusTypeReady))
			Expect(updatedTC.Status.StatusDetail).To(ContainSubstring("Delegating to sub-agent"))

			By("checking that a child Task was created")
			var childTasks acp.TaskList
			err = k8sClient.List(ctx, &childTasks,
				client.InNamespace(tc.Namespace),
				client.MatchingLabels{"acp.humanlayer.dev/parent-toolcall": tc.Name})

			Expect(err).NotTo(HaveOccurred())
			Expect(childTasks.Items).NotTo(BeEmpty())
			Expect(childTasks.Items[0].Spec.AgentRef.Name).To(Equal("test-sub-agent"))
			Expect(childTasks.Items[0].Spec.UserMessage).To(Equal("Please analyze this data"))

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("DelegatingToSubAgent")
		})
	})

	Context("'':'' -> Pending:Pending", func() {
		XIt("moves to Pending:Pending - need a non-builtin test here", func() {
		})
	})

	Context("Pending:Pending -> Ready:Pending", func() {
		XIt("moves from Pending:Pending to Ready:Pending during completeSetup - need a non-builtin test here", func() {
		})
	})

	Context("Ready:Pending -> Error:Pending", func() {
		XIt("fails when arguments are invalid - need a non-builtin test here", func() {})
	})

	// Tests for MCP tools without approval requirement
	Context("Pending:Pending -> Succeeded:Succeeded (MCP Tool)", func() {
		XIt("successfully executes an MCP tool without requiring approval - todo wth is an MCPTool we only have MCPServer when using MCP?", func() {})
	})

	// Tests for MCP tools with approval requirement
	Context("Ready:Pending -> Ready:AwaitingHumanApproval (MCP Tool, Slack Contact Channel)", func() {
		It("transitions to Ready:AwaitingHumanApproval when MCPServer has approval channel", func() {
			// Note setupTestApprovalResources sets up the MCP server, MCP tool, and TaskRunToolCall
			trtc, teardown := setupTestApprovalResources(ctx, nil)
			defer teardown()

			By("reconciling the taskruntoolcall that uses MCP tool with approval")
			reconciler, recorder := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: true,
			}

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:  false,
				StatusCode:  200,
				ReturnError: nil,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second)) // Should requeue after 5 seconds

			By("checking the taskruntoolcall has AwaitingHumanApproval phase and Ready status")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseAwaitingHumanApproval))
			Expect(updatedTRTC.Status.Status).To(Equal(acp.ToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Waiting for human approval via contact channel"))

			_ = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("AwaitingHumanApproval")
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseAwaitingHumanApproval))
		})
	})

	Context("Ready:Pending -> Ready:AwaitingHumanApproval (MCP Tool, Email Contact Channel)", func() {
		It("transitions to Ready:AwaitingHumanApproval when MCPServer has email approval channel", func() {
			// Set up resources with email contact channel
			trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
				ContactChannelType: "email",
			})
			defer teardown()

			By("reconciling the taskruntoolcall that uses MCP tool with email approval")
			reconciler, recorder := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: true,
			}

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:  false,
				StatusCode:  200,
				ReturnError: nil,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second)) // Should requeue after 5 seconds

			By("checking the taskruntoolcall has AwaitingHumanApproval phase and Ready status")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseAwaitingHumanApproval))
			Expect(updatedTRTC.Status.Status).To(Equal(acp.ToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Waiting for human approval via contact channel"))

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("AwaitingHumanApproval")
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseAwaitingHumanApproval))

			By("verifying the contact channel type is email")
			var contactChannel acp.ContactChannel
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      testContactChannel.name,
				Namespace: "default",
			}, &contactChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(contactChannel.Spec.Type).To(Equal(acp.ContactChannelTypeEmail))
		})
	})

	Context("Ready:AwaitingHumanApproval -> Ready:ReadyToExecuteApprovedTool", func() {
		It("transitions from Ready:AwaitingHumanApproval to Ready:ReadyToExecuteApprovedTool when MCP tool is approved", func() {
			trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
				ToolCallStatus: &acp.ToolCallStatus{
					ExternalCallID: "call-ready-to-execute-test",
					Phase:          acp.ToolCallPhaseAwaitingHumanApproval,
					Status:         acp.ToolCallStatusTypeReady,
					StatusDetail:   "Waiting for human approval via contact channel",
					StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			})
			defer teardown()

			By("reconciling the trtc against an approval-granting HumanLayer client")

			reconciler, _ := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: true,
			}

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:           false,
				StatusCode:           200,
				ReturnError:          nil,
				ShouldReturnApproval: true,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall status is set to ReadyToExecuteApprovedTool")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseReadyToExecuteApprovedTool))
			Expect(updatedTRTC.Status.Status).To(Equal(acp.ToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Ready to execute approved tool"))
		})
	})

	Context("Ready:AwaitingHumanApproval -> Succeeded:ToolCallRejected", func() {
		It("transitions from Ready:AwaitingHumanApproval to Succeeded:ToolCallRejected when MCP tool is rejected", func() {
			trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
				ToolCallStatus: &acp.ToolCallStatus{
					ExternalCallID: "call-tool-call-rejected-test",
					Phase:          acp.ToolCallPhaseAwaitingHumanApproval,
					Status:         acp.ToolCallStatusTypeReady,
					StatusDetail:   "Waiting for human approval via contact channel",
					StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			})
			defer teardown()

			By("reconciling the trtc against an approval-rejecting HumanLayer client")

			reconciler, _ := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: true,
			}

			rejectionComment := "You know what, I strongly disagree with this tool call and feel it should not be be given permission to execute. I, by the powers granted to me by The System, hereby reject it. If you too feel strongly, you can try again. I will reject it a second time, but with greater vigor."

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:            false,
				StatusCode:            200,
				ReturnError:           nil,
				ShouldReturnRejection: true,
				StatusComment:         rejectionComment,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall has ToolCallRejected phase and Succeeded status")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseToolCallRejected))
			Expect(updatedTRTC.Status.Status).To(Equal(acp.ToolCallStatusTypeSucceeded))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Tool execution rejected"))
			Expect(updatedTRTC.Status.Result).To(ContainSubstring(rejectionComment))
		})
	})

	Context("Ready:ReadyToExecuteApprovedTool -> Succeeded:Succeeded", func() {
		It("transitions from Ready:ReadyToExecuteApprovedTool to Succeeded:Succeeded when a tool is executed", func() {
			trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
				ToolCallStatus: &acp.ToolCallStatus{
					ExternalCallID: "call-ready-to-execute-test",
					Phase:          acp.ToolCallPhaseReadyToExecuteApprovedTool,
					Status:         acp.ToolCallStatusTypeReady,
					StatusDetail:   "Ready to execute tool, with great vigor",
					StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			})
			defer teardown()

			By("reconciling the trtc against an approval-granting HumanLayer client")

			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall status is set to Ready:Succeeded")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseSucceeded))
			Expect(updatedTRTC.Status.Status).To(Equal(acp.ToolCallStatusTypeSucceeded))
			Expect(updatedTRTC.Status.Result).To(Equal("5")) // From our mock implementation
		})
	})

	Context("Ready:Pending -> Error:ErrorRequestingHumanApproval (MCP Tool)", func() {
		It("transitions to Error:ErrorRequestingHumanApproval when request to HumanLayer fails", func() {
			// Note setupTestApprovalResources sets up the MCP server, MCP tool, and TaskRunToolCall
			trtc, teardown := setupTestApprovalResources(ctx, nil)
			defer teardown()

			By("reconciling the taskruntoolcall that uses MCP tool with approval")
			reconciler, _ := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: false,
			}

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:  true,
				StatusCode:  500,
				ReturnError: fmt.Errorf("While taking pizzas from the kitchen to the lobby, Pete passed through the server room where he tripped over a network cable and now there's pizza all over the place. Also this request failed. No more pizza in the server room Pete."),
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall has ErrorRequestingHumanApproval phase and Error status")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseErrorRequestingHumanApproval))
			Expect(updatedTRTC.Status.Status).To(Equal(acp.ToolCallStatusTypeError))
		})
	})
})
