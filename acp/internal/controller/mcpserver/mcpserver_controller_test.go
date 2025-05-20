package mcpserver

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

func teardownMCPServer(ctx context.Context, mcpServer *acp.MCPServer) {
	Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, mcpServer))).To(Succeed())
}

func teardownContactChannel(ctx context.Context, contactChannel *acp.ContactChannel) {
	Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, contactChannel))).To(Succeed())
}

// MockMCPServerManager is a mock implementation of the MCPServerManager for testing
type MockMCPServerManager struct {
	ConnectServerFunc func(ctx context.Context, mcpServer *acp.MCPServer) error
	GetToolsFunc      func(serverName string) ([]acp.MCPTool, bool)
}

func (m *MockMCPServerManager) ConnectServer(ctx context.Context, mcpServer *acp.MCPServer) error {
	if m.ConnectServerFunc != nil {
		return m.ConnectServerFunc(ctx, mcpServer)
	}
	return nil
}

func (m *MockMCPServerManager) GetTools(serverName string) ([]acp.MCPTool, bool) {
	if m.GetToolsFunc != nil {
		return m.GetToolsFunc(serverName)
	}
	return nil, false
}

func (m *MockMCPServerManager) GetConnection(serverName string) (*mcpmanager.MCPConnection, bool) {
	return nil, false
}

func (m *MockMCPServerManager) DisconnectServer(serverName string) {
	// No-op for testing
}

func (m *MockMCPServerManager) GetToolsForAgent(agent *acp.Agent) []acp.MCPTool {
	return nil
}

func (m *MockMCPServerManager) CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (string, error) {
	return "", nil
}

func (m *MockMCPServerManager) FindServerForTool(fullToolName string) (serverName string, toolName string, found bool) {
	return "", "", false
}

func (m *MockMCPServerManager) Close() {
	// No-op for testing
}

var _ = Describe("MCPServer Controller", func() {
	const (
		MCPServerNamespace = "default"
	)

	Context("When reconciling a MCPServer", func() {
		It("Should validate and connect to the MCP server", func() {
			ctx := context.Background()

			// Create a test with a command that exists to avoid the command-not-found error
			testName := "test-mcpserver-echo"

			By("Creating a new MCPServer with a valid command")
			mcpServer := &acp.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: MCPServerNamespace,
					// Add labels to identify this as our test server
					Labels: map[string]string{
						"test": "true",
					},
				},
				Spec: acp.MCPServerSpec{
					Transport: "stdio",
					Command:   "sh", // This command exists
					Args:      []string{"-c", "echo test"},
				},
			}

			Expect(k8sClient.Create(ctx, mcpServer)).To(Succeed())
			defer teardownMCPServer(ctx, mcpServer)

			lookupKey := types.NamespacedName{Name: testName, Namespace: MCPServerNamespace}

			By("Setting up a mock MCPServer manager")
			mockManager := &MockMCPServerManager{
				ConnectServerFunc: func(ctx context.Context, mcpServer *acp.MCPServer) error {
					return nil // Return success
				},
				GetToolsFunc: func(serverName string) ([]acp.MCPTool, bool) {
					return []acp.MCPTool{
						{
							Name:        "test-tool",
							Description: "A test tool for validation",
						},
					}, true
				},
			}

			By("Creating a new test reconciler with our mock manager")
			reconciler := &MCPServerReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				recorder:   record.NewFakeRecorder(10),
				MCPManager: mockManager,
			}

			By("Performing reconciliation with our mock manager")
			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: lookupKey,
			})

			// This should succeed since our mock returns success
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute * 10)) // Successful requeue after 10 minutes

			// Create a validation test that uses the simpler approach - directly updating status
			testName2 := "test-mcpserver-direct"

			By("Creating a second MCPServer to test direct status updates")
			mcpServer2 := &acp.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName2,
					Namespace: MCPServerNamespace,
				},
				Spec: acp.MCPServerSpec{
					Transport: "stdio",
					Command:   "sh",
					Args:      []string{"-c", "echo test"},
				},
			}

			Expect(k8sClient.Create(ctx, mcpServer2)).To(Succeed())
			defer teardownMCPServer(ctx, mcpServer2)

			lookupKey2 := types.NamespacedName{Name: testName2, Namespace: MCPServerNamespace}

			// Wait for it to be created
			createdServer := &acp.MCPServer{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey2, createdServer)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			By("Directly updating the status")
			createdServer.Status.Connected = true
			createdServer.Status.Status = "Ready"
			createdServer.Status.StatusDetail = "Manually set status"
			createdServer.Status.Tools = []acp.MCPTool{
				{
					Name:        "manual-tool",
					Description: "A tool for testing",
				},
			}

			err = k8sClient.Status().Update(ctx, createdServer)
			Expect(err).NotTo(HaveOccurred())

			// Verify the direct update worked
			By("Verifying the status was properly updated")
			updatedServer := &acp.MCPServer{}
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, lookupKey2, updatedServer); err != nil {
					return false
				}
				return updatedServer.Status.Connected &&
					updatedServer.Status.Status == "Ready" &&
					len(updatedServer.Status.Tools) == 1
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			Expect(updatedServer.Status.Tools[0].Name).To(Equal("manual-tool"))
		})

		It("Should handle invalid MCP server specs", func() {
			ctx := context.Background()

			By("Creating a new MCPServer with invalid spec")
			invalidMCPServer := &acp.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-mcpserver",
					Namespace: MCPServerNamespace,
				},
				Spec: acp.MCPServerSpec{
					Transport: "stdio",
					// Missing command, which is required for stdio type
				},
			}

			Expect(k8sClient.Create(ctx, invalidMCPServer)).To(Succeed())
			defer teardownMCPServer(ctx, invalidMCPServer)

			invalidMCPServerLookupKey := types.NamespacedName{Name: "invalid-mcpserver", Namespace: MCPServerNamespace}
			createdInvalidMCPServer := &acp.MCPServer{}

			By("Verifying the invalid MCPServer was created")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, invalidMCPServerLookupKey, createdInvalidMCPServer)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			By("Creating a controller with a mock manager")
			reconciler := &MCPServerReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				recorder:   record.NewFakeRecorder(10),
				MCPManager: &MockMCPServerManager{},
			}

			By("Reconciling the invalid MCPServer")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: invalidMCPServerLookupKey,
			})
			Expect(err).To(HaveOccurred()) // Validation should fail

			By("Checking that the status was updated correctly to reflect the error")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, invalidMCPServerLookupKey, createdInvalidMCPServer)
				if err != nil {
					return false
				}
				return createdInvalidMCPServer.Status.Status == "Error"
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())
		})

		It("Should error if the approval contact channel is non-existent", func() {
			ctx := context.Background()

			By("Creating a new MCPServer with non-existent approval contact channel reference")
			mcpServer := &acp.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mcpserver-missing-channel",
					Namespace: MCPServerNamespace,
				},
				Spec: acp.MCPServerSpec{
					Transport: "stdio",
					Command:   "test-command",
					ApprovalContactChannel: &acp.LocalObjectReference{
						Name: "non-existent-channel",
					},
				},
			}

			Expect(k8sClient.Create(ctx, mcpServer)).To(Succeed())
			defer teardownMCPServer(ctx, mcpServer)

			By("Creating a controller with a mock manager")
			recorder := record.NewFakeRecorder(10)
			reconciler := &MCPServerReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				recorder:   recorder,
				MCPManager: &MockMCPServerManager{},
			}

			By("Reconciling the MCPServer with non-existent contact channel")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: "mcpserver-missing-channel", Namespace: MCPServerNamespace},
			})
			Expect(err).To(HaveOccurred()) // Should fail because contact channel doesn't exist

			By("Checking that the status was updated correctly to reflect the error")
			createdMCPServer := &acp.MCPServer{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "mcpserver-missing-channel", Namespace: MCPServerNamespace}, createdMCPServer)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdMCPServer.Status.Status).To(Equal("Error"))
			Expect(createdMCPServer.Status.StatusDetail).To(ContainSubstring("ContactChannel \"non-existent-channel\" not found"))
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ContactChannelNotFound")
		})

		It("Should stay in pending if the approval contact channel is not ready", func() {
			ctx := context.Background()

			By("Creating a new MCPServer with approval contact channel reference")
			mcpServer := &acp.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mcpserver-channel-ready",
					Namespace: MCPServerNamespace,
				},
				Spec: acp.MCPServerSpec{
					Transport: "stdio",
					Command:   "test-command",
					ApprovalContactChannel: &acp.LocalObjectReference{
						Name: "test-channel",
					},
				},
			}

			Expect(k8sClient.Create(ctx, mcpServer)).To(Succeed())
			defer teardownMCPServer(ctx, mcpServer)

			By("Creating the contact channel in not-ready state")
			contactChannel := &acp.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-channel",
					Namespace: MCPServerNamespace,
				},
				Spec: acp.ContactChannelSpec{
					Type: "slack",
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: "test-secret",
							Key:  "token",
						},
					},
					Slack: &acp.SlackChannelConfig{
						ChannelOrUserID: "C12345678",
					},
				},
				Status: acp.ContactChannelStatus{
					Ready:        false,
					Status:       "Pending",
					StatusDetail: "Initializing",
				},
			}
			Expect(k8sClient.Create(ctx, contactChannel)).To(Succeed())
			defer teardownContactChannel(ctx, contactChannel)

			By("Reconciling the MCPServer with not-ready contact channel")
			recorder := record.NewFakeRecorder(10)
			reconciler := &MCPServerReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				recorder:   recorder,
				MCPManager: &MockMCPServerManager{},
			}

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: "mcpserver-channel-ready", Namespace: MCPServerNamespace},
			})
			Expect(err).NotTo(HaveOccurred()) // Should stay in pending because contact channel is not ready
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("Checking that the status was updated to Pending")
			createdMCPServer := &acp.MCPServer{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "mcpserver-channel-ready", Namespace: MCPServerNamespace}, createdMCPServer)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdMCPServer.Status.Status).To(Equal("Pending"))
			Expect(createdMCPServer.Status.StatusDetail).To(ContainSubstring("ContactChannel \"test-channel\" is not ready"))
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ContactChannelNotReady")
		})
	})
})
