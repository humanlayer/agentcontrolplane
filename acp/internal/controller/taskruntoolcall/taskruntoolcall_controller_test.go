package taskruntoolcall

import (
	"context" // Added context import
	"fmt"
	"time"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayer"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi" // Added import
	// "github.com/humanlayer/agentcontrolplane/acp/test/utils" // Commented out unused import
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client" // Added client import
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("TaskRunToolCall Controller", func() {
	Context("'':'' -> Pending:Pending", func() {
		It("moves to Pending:Pending", func() {
			// Create the test tool resource
			addTool := createTestTool("add-tool-init") // Use helper from utils_test.go
			Setup(ctx, k8sClient, addTool)             // Create the tool
			defer Teardown(ctx, k8sClient, addTool)    // Ensure cleanup

			// Setup the tool with ready status using the updated function
			setupTestAddTool(ctx, k8sClient, addTool, true) // Pass client, tool, and ready=true

			// Create the TaskRunToolCall that uses this tool
			trtcForAddTool := createTestTaskRunToolCall("trtc-init", addTool.Name, acp.ToolType(""),
				map[string]interface{}{"a": 1, "b": 2})
			Setup(ctx, k8sClient, trtcForAddTool)
			defer Teardown(ctx, k8sClient, trtcForAddTool)

			By("reconciling the taskruntoolcall")
			// Create a reconciler instance for this test
			testReconciler := reconciler(k8sClient, &MockMCPManager{}, SetupTestApprovalConfig(true, "", nil)) // Use test helper

			result, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtcForAddTool.Name,
					Namespace: trtcForAddTool.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			// Expect Requeue to be false because the status update should trigger the next reconcile
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeZero()) // Explicitly check RequeueAfter is zero

			By("checking the taskruntoolcall status was initialized")
			updatedTRTC := &acp.TaskRunToolCall{}
			// Use Eventually to wait for the status update
			Eventually(func(g Gomega) {
				errGet := k8sClient.Get(ctx, types.NamespacedName{
					Name:      trtcForAddTool.Name,
					Namespace: trtcForAddTool.Namespace,
				}, updatedTRTC)
				g.Expect(errGet).NotTo(HaveOccurred())
				g.Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhasePending))
				g.Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypePending))
				g.Expect(updatedTRTC.Status.StatusDetail).To(Equal("Initializing"))
				g.Expect(updatedTRTC.Status.StartTime).NotTo(BeNil())
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Pending:Pending -> Ready:Pending", func() {
		It("moves from Pending:Pending to Ready:Pending during completeSetup", func() {
			// Create the test tool resource
			addTool := createTestTool("add-tool-ready")
			Setup(ctx, k8sClient, addTool)
			defer Teardown(ctx, k8sClient, addTool)
			setupTestAddTool(ctx, k8sClient, addTool, true) // Ensure tool is ready

			// Create TaskRunToolCall with Pending:Pending status
			trtcForAddTool := createTestTaskRunToolCall("trtc-ready", addTool.Name, acp.ToolType(""),
				map[string]interface{}{"a": 1, "b": 2})
			SetupWithStatus(ctx, k8sClient, trtcForAddTool, func(obj client.Object) {
				trtc := obj.(*acp.TaskRunToolCall)
				trtc.Status = acp.TaskRunToolCallStatus{
					Phase:        acp.TaskRunToolCallPhasePending,
					Status:       acp.TaskRunToolCallStatusTypePending,
					StatusDetail: "Initializing",
					StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				}
			})
			defer Teardown(ctx, k8sClient, trtcForAddTool)

			By("reconciling the taskruntoolcall")
			testReconciler := reconciler(k8sClient, &MockMCPManager{}, SetupTestApprovalConfig(true, "", nil))

			result, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtcForAddTool.Name,
					Namespace: trtcForAddTool.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse()) // No requeue expected after setup completion
			Expect(result.RequeueAfter).To(BeZero())

			By("checking the taskruntoolcall status has changed to Ready:Pending")
			updatedTRTC := &acp.TaskRunToolCall{}
			Eventually(func(g Gomega) {
				errGet := k8sClient.Get(ctx, types.NamespacedName{
					Name:      trtcForAddTool.Name,
					Namespace: trtcForAddTool.Namespace,
				}, updatedTRTC)
				g.Expect(errGet).NotTo(HaveOccurred())
				g.Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhasePending))     // Phase remains Pending
				g.Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypeReady)) // Status becomes Ready
				g.Expect(updatedTRTC.Status.StatusDetail).To(Equal("Setup complete"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Ready:Pending -> Error:Pending", func() {
		It("fails when arguments are invalid", func() {
			// Create the test tool resource
			addTool := createTestTool("add-tool-invalid-args")
			Setup(ctx, k8sClient, addTool)
			defer Teardown(ctx, k8sClient, addTool)
			setupTestAddTool(ctx, k8sClient, addTool, true)

			// Create TaskRunToolCall with Ready:Pending status but invalid arguments
			trtcForInvalidArgs := createTestTaskRunToolCall("trtc-invalid-args", addTool.Name, acp.ToolType(""),
				nil) // Create first
			trtcForInvalidArgs.Spec.Arguments = "invalid json" // Set invalid args
			SetupWithStatus(ctx, k8sClient, trtcForInvalidArgs, func(obj client.Object) {
				trtc := obj.(*acp.TaskRunToolCall)
				trtc.Status = acp.TaskRunToolCallStatus{
					Phase:        acp.TaskRunToolCallPhasePending,    // Start in Pending phase
					Status:       acp.TaskRunToolCallStatusTypeReady, // But Ready status
					StatusDetail: "Setup complete",
					StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				}
			})
			defer Teardown(ctx, k8sClient, trtcForInvalidArgs)

			By("reconciling the taskruntoolcall with invalid arguments")
			testReconciler := reconciler(k8sClient, &MockMCPManager{}, SetupTestApprovalConfig(true, "", nil))

			_, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtcForInvalidArgs.Name,
					Namespace: trtcForInvalidArgs.Namespace,
				},
			})

			// We expect an error during reconciliation because parseArguments returns an error
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid character")) // Check for JSON parse error

			By("checking the taskruntoolcall status is set to Error")
			updatedTRTC := &acp.TaskRunToolCall{}
			Eventually(func(g Gomega) {
				errGet := k8sClient.Get(ctx, types.NamespacedName{
					Name:      trtcForInvalidArgs.Name,
					Namespace: trtcForInvalidArgs.Namespace,
				}, updatedTRTC)
				g.Expect(errGet).NotTo(HaveOccurred())
				g.Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypeError))
				g.Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhaseFailed)) // Should move to Failed phase
				g.Expect(updatedTRTC.Status.StatusDetail).To(Equal(DetailInvalidArgsJSON))
				g.Expect(updatedTRTC.Status.Error).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())

			// By("checking that error events were emitted")
			// utils.ExpectRecorder(testReconciler.recorder).ToEmitEventContaining("ExecutionFailed") // Check recorder on reconciler
		})
	})

	// Tests for MCP tools without approval requirement
	Context("Pending:Pending -> Succeeded:Succeeded (MCP Tool)", func() {
		It("successfully executes an MCP tool without requiring approval", func() {
			// Setup MCP server without approval channel
			mcpServerNoApproval := createTestMCPServer("mcp-no-approval", nil)
			SetupWithStatus(ctx, k8sClient, mcpServerNoApproval, func(obj client.Object) {
				mcp := obj.(*acp.MCPServer)
				mcp.Status = acp.MCPServerStatus{Status: "Ready", StatusDetail: "Ready"}
			})
			defer Teardown(ctx, k8sClient, mcpServerNoApproval)

			// Setup MCP tool associated with this server
			mcpTool := createTestMCPTool(mcpServerNoApproval.Name, "add")
			setupTestAddTool(ctx, k8sClient, mcpTool, true) // Use setupTestAddTool for consistency
			defer Teardown(ctx, k8sClient, mcpTool)

			// Create TaskRunToolCall with MCP tool reference
			trtcMCP := createTestTaskRunToolCall("trtc-mcp-no-approval", mcpTool.Name, acp.ToolTypeMCP, map[string]interface{}{"a": 2, "b": 3})
			SetupWithStatus(ctx, k8sClient, trtcMCP, func(obj client.Object) {
				trtc := obj.(*acp.TaskRunToolCall)
				trtc.Status = acp.TaskRunToolCallStatus{
					Phase:        acp.TaskRunToolCallPhasePending,
					Status:       acp.TaskRunToolCallStatusTypeReady, // Start as Ready:Pending
					StatusDetail: "Setup complete",
					StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				}
			})
			defer Teardown(ctx, k8sClient, trtcMCP)

			By("reconciling the taskruntoolcall that uses MCP tool without approval")
			mockMCPMgr := &MockMCPManager{
				CallToolFunc: func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (string, error) {
					Expect(serverName).To(Equal(mcpServerNoApproval.Name))
					Expect(toolName).To(Equal("add")) // The actual tool name part
					a := args["a"].(float64)
					b := args["b"].(float64)
					return fmt.Sprintf("%v", a+b), nil
				},
			}
			testReconciler := reconciler(k8sClient, mockMCPMgr, SetupTestApprovalConfig(true, "", nil))

			_, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtcMCP.Name,
					Namespace: trtcMCP.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())

			By("checking the taskruntoolcall status is set to Succeeded")
			updatedTRTC := &acp.TaskRunToolCall{}
			Eventually(func(g Gomega) {
				errGet := k8sClient.Get(ctx, types.NamespacedName{
					Name:      trtcMCP.Name,
					Namespace: trtcMCP.Namespace,
				}, updatedTRTC)
				g.Expect(errGet).NotTo(HaveOccurred())
				g.Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhaseSucceeded))
				g.Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypeSucceeded))
				g.Expect(updatedTRTC.Status.Result).To(Equal("5")) // From our mock implementation
			}, timeout, interval).Should(Succeed())

			// By("checking that appropriate events were emitted")
			// utils.ExpectRecorder(testReconciler.recorder).ToEmitEventContaining("ExecutionSucceeded")
		})
	})

	// Tests for MCP tools with approval requirement
	Context("Ready:Pending -> Ready:AwaitingHumanApproval (MCP Tool, Slack Contact Channel)", func() {
		It("transitions to Ready:AwaitingHumanApproval when MCPServer has approval channel", func() {
			baseName := "approval-slack"
			secret, contactChannel, mcpServer := setupTestApprovalResources(ctx, k8sClient, baseName)
			defer Teardown(ctx, k8sClient, mcpServer)
			defer Teardown(ctx, k8sClient, contactChannel)
			defer Teardown(ctx, k8sClient, secret)

			mcpTool := createTestMCPTool(mcpServer.Name, "needs-approval")
			setupTestAddTool(ctx, k8sClient, mcpTool, true)
			defer Teardown(ctx, k8sClient, mcpTool)

			trtcApproval := createTestTaskRunToolCall("trtc-"+baseName, mcpTool.Name, acp.ToolTypeMCP, map[string]interface{}{"action": "deploy"})
			SetupWithStatus(ctx, k8sClient, trtcApproval, func(obj client.Object) {
				trtc := obj.(*acp.TaskRunToolCall)
				trtc.Status = acp.TaskRunToolCallStatus{
					Phase:        acp.TaskRunToolCallPhasePending,
					Status:       acp.TaskRunToolCallStatusTypeReady,
					StatusDetail: "Setup complete",
					StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				}
			})
			defer Teardown(ctx, k8sClient, trtcApproval)

			By("reconciling the taskruntoolcall that uses MCP tool with approval")
			var generatedCallID string
			mockHLFactory := SetupTestApprovalConfig(true, "", &generatedCallID) // Setup mock HL client
			testReconciler := reconciler(k8sClient, &MockMCPManager{}, mockHLFactory)

			result, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtcApproval.Name,
					Namespace: trtcApproval.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second)) // Should requeue after requesting approval

			By("checking the taskruntoolcall has AwaitingHumanApproval phase and Ready status")
			updatedTRTC := &acp.TaskRunToolCall{}
			Eventually(func(g Gomega) {
				errGet := k8sClient.Get(ctx, types.NamespacedName{
					Name:      trtcApproval.Name,
					Namespace: trtcApproval.Namespace,
				}, updatedTRTC)
				g.Expect(errGet).NotTo(HaveOccurred())
				g.Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhaseAwaitingHumanApproval))
				g.Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypeReady)) // Status remains Ready
				g.Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Waiting for human approval via contact channel"))
				g.Expect(updatedTRTC.Status.ExternalCallID).To(Equal(generatedCallID)) // Check CallID was stored
			}, timeout, interval).Should(Succeed())

			// By("checking that appropriate events were emitted")
			// utils.ExpectRecorder(testReconciler.recorder).ToEmitEventContaining("AwaitingHumanApproval")
			// utils.ExpectRecorder(testReconciler.recorder).ToEmitEventContaining("HumanLayerRequestSent")
		})
	})

	// Context("Ready:Pending -> Ready:AwaitingHumanApproval (MCP Tool, Email Contact Channel)", func() {
	// 	It("transitions to Ready:AwaitingHumanApproval when MCPServer has email approval channel", func() {
	// 		// Set up resources with email contact channel
	// 		trtc, teardown := setupTestApprovalResources(ctx, &SetupTestApprovalConfig{
	// 			ContactChannelType: "email",
	// 		})
	// 		defer teardown()

	// 		By("reconciling the taskruntoolcall that uses MCP tool with email approval")
	// 		reconciler, recorder := reconciler()

	// 		reconciler.MCPManager = &MockMCPManager{
	// 			NeedsApproval: true,
	// 		}

	// 		reconciler.HLClientFactory = &humanlayer.MockHumanLayerClientFactory{
	// 			ShouldFail:  false,
	// 			StatusCode:  200,
	// 			ReturnError: nil,
	// 		}

	// 		result, err := reconciler.Reconcile(ctx, reconcile.Request{
	// 			NamespacedName: types.NamespacedName{
	// 				Name:      trtc.Name,
	// 				Namespace: trtc.Namespace,
	// 			},
	// 		})

	// 		Expect(err).NotTo(HaveOccurred())
	// 		Expect(result.RequeueAfter).To(Equal(5 * time.Second)) // Should requeue after 5 seconds

	// 		By("checking the taskruntoolcall has AwaitingHumanApproval phase and Ready status")
	// 		updatedTRTC := &acp.TaskRunToolCall{}
	// 		err = k8sClient.Get(ctx, types.NamespacedName{
	// 			Name:      trtc.Name,
	// 			Namespace: trtc.Namespace,
	// 		}, updatedTRTC)

	// 		Expect(err).NotTo(HaveOccurred())
	// 		Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhaseAwaitingHumanApproval))
	// 		Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypeReady))
	// 		Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("Waiting for human approval via contact channel"))

	// 		By("checking that appropriate events were emitted")
	// 		utils.ExpectRecorder(recorder).ToEmitEventContaining("AwaitingHumanApproval")
	// 		Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhaseAwaitingHumanApproval))

	// 		By("verifying the contact channel type is email")
	// 		var contactChannel acp.ContactChannel
	// 		err = k8sClient.Get(ctx, types.NamespacedName{
	// 			Name:      testContactChannel.name,
	// 			Namespace: "default",
	// 		}, &contactChannel)
	// 		Expect(err).NotTo(HaveOccurred())
	// 		Expect(contactChannel.Spec.Type).To(Equal(acp.ContactChannelTypeEmail))
	// 	})
	// })

	Context("Ready:AwaitingHumanApproval -> Ready:ReadyToExecuteApprovedTool", func() {
		It("transitions from Ready:AwaitingHumanApproval to Ready:ReadyToExecuteApprovedTool when MCP tool is approved", func() {
			baseName := "approval-granted"
			secret, contactChannel, mcpServer := setupTestApprovalResources(ctx, k8sClient, baseName)
			defer Teardown(ctx, k8sClient, mcpServer)
			defer Teardown(ctx, k8sClient, contactChannel)
			defer Teardown(ctx, k8sClient, secret)

			mcpTool := createTestMCPTool(mcpServer.Name, "approved-tool")
			setupTestAddTool(ctx, k8sClient, mcpTool, true)
			defer Teardown(ctx, k8sClient, mcpTool)

			initialCallID := "call-id-for-approval"
			trtcApproval := createTestTaskRunToolCall("trtc-"+baseName, mcpTool.Name, acp.ToolTypeMCP, map[string]interface{}{"action": "proceed"})
			SetupWithStatus(ctx, k8sClient, trtcApproval, func(obj client.Object) {
				trtc := obj.(*acp.TaskRunToolCall)
				trtc.Status = acp.TaskRunToolCallStatus{
					ExternalCallID: initialCallID, // Set the call ID from the initial request
					Phase:          acp.TaskRunToolCallPhaseAwaitingHumanApproval,
					Status:         acp.TaskRunToolCallStatusTypeReady,
					StatusDetail:   "Waiting for human approval via contact channel",
					StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				}
			})
			defer Teardown(ctx, k8sClient, trtcApproval)

			By("reconciling the trtc against an approval-granting HumanLayer client")
			mockHLFactory := SetupTestApprovalConfig(true, "Looks good!", nil) // Should approve
			testReconciler := reconciler(k8sClient, &MockMCPManager{}, mockHLFactory)

			// Set the CallID on the mock client wrapper if the factory provides access
			// This might require adjusting SetupTestApprovalConfig or the mock structure
			// if hlFactory.(*humanlayer.MockHumanLayerClientFactory).MockClientWrapper != nil {
			// 	hlFactory.(*humanlayer.MockHumanLayerClientFactory).MockClientWrapper.CallID = initialCallID
			// }

			result, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtcApproval.Name,
					Namespace: trtcApproval.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse()) // Should not requeue after successful status check
			Expect(result.RequeueAfter).To(BeZero())

			By("checking the taskruntoolcall status is set to ReadyToExecuteApprovedTool")
			updatedTRTC := &acp.TaskRunToolCall{}
			Eventually(func(g Gomega) {
				errGet := k8sClient.Get(ctx, types.NamespacedName{
					Name:      trtcApproval.Name,
					Namespace: trtcApproval.Namespace,
				}, updatedTRTC)
				g.Expect(errGet).NotTo(HaveOccurred())
				g.Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhaseReadyToExecuteApprovedTool))
				g.Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypeReady)) // Status remains Ready
				g.Expect(updatedTRTC.Status.StatusDetail).To(Equal("Human approval received"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Ready:AwaitingHumanApproval -> Succeeded:ToolCallRejected", func() {
		It("transitions from Ready:AwaitingHumanApproval to Succeeded:ToolCallRejected when MCP tool is rejected", func() {
			baseName := "approval-rejected"
			secret, contactChannel, mcpServer := setupTestApprovalResources(ctx, k8sClient, baseName)
			defer Teardown(ctx, k8sClient, mcpServer)
			defer Teardown(ctx, k8sClient, contactChannel)
			defer Teardown(ctx, k8sClient, secret)

			mcpTool := createTestMCPTool(mcpServer.Name, "rejected-tool")
			setupTestAddTool(ctx, k8sClient, mcpTool, true)
			defer Teardown(ctx, k8sClient, mcpTool)

			initialCallID := "call-id-for-rejection"
			rejectionComment := "Nope, not allowed."
			trtcRejection := createTestTaskRunToolCall("trtc-"+baseName, mcpTool.Name, acp.ToolTypeMCP, map[string]interface{}{"action": "dangerous-op"})
			SetupWithStatus(ctx, k8sClient, trtcRejection, func(obj client.Object) {
				trtc := obj.(*acp.TaskRunToolCall)
				trtc.Status = acp.TaskRunToolCallStatus{
					ExternalCallID: initialCallID,
					Phase:          acp.TaskRunToolCallPhaseAwaitingHumanApproval,
					Status:         acp.TaskRunToolCallStatusTypeReady,
					StatusDetail:   "Waiting for human approval",
					StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				}
			})
			defer Teardown(ctx, k8sClient, trtcRejection)

			By("reconciling the trtc against an approval-rejecting HumanLayer client")
			mockHLFactory := SetupTestApprovalConfig(false, rejectionComment, nil) // Should reject
			testReconciler := reconciler(k8sClient, &MockMCPManager{}, mockHLFactory)

			result, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtcRejection.Name,
					Namespace: trtcRejection.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse()) // Terminal state

			By("checking the taskruntoolcall has ToolCallRejected phase and Succeeded status")
			updatedTRTC := &acp.TaskRunToolCall{}
			Eventually(func(g Gomega) {
				errGet := k8sClient.Get(ctx, types.NamespacedName{
					Name:      trtcRejection.Name,
					Namespace: trtcRejection.Namespace,
				}, updatedTRTC)
				g.Expect(errGet).NotTo(HaveOccurred())
				g.Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhaseToolCallRejected))
				g.Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypeSucceeded)) // Status is Succeeded (rejection processed)
				g.Expect(updatedTRTC.Status.StatusDetail).To(Equal("Tool execution rejected by human"))
				g.Expect(updatedTRTC.Status.Result).To(ContainSubstring(rejectionComment)) // Result contains comment
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Ready:ReadyToExecuteApprovedTool -> Succeeded:Succeeded", func() {
		It("transitions from Ready:ReadyToExecuteApprovedTool to Succeeded:Succeeded when a tool is executed", func() {
			baseName := "execute-approved"
			secret, contactChannel, mcpServer := setupTestApprovalResources(ctx, k8sClient, baseName)
			defer Teardown(ctx, k8sClient, mcpServer)
			defer Teardown(ctx, k8sClient, contactChannel)
			defer Teardown(ctx, k8sClient, secret)

			mcpTool := createTestMCPTool(mcpServer.Name, "add-approved")
			setupTestAddTool(ctx, k8sClient, mcpTool, true)
			defer Teardown(ctx, k8sClient, mcpTool)

			trtcExec := createTestTaskRunToolCall("trtc-"+baseName, mcpTool.Name, acp.ToolTypeMCP, map[string]interface{}{"a": 10, "b": 5})
			SetupWithStatus(ctx, k8sClient, trtcExec, func(obj client.Object) {
				trtc := obj.(*acp.TaskRunToolCall)
				trtc.Status = acp.TaskRunToolCallStatus{
					ExternalCallID: "call-id-approved-for-exec",
					Phase:          acp.TaskRunToolCallPhaseReadyToExecuteApprovedTool, // Start in this phase
					Status:         acp.TaskRunToolCallStatusTypeReady,
					StatusDetail:   "Human approval received",
					StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				}
			})
			defer Teardown(ctx, k8sClient, trtcExec)

			By("reconciling the trtc ready for approved execution")
			mockMCPMgr := &MockMCPManager{
				CallToolFunc: func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (string, error) {
					Expect(serverName).To(Equal(mcpServer.Name))
					Expect(toolName).To(Equal("add-approved"))
					a := args["a"].(float64)
					b := args["b"].(float64)
					return fmt.Sprintf("%v", a+b), nil // Simulate successful execution
				},
			}
			// HL Client doesn't matter here as approval is already done
			testReconciler := reconciler(k8sClient, mockMCPMgr, SetupTestApprovalConfig(true, "", nil))

			result, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtcExec.Name,
					Namespace: trtcExec.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse()) // Terminal state

			By("checking the taskruntoolcall status is set to Succeeded:Succeeded")
			updatedTRTC := &acp.TaskRunToolCall{}
			Eventually(func(g Gomega) {
				errGet := k8sClient.Get(ctx, types.NamespacedName{
					Name:      trtcExec.Name,
					Namespace: trtcExec.Namespace,
				}, updatedTRTC)
				g.Expect(errGet).NotTo(HaveOccurred())
				g.Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhaseSucceeded))
				g.Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypeSucceeded))
				g.Expect(updatedTRTC.Status.Result).To(Equal("15")) // Check result from mock MCP call
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Ready:Pending -> Error:ErrorRequestingHumanApproval (MCP Tool)", func() {
		It("transitions to Error:ErrorRequestingHumanApproval when request to HumanLayer fails", func() {
			baseName := "approval-hl-fail"
			secret, contactChannel, mcpServer := setupTestApprovalResources(ctx, k8sClient, baseName)
			defer Teardown(ctx, k8sClient, mcpServer)
			defer Teardown(ctx, k8sClient, contactChannel)
			defer Teardown(ctx, k8sClient, secret)

			mcpTool := createTestMCPTool(mcpServer.Name, "hl-fail-tool")
			setupTestAddTool(ctx, k8sClient, mcpTool, true)
			defer Teardown(ctx, k8sClient, mcpTool)

			trtcHLFail := createTestTaskRunToolCall("trtc-"+baseName, mcpTool.Name, acp.ToolTypeMCP, map[string]interface{}{"action": "risky"})
			SetupWithStatus(ctx, k8sClient, trtcHLFail, func(obj client.Object) {
				trtc := obj.(*acp.TaskRunToolCall)
				trtc.Status = acp.TaskRunToolCallStatus{
					Phase:        acp.TaskRunToolCallPhasePending,
					Status:       acp.TaskRunToolCallStatusTypeReady,
					StatusDetail: "Setup complete",
					StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				}
			})
			defer Teardown(ctx, k8sClient, trtcHLFail)

			By("reconciling the taskruntoolcall against a failing HumanLayer client")
			errorMsg := "HumanLayer API unavailable"
			mockHLClient := &humanlayer.MockHumanLayerClientWrapper{
				RequestApprovalFunc: func(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
					return nil, 500, fmt.Errorf(errorMsg) // Simulate API failure
				},
			}
			mockHLFactory := &humanlayer.MockHumanLayerClientFactory{
				NewHumanLayerClientFunc: func() humanlayer.HumanLayerClientWrapper { return mockHLClient },
			}
			testReconciler := reconciler(k8sClient, &MockMCPManager{}, mockHLFactory)

			result, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtcHLFail.Name,
					Namespace: trtcHLFail.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred()) // The reconcile loop itself shouldn't error, status is updated
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskruntoolcall has ErrorRequestingHumanApproval phase and Error status")
			updatedTRTC := &acp.TaskRunToolCall{}
			Eventually(func(g Gomega) {
				errGet := k8sClient.Get(ctx, types.NamespacedName{
					Name:      trtcHLFail.Name,
					Namespace: trtcHLFail.Namespace,
				}, updatedTRTC)
				g.Expect(errGet).NotTo(HaveOccurred())
				g.Expect(updatedTRTC.Status.Phase).To(Equal(acp.TaskRunToolCallPhaseErrorRequestingHumanApproval))
				g.Expect(updatedTRTC.Status.Status).To(Equal(acp.TaskRunToolCallStatusTypeError))
				g.Expect(updatedTRTC.Status.StatusDetail).To(ContainSubstring("HumanLayer request failed"))
				g.Expect(updatedTRTC.Status.Error).To(ContainSubstring(errorMsg))
			}, timeout, interval).Should(Succeed())
		})
	})
})
