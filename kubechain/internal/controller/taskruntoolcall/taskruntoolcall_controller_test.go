package taskruntoolcall

import (
	"fmt"
	"time"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/humanlayer"
	"github.com/humanlayer/smallchain/kubechain/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("TaskRunToolCall Controller", func() {
	Context("'':'' -> Pending:Pending", func() {
		It("moves to Pending:Pending", func() {
			teardown := setupTestAddTool(ctx)
			defer teardown()

			taskRunToolCall := trtcForAddTool.Setup(ctx)

			By("reconciling the taskruntoolcall")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunToolCall.Name,
					Namespace: taskRunToolCall.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse()) // No requeue since initialization is complete

			By("checking the taskruntoolcall status was initialized")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      taskRunToolCall.Name,
				Namespace: taskRunToolCall.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhasePending))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypePending))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Initializing"))
			Expect(updatedTRTC.Status.StartTime).NotTo(BeNil())
		})
	})

	Context("Pending:Pending -> Ready:Pending", func() {
		It("moves from Pending:Pending to Ready:Pending during completeSetup", func() {
			teardown := setupTestAddTool(ctx)
			defer teardown()

			// Create TaskRunToolCall with Pending status (after initialization)
			taskRunToolCall := trtcForAddTool.SetupWithStatus(ctx, kubechainv1alpha1.TaskRunToolCallStatus{
				Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
				Status:       kubechainv1alpha1.TaskRunToolCallStatusTypePending,
				StatusDetail: "Initializing",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
			})

			By("reconciling the taskruntoolcall")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunToolCall.Name,
					Namespace: taskRunToolCall.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall status has changed to Ready:Pending")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      taskRunToolCall.Name,
				Namespace: taskRunToolCall.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhasePending))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Setup complete"))
		})
	})

	Context("Ready:Pending -> Error:Pending", func() {
		It("fails when arguments are invalid", func() {
			teardown := setupTestAddTool(ctx)
			defer teardown()

			// Create TaskRunToolCall with Ready:Pending status but invalid arguments
			taskRunToolCall := &TestTaskRunToolCall{
				name:      "invalid-args-trtc",
				toolName:  addTool.name,
				arguments: `invalid json`, // Invalid JSON
			}

			trtc := taskRunToolCall.SetupWithStatus(ctx, kubechainv1alpha1.TaskRunToolCallStatus{
				Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
				Status:       kubechainv1alpha1.TaskRunToolCallStatusTypeReady,
				StatusDetail: "Setup complete",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
			})

			defer taskRunToolCall.Teardown(ctx)

			By("reconciling the taskruntoolcall with invalid arguments")
			reconciler, recorder := reconciler()

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			// We expect an error during reconciliation
			Expect(err).To(HaveOccurred())

			By("checking the taskruntoolcall status is set to Error")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeError))
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseFailed))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Invalid arguments JSON"))
			Expect(updatedTRTC.Status.Error).NotTo(BeEmpty())

			By("checking that error events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ExecutionFailed")
		})
	})

	// Tests for MCP tools without approval requirement
	Context("Pending:Pending -> Succeeded:Succeeded (MCP Tool)", func() {
		It("successfully executes an MCP tool without requiring approval", func() {
			// Setup MCP server without approval channel
			testSecret.Setup(ctx)
			mcpServer := &TestMCPServer{
				name:                   "test-mcp-no-approval",
				needsApproval:          false,
				approvalContactChannel: "",
			}
			mcpServer.SetupWithStatus(ctx, kubechainv1alpha1.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer mcpServer.Teardown(ctx)

			// Setup MCP tool
			mcpTool := &TestMCPTool{
				name:        "test-mcp-no-approval-tool",
				mcpServer:   mcpServer.name,
				mcpToolName: "test-tool",
			}
			tool := mcpTool.SetupWithStatus(ctx, kubechainv1alpha1.ToolStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer mcpTool.Teardown(ctx)

			// Create TaskRunToolCall with MCP tool reference
			taskRunToolCall := &TestTaskRunToolCall{
				name:      "test-mcp-no-approval-trtc",
				toolName:  tool.Spec.Name,
				arguments: `{"a": 2, "b": 3}`,
			}
			trtc := taskRunToolCall.SetupWithStatus(ctx, kubechainv1alpha1.TaskRunToolCallStatus{
				Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
				Status:       kubechainv1alpha1.TaskRunToolCallStatusTypeReady,
				StatusDetail: "Setup complete",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
			})
			defer taskRunToolCall.Teardown(ctx)

			By("reconciling the taskruntoolcall that uses MCP tool without approval")
			reconciler, recorder := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: false, // This test specifically doesn't want approval
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())

			By("checking the taskruntoolcall status is set to Succeeded")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseSucceeded))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeSucceeded))
			Expect(updatedTRTC.Status.Result).To(Equal("5")) // From our mock implementation

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ExecutionSucceeded")
		})
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
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanApproval))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Waiting for human approval via contact channel"))

			_ = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("AwaitingHumanApproval")
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanApproval))
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
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanApproval))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Waiting for human approval via contact channel"))

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("AwaitingHumanApproval")
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanApproval))

			By("verifying the contact channel type is email")
			var contactChannel kubechainv1alpha1.ContactChannel
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      testContactChannel.name,
				Namespace: "default",
			}, &contactChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(contactChannel.Spec.Type).To(Equal(kubechainv1alpha1.ContactChannelTypeEmail))
		})
	})

	Context("Ready:AwaitingHumanApproval -> Ready:ReadyToExecuteApprovedTool", func() {
		It("transitions from Ready:AwaitingHumanApproval to Ready:ReadyToExecuteApprovedTool when MCP tool is approved", func() {
			trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
				TaskRunToolCallStatus: &kubechainv1alpha1.TaskRunToolCallStatus{
					ExternalCallID: "call-ready-to-execute-test",
					Phase:          kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanApproval,
					Status:         kubechainv1alpha1.TaskRunToolCallStatusTypeReady,
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
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseReadyToExecuteApprovedTool))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Ready to execute approved tool"))
		})
	})

	Context("Ready:AwaitingHumanApproval -> Succeeded:ToolCallRejected", func() {
		It("transitions from Ready:AwaitingHumanApproval to Succeeded:ToolCallRejected when MCP tool is rejected", func() {
			trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
				TaskRunToolCallStatus: &kubechainv1alpha1.TaskRunToolCallStatus{
					ExternalCallID: "call-tool-call-rejected-test",
					Phase:          kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanApproval,
					Status:         kubechainv1alpha1.TaskRunToolCallStatusTypeReady,
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
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseToolCallRejected))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeSucceeded))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Tool execution rejected"))
			Expect(updatedTRTC.Status.Result).To(Equal(rejectionComment))
		})
	})

	Context("Ready:ReadyToExecuteApprovedTool -> Succeeded:Succeeded", func() {
		It("transitions from Ready:ReadyToExecuteApprovedTool to Succeeded:Succeeded when a tool is executed", func() {
			trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
				TaskRunToolCallStatus: &kubechainv1alpha1.TaskRunToolCallStatus{
					ExternalCallID: "call-ready-to-execute-test",
					Phase:          kubechainv1alpha1.TaskRunToolCallPhaseReadyToExecuteApprovedTool,
					Status:         kubechainv1alpha1.TaskRunToolCallStatusTypeReady,
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
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseSucceeded))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeSucceeded))
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
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseErrorRequestingHumanApproval))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeError))
		})
	})
})
