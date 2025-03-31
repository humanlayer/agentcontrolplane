package taskruntoolcall

import (
	"context"
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
	Context("'' -> Pending", func() {
		It("moves to Pending:Initializing", func() {
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

	Context("Pending -> Succeeded", func() {
		It("moves to Succeeded after executing a simple function tool call", func() {
			ctx := context.Background()

			teardown := setupTestAddTool(ctx)
			defer teardown()

			// Create TaskRunToolCall with Pending status
			taskRunToolCall := trtcForAddTool.SetupWithStatus(ctx, kubechainv1alpha1.TaskRunToolCallStatus{
				Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
				Status:       "Pending",
				StatusDetail: "Ready for execution",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
			})

			By("reconciling the trtc")
			reconciler, recorder := reconciler()

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunToolCall.Name,
					Namespace: taskRunToolCall.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())

			By("checking the taskruntoolcall status has changed to Succeeded")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      taskRunToolCall.Name,
				Namespace: taskRunToolCall.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseSucceeded))
			Expect(updatedTRTC.Status.Result).To(Equal("5")) // 2 + 3 = 5
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Tool executed successfully"))

			By("checking that execution events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ExecutionSucceeded")
		})
	})

	Context("Pending -> Failed", func() {
		It("fails when arguments are invalid", func() {
			teardown := setupTestAddTool(ctx)
			defer teardown()

			// Create TaskRunToolCall with Pending status but invalid arguments
			taskRunToolCall := &TestTaskRunToolCall{
				name:      "invalid-args-trtc",
				toolName:  addTool.name,
				arguments: `invalid json`, // Invalid JSON
			}

			trtc := taskRunToolCall.SetupWithStatus(ctx, kubechainv1alpha1.TaskRunToolCallStatus{
				Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
				Status:       "Pending",
				StatusDetail: "Ready for execution",
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
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Invalid arguments JSON"))
			Expect(updatedTRTC.Status.Error).NotTo(BeEmpty())

			By("checking that error events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ExecutionFailed")
		})
	})

	// Tests for MCP tools without approval requirement
	Context("Pending -> Succeeded (MCP Tool)", func() {
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
				Status:       "Pending",
				StatusDetail: "Ready for execution",
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
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.Result).To(Equal("5")) // From our mock implementation

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ExecutionSucceeded")
		})
	})

	// Tests for approval workflow
	Context("Pending -> AwaitingHumanApproval (MCP Tool) -> Ready:Succeeded", func() {
		It("transitions to AwaitingHumanApproval when MCPServer has approval channel", func() {
			// Note setupTestApprovalResources sets up the MCP server, MCP tool, and TaskRunToolCall
			trtc, teardown := setupTestApprovalResources(ctx, nil)
			defer teardown()

			By("reconciling the taskruntoolcall that uses MCP tool with approval")
			reconciler, recorder := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: true,
			}

			reconciler.HLClient = &humanlayer.MockHumanLayerClient{
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

			By("checking the taskruntoolcall status is set to AwaitingHumanApproval")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhasePending))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeAwaitingHumanApproval))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Waiting for human approval via contact channel"))

			_ = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("AwaitingHumanApproval")
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeAwaitingHumanApproval))

			// By("reconciling the taskrunttoolcall again to check if it is approved")
			// reconciler.MCPManager = &MockMCPManager{
			// 	NeedsApproval: false,
			// }
			// result, err = reconciler.Reconcile(ctx, reconcile.Request{
			// 	NamespacedName: types.NamespacedName{
			// 		Name:      trtc.Name,
			// 		Namespace: trtc.Namespace,
			// 	},
			// })

			// Expect(err).NotTo(HaveOccurred())
			// Expect(result.Requeue).To(BeFalse())

			// By("checking the taskruntoolcall status is set to Ready:Succeeded")
			// updatedTRTC = &kubechainv1alpha1.TaskRunToolCall{}
			// err = k8sClient.Get(ctx, types.NamespacedName{
			// 	Name:      trtc.Name,
			// 	Namespace: trtc.Namespace,
			// }, updatedTRTC)

			// Expect(err).NotTo(HaveOccurred())
			// Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseSucceeded))
			// Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeReady))
			// Expect(updatedTRTC.Status.Result).To(Equal("5")) // From our mock implementation
		})
	})

	Context("AwaitingHumanApproval -> ReadyToExecuteApprovedTool", func() {
		It("transitions from AwaitingHumanApproval to ReadyToExecuteApprovedTool when MCP tool is approved", func() {
			trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
				TaskRunToolCallStatus: &kubechainv1alpha1.TaskRunToolCallStatus{
					Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
					Status:       kubechainv1alpha1.TaskRunToolCallStatusTypeAwaitingHumanApproval,
					StatusDetail: "Waiting for human approval via contact channel",
					StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			})
			defer teardown()

			By("reconciling the trtc against an approval-granting HumanLayer client")

			reconciler, _ := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: true,
			}

			reconciler.HLClient = &humanlayer.MockHumanLayerClient{
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
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhasePending))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeReadyToExecuteApprovedTool))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Ready to execute approved tool"))
		})
	})

	Context("AwaitingHumanApproval -> ToolCallRejected", func() {
		It("transitions from AwaitingHumanApproval to ToolCallRejected when MCP tool is rejected", func() {
			trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
				TaskRunToolCallStatus: &kubechainv1alpha1.TaskRunToolCallStatus{
					Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
					Status:       kubechainv1alpha1.TaskRunToolCallStatusTypeAwaitingHumanApproval,
					StatusDetail: "Waiting for human approval via contact channel",
					StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			})
			defer teardown()

			By("reconciling the trtc against an approval-rejecting HumanLayer client")

			reconciler, _ := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: true,
			}

			reconciler.HLClient = &humanlayer.MockHumanLayerClient{
				ShouldFail:            false,
				StatusCode:            200,
				ReturnError:           nil,
				ShouldReturnRejection: true,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall status is set to ToolCallRejected")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseFailed))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeToolCallRejected))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Tool execution rejected"))
		})
	})

	Context("Pending -> ErrorRequestingHumanApproval (MCP Tool)", func() {
		It("transitions to ErrorRequestingHumanApproval when request to HumanLayer fails", func() {
			// Note setupTestApprovalResources sets up the MCP server, MCP tool, and TaskRunToolCall
			trtc, teardown := setupTestApprovalResources(ctx, nil)
			defer teardown()

			By("reconciling the taskruntoolcall that uses MCP tool with approval")
			reconciler, _ := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: false,
			}

			reconciler.HLClient = &humanlayer.MockHumanLayerClient{
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

			By("checking the taskruntoolcall status is set to ErrorRequestingHumanApproval")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhasePending))
			Expect(updatedTRTC.Status.Status).To(Equal(kubechainv1alpha1.TaskRunToolCallStatusTypeErrorRequestingHumanApproval))
		})
	})
})
