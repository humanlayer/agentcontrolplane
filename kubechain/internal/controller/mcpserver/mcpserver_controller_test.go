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

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/mcpmanager"
	"github.com/humanlayer/smallchain/kubechain/test/utils"
)

// MockMCPServerManager is a mock implementation of the MCPServerManager for testing
type MockMCPServerManager struct {
	ConnectServerFunc func(ctx context.Context, mcpServer *kubechainv1alpha1.MCPServer) error
	GetToolsFunc      func(serverName string) ([]kubechainv1alpha1.MCPTool, bool)
}

func (m *MockMCPServerManager) ConnectServer(ctx context.Context, mcpServer *kubechainv1alpha1.MCPServer) error {
	if m.ConnectServerFunc != nil {
		return m.ConnectServerFunc(ctx, mcpServer)
	}
	return nil
}

func (m *MockMCPServerManager) GetTools(serverName string) ([]kubechainv1alpha1.MCPTool, bool) {
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

func (m *MockMCPServerManager) GetToolsForAgent(agent *kubechainv1alpha1.Agent) []kubechainv1alpha1.MCPTool {
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
		MCPServerName      = "test-mcpserver"
		MCPServerNamespace = "default"
	)

	Context("When reconciling a MCPServer", func() {
		It("Should validate and connect to the MCP server", func() {
			ctx := context.Background()

			By("Creating a new MCPServer")
			mcpServer := &kubechainv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      MCPServerName,
					Namespace: MCPServerNamespace,
				},
				Spec: kubechainv1alpha1.MCPServerSpec{
					Transport: "stdio",
					Command:   "test-command",
					Args:      []string{"--arg1", "value1"},
					Env: []kubechainv1alpha1.EnvVar{
						{
							Name:  "TEST_ENV",
							Value: "test-value",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, mcpServer)).To(Succeed())

			mcpServerLookupKey := types.NamespacedName{Name: MCPServerName, Namespace: MCPServerNamespace}
			createdMCPServer := &kubechainv1alpha1.MCPServer{}

			By("Verifying the MCPServer was created")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, mcpServerLookupKey, createdMCPServer)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			By("Setting up a mock MCPServerManager")
			mockManager := &MockMCPServerManager{
				ConnectServerFunc: func(ctx context.Context, mcpServer *kubechainv1alpha1.MCPServer) error {
					return nil // Simulate successful connection
				},
				GetToolsFunc: func(serverName string) ([]kubechainv1alpha1.MCPTool, bool) {
					return []kubechainv1alpha1.MCPTool{
						{
							Name:        "test-tool",
							Description: "A test tool",
						},
					}, true
				},
			}

			By("Creating a controller with the mock manager")
			reconciler := &MCPServerReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				recorder:   record.NewFakeRecorder(10),
				MCPManager: mockManager,
			}

			By("Reconciling the created MCPServer")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: mcpServerLookupKey,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the status was updated correctly")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, mcpServerLookupKey, createdMCPServer)
				if err != nil {
					return false
				}
				return createdMCPServer.Status.Connected &&
					len(createdMCPServer.Status.Tools) == 1 &&
					createdMCPServer.Status.Status == "Ready"
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			By("Cleaning up the MCPServer")
			Expect(k8sClient.Delete(ctx, mcpServer)).To(Succeed())
		})

		It("Should handle invalid MCP server specs", func() {
			ctx := context.Background()

			By("Creating a new MCPServer with invalid spec")
			invalidMCPServer := &kubechainv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-mcpserver",
					Namespace: MCPServerNamespace,
				},
				Spec: kubechainv1alpha1.MCPServerSpec{
					Transport: "stdio",
					// Missing command, which is required for stdio type
				},
			}

			Expect(k8sClient.Create(ctx, invalidMCPServer)).To(Succeed())

			invalidMCPServerLookupKey := types.NamespacedName{Name: "invalid-mcpserver", Namespace: MCPServerNamespace}
			createdInvalidMCPServer := &kubechainv1alpha1.MCPServer{}

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
				return !createdInvalidMCPServer.Status.Connected &&
					createdInvalidMCPServer.Status.Status == "Error"
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			By("Cleaning up the invalid MCPServer")
			Expect(k8sClient.Delete(ctx, invalidMCPServer)).To(Succeed())
		})

		It("Should error if the approval contact channel is non-existent", func() {
			ctx := context.Background()

			By("Creating a new MCPServer with non-existent approval contact channel reference")
			mcpServer := &kubechainv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mcpserver-missing-channel",
					Namespace: MCPServerNamespace,
				},
				Spec: kubechainv1alpha1.MCPServerSpec{
					Transport: "stdio",
					Command:   "test-command",
					ApprovalContactChannel: &kubechainv1alpha1.LocalObjectReference{
						Name: "non-existent-channel",
					},
				},
			}

			Expect(k8sClient.Create(ctx, mcpServer)).To(Succeed())

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
			createdMCPServer := &kubechainv1alpha1.MCPServer{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "mcpserver-missing-channel", Namespace: MCPServerNamespace}, createdMCPServer)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdMCPServer.Status.Connected).To(BeFalse())
			Expect(createdMCPServer.Status.Status).To(Equal("Error"))
			Expect(createdMCPServer.Status.StatusDetail).To(ContainSubstring("ContactChannel \"non-existent-channel\" not found"))
			By("Checking that the event was emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ContactChannelNotFound")

			By("Cleaning up the MCPServer")
			Expect(k8sClient.Delete(ctx, mcpServer)).To(Succeed())
		})

		It("Should stay in pending if the approval contact channel is not ready", func() {
			ctx := context.Background()

			By("Creating a new MCPServer with approval contact channel reference")
			mcpServer := &kubechainv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mcpserver-channel-ready",
					Namespace: MCPServerNamespace,
				},
				Spec: kubechainv1alpha1.MCPServerSpec{
					Transport: "stdio",
					Command:   "test-command",
					ApprovalContactChannel: &kubechainv1alpha1.LocalObjectReference{
						Name: "test-channel",
					},
				},
			}

			Expect(k8sClient.Create(ctx, mcpServer)).To(Succeed())

			By("Creating the contact channel in not-ready state")
			contactChannel := &kubechainv1alpha1.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-channel",
					Namespace: MCPServerNamespace,
				},
				Status: kubechainv1alpha1.ContactChannelStatus{
					Ready:        false,
					Status:       "Pending",
					StatusDetail: "Initializing",
				},
			}
			Expect(k8sClient.Create(ctx, contactChannel)).To(Succeed())

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
			createdMCPServer := &kubechainv1alpha1.MCPServer{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "mcpserver-channel-ready", Namespace: MCPServerNamespace}, createdMCPServer)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdMCPServer.Status.Status).To(Equal("Pending"))
			Expect(createdMCPServer.Status.StatusDetail).To(ContainSubstring("ContactChannel \"test-channel\" is not ready"))
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ContactChannelNotReady")

			By("Cleaning up the MCPServer and ContactChannel")
			Expect(k8sClient.Delete(ctx, mcpServer)).To(Succeed())
			Expect(k8sClient.Delete(ctx, contactChannel)).To(Succeed())
		})
	})
})
