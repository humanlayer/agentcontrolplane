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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var fakeSpanContext = &acp.SpanContext{TraceID: "0123456789abcdef", SpanID: "fedcba9876543210"}

var testSecret = &utils.TestSecret{
	Name: "test-secret",
}

var testSlackContactChannel = &utils.TestContactChannel{
	Name:        "test-contact-channel",
	ChannelType: acp.ContactChannelTypeSlack,
	SecretName:  testSecret.Name,
}

var testMCPServer = &utils.TestMCPServer{
	Name:                   "test-mcp-server",
	ApprovalContactChannel: testSlackContactChannel.Name,
}

var _ = Describe("ToolCall Controller", func() {
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
			By("setting up test resources for MCP")
			testSecret.Setup(ctx, k8sClient)
			defer testSecret.Teardown(ctx)

			testSlackContactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer testSlackContactChannel.Teardown(ctx)

			testMCPServer.SetupWithStatus(ctx, k8sClient, acp.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer testMCPServer.Teardown(ctx)

			toolCall := &utils.TestToolCall{
				Name:     "test-mcp-tool-call",
				ToolRef:  testMCPServer.Name + "__fetch",
				TaskName: "task-party-2025",
				ToolType: acp.ToolTypeMCP,
			}
			tc := toolCall.SetupWithStatus(ctx, k8sClient, acp.ToolCallStatus{
				Phase:        acp.ToolCallPhasePending,
				Status:       acp.ToolCallStatusTypeReady,
				StatusDetail: "Setup complete",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:  fakeSpanContext,
			})
			defer toolCall.Teardown(ctx)

			By("reconciling the taskruntoolcall that uses MCP tool with approval")
			reconciler, recorder := reconciler()

			reconciler.MCPManager = &utils.MockMCPManager{
				NeedsApproval: true,
			}

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:  false,
				StatusCode:  200,
				ReturnError: nil,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second)) // Should requeue after 5 seconds

			By("checking the taskruntoolcall has AwaitingHumanApproval phase and Ready status")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseAwaitingHumanApproval))
			Expect(updatedTRTC.Status.Status).To(Equal(acp.ToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Waiting for human approval via contact channel"))

			_ = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			}, updatedTRTC)

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("AwaitingHumanApproval")
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseAwaitingHumanApproval))
		})
	})

	Context("Ready:Pending -> Ready:AwaitingHumanApproval (MCP Tool, Email Contact Channel)", func() {
		It("transitions to Ready:AwaitingHumanApproval when MCPServer has email approval channel", func() {
			By("setting up test resources for MCP")
			testSecret.Setup(ctx, k8sClient)
			defer testSecret.Teardown(ctx)

			testEmailContactChannel := &utils.TestContactChannel{
				Name:        "test-contact-channel",
				ChannelType: acp.ContactChannelTypeEmail,
				SecretName:  testSecret.Name,
			}

			testEmailContactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer testSlackContactChannel.Teardown(ctx)

			testMCPServer.SetupWithStatus(ctx, k8sClient, acp.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer testMCPServer.Teardown(ctx)

			toolCall := &utils.TestToolCall{
				Name:     "test-mcp-tool-call",
				ToolRef:  testMCPServer.Name + "__fetch",
				TaskName: "task-party-2025",
				ToolType: acp.ToolTypeMCP,
			}

			tc := toolCall.SetupWithStatus(ctx, k8sClient, acp.ToolCallStatus{
				Phase:        acp.ToolCallPhasePending,
				Status:       acp.ToolCallStatusTypeReady,
				StatusDetail: "Setup complete",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:  fakeSpanContext,
			})
			defer toolCall.Teardown(ctx)

			By("reconciling the taskruntoolcall that uses MCP tool with email approval")
			reconciler, recorder := reconciler()

			reconciler.MCPManager = &utils.MockMCPManager{
				NeedsApproval: true,
			}

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:  false,
				StatusCode:  200,
				ReturnError: nil,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second)) // Should requeue after 5 seconds

			By("checking the taskruntoolcall has AwaitingHumanApproval phase and Ready status")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
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
				Name:      testEmailContactChannel.Name,
				Namespace: "default",
			}, &contactChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(contactChannel.Spec.Type).To(Equal(acp.ContactChannelTypeEmail))
		})
	})

	Context("Ready:Pending -> Ready:AwaitingHumanInput (HumanContact Tool)", func() {
		It("transitions to Ready:AwaitingHumanInput when ToolType is HumanContact", func() {
			// Set up resources for human contact
			By("setting up test resources for MCP")
			testSecret.Setup(ctx, k8sClient)
			defer testSecret.Teardown(ctx)

			testSlackContactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer testSlackContactChannel.Teardown(ctx)

			testMCPServer.SetupWithStatus(ctx, k8sClient, acp.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer testMCPServer.Teardown(ctx)

			testHumanContactTool := &utils.TestToolCall{
				Name:     "test-human-contact-tool",
				ToolRef:  fmt.Sprintf("%s__%s", testSlackContactChannel.Name, "test-human-contact-tool"),
				TaskName: "task-party-2025",
				ToolType: acp.ToolTypeHumanContact,
			}

			tc := testHumanContactTool.SetupWithStatus(ctx, k8sClient, acp.ToolCallStatus{
				Phase:        acp.ToolCallPhasePending,
				Status:       acp.ToolCallStatusTypeReady,
				StatusDetail: "Setup complete",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:  fakeSpanContext,
			})
			defer testHumanContactTool.Teardown(ctx)

			By("reconciling the toolcall that uses HumanContact tool")
			reconciler, recorder := reconciler()

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:  false,
				StatusCode:  200,
				ReturnError: nil,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second)) // Should requeue after 5 seconds

			By("checking the toolcall has AwaitingHumanInput phase and Ready status")
			updatedToolCall := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			}, updatedToolCall)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedToolCall.Status.Phase).To(Equal(acp.ToolCallPhaseAwaitingHumanInput))
			Expect(updatedToolCall.Status.Status).To(Equal(acp.ToolCallStatusTypeReady))
			Expect(updatedToolCall.Status.StatusDetail).To(ContainSubstring("Waiting for human input via contact channel"))

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("AwaitingHumanContact")
		})
	})

	Context("Ready:Pending -> Error:ErrorRequestingHumanInput (HumanContact Tool)", func() {
		It("transitions to Error:ErrorRequestingHumanInput when request to HumanLayer for human contact fails", func() {
			// Set up resources for human contact
			By("setting up test resources for MCP")
			testSecret.Setup(ctx, k8sClient)
			defer testSecret.Teardown(ctx)

			testSlackContactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer testSlackContactChannel.Teardown(ctx)

			testMCPServer.SetupWithStatus(ctx, k8sClient, acp.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer testMCPServer.Teardown(ctx)

			testHumanContactTool := &utils.TestToolCall{
				Name:     "test-human-contact-tool",
				ToolRef:  fmt.Sprintf("%s__%s", testSlackContactChannel.Name, "test-human-contact-tool"),
				TaskName: "task-party-2025",
				ToolType: acp.ToolTypeHumanContact,
			}

			tc := testHumanContactTool.SetupWithStatus(ctx, k8sClient, acp.ToolCallStatus{
				Phase:        acp.ToolCallPhasePending,
				Status:       acp.ToolCallStatusTypeReady,
				StatusDetail: "Setup complete",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:  fakeSpanContext,
			})
			defer testHumanContactTool.Teardown(ctx)

			By("reconciling the toolcall that uses HumanContact tool with a failing API call")
			reconciler, _ := reconciler()

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:  true,
				StatusCode:  500,
				ReturnError: fmt.Errorf("failed to contact human: service unavailable"),
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the toolcall has ErrorRequestingHumanInput phase and Error status")
			updatedToolCall := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			}, updatedToolCall)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedToolCall.Status.Phase).To(Equal(acp.ToolCallPhaseErrorRequestingHumanInput))
			Expect(updatedToolCall.Status.Status).To(Equal(acp.ToolCallStatusTypeError))
		})
	})

	Context("Ready:AwaitingHumanApproval -> Ready:ReadyToExecuteApprovedTool", func() {
		It("transitions from Ready:AwaitingHumanApproval to Ready:ReadyToExecuteApprovedTool when MCP tool is approved", func() {
			By("setting up test resources for MCP")
			testSecret.Setup(ctx, k8sClient)
			defer testSecret.Teardown(ctx)

			testSlackContactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer testSlackContactChannel.Teardown(ctx)

			testMCPServer.SetupWithStatus(ctx, k8sClient, acp.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer testMCPServer.Teardown(ctx)

			toolCall := &utils.TestToolCall{
				Name:     "test-mcp-tool-call",
				ToolRef:  testMCPServer.Name + "__fetch",
				TaskName: "task-party-2025",
				ToolType: acp.ToolTypeMCP,
			}
			tc := toolCall.SetupWithStatus(ctx, k8sClient, acp.ToolCallStatus{
				ExternalCallID: "call-ready-to-execute-test",
				Phase:          acp.ToolCallPhaseAwaitingHumanApproval,
				Status:         acp.ToolCallStatusTypeReady,
				StatusDetail:   "Waiting for human approval via contact channel",
				StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:    fakeSpanContext,
			})
			defer toolCall.Teardown(ctx)

			By("reconciling the trtc against an approval-granting HumanLayer client")

			reconciler, _ := reconciler()

			reconciler.MCPManager = &utils.MockMCPManager{
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
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall status is set to ReadyToExecuteApprovedTool")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseReadyToExecuteApprovedTool))
			Expect(updatedTRTC.Status.Status).To(Equal(acp.ToolCallStatusTypeReady))
			Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Ready to execute approved tool"))
		})
	})

	Context("Ready:AwaitingHumanApproval -> Succeeded:ToolCallRejected", func() {
		It("transitions from Ready:AwaitingHumanApproval to Succeeded:ToolCallRejected when MCP tool is rejected", func() {
			By("setting up test resources for MCP")
			testSecret.Setup(ctx, k8sClient)
			defer testSecret.Teardown(ctx)

			testSlackContactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer testSlackContactChannel.Teardown(ctx)

			testMCPServer.SetupWithStatus(ctx, k8sClient, acp.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer testMCPServer.Teardown(ctx)

			toolCall := &utils.TestToolCall{
				Name:     "test-mcp-tool-call",
				ToolRef:  testMCPServer.Name + "__fetch",
				TaskName: "task-party-2025",
				ToolType: acp.ToolTypeMCP,
			}
			tc := toolCall.SetupWithStatus(ctx, k8sClient, acp.ToolCallStatus{
				ExternalCallID: "call-tool-call-rejected-test",
				Phase:          acp.ToolCallPhaseAwaitingHumanApproval,
				Status:         acp.ToolCallStatusTypeReady,
				StatusDetail:   "Waiting for human approval via contact channel",
				StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:    fakeSpanContext,
			})
			defer toolCall.Teardown(ctx)

			By("reconciling the trtc against an approval-rejecting HumanLayer client")

			reconciler, _ := reconciler()

			reconciler.MCPManager = &utils.MockMCPManager{
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
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall has ToolCallRejected phase and Succeeded status")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
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
			By("setting up test resources for MCP")
			testSecret.Setup(ctx, k8sClient)
			defer testSecret.Teardown(ctx)

			testSlackContactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer testSlackContactChannel.Teardown(ctx)

			testMCPServer.SetupWithStatus(ctx, k8sClient, acp.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer testMCPServer.Teardown(ctx)

			toolCall := &utils.TestToolCall{
				Name:     "test-mcp-tool-call",
				ToolRef:  testMCPServer.Name + "__fetch",
				TaskName: "task-party-2025",
				ToolType: acp.ToolTypeMCP,
			}
			tc := toolCall.SetupWithStatus(ctx, k8sClient, acp.ToolCallStatus{
				ExternalCallID: "call-ready-to-execute-test",
				Phase:          acp.ToolCallPhaseReadyToExecuteApprovedTool,
				Status:         acp.ToolCallStatusTypeReady,
				StatusDetail:   "Ready to execute tool, with great vigor",
				StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:    fakeSpanContext,
			})
			defer toolCall.Teardown(ctx)

			By("reconciling the trtc against an approval-granting HumanLayer client")

			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall status is set to Ready:Succeeded")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
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
			By("setting up test resources for MCP")
			testSecret.Setup(ctx, k8sClient)
			defer testSecret.Teardown(ctx)

			testSlackContactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer testSlackContactChannel.Teardown(ctx)

			testMCPServer.SetupWithStatus(ctx, k8sClient, acp.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer testMCPServer.Teardown(ctx)

			toolCall := &utils.TestToolCall{
				Name:     "test-mcp-tool-call",
				ToolRef:  testMCPServer.Name + "__fetch",
				TaskName: "task-party-2025",
				ToolType: acp.ToolTypeMCP,
			}
			tc := toolCall.SetupWithStatus(ctx, k8sClient, acp.ToolCallStatus{
				Phase:        acp.ToolCallPhasePending,
				Status:       acp.ToolCallStatusTypeReady,
				StatusDetail: "Setup complete",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				SpanContext:  fakeSpanContext,
			})
			defer toolCall.Teardown(ctx)

			By("reconciling the ToolCall that uses MCP tool with approval")
			reconciler, _ := reconciler()

			reconciler.MCPManager = &utils.MockMCPManager{
				NeedsApproval: true,
			}

			reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
				ShouldFail:  true,
				StatusCode:  500,
				ReturnError: fmt.Errorf("While taking pizzas from the kitchen to the lobby, Pete passed through the server room where he tripped over a network cable and now there's pizza all over the place. Also this request failed. No more pizza in the server room Pete."),
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall has ErrorRequestingHumanApproval phase and Error status")
			updatedTRTC := &acp.ToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(acp.ToolCallPhaseErrorRequestingHumanApproval))
			Expect(updatedTRTC.Status.Status).To(Equal(acp.ToolCallStatusTypeError))
		})
	})

})
