package agent

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

var llm = utils.TestLLM{
	Name:       "test-llm",
	SecretName: "fake-secret",
}

var _ = Describe("Agent Controller", func() {
	const resourceName = "test-agent"
	const llmName = "test-llm"

	ctx := context.Background()

	typeNamespacedName := types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}

	Context("'':'' -> Ready:Ready", func() {
		It("moves to Ready:Ready when all dependencies are valid", func() {
			llm.SetupWithStatus(ctx, k8sClient, acp.LLMStatus{
				Ready:        true,
				Status:       "Ready",
				StatusDetail: "Ready for testing",
			})
			defer llm.Teardown(ctx)

			By("creating a test contact channel")
			contactChannel := &utils.TestContactChannel{
				Name:        "test-humancontactchannel",
				ChannelType: acp.ContactChannelTypeEmail,
				SecretName:  "test-secret",
			}
			contactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:        true,
				Status:       "Ready",
				StatusDetail: "Ready for testing",
			})
			defer contactChannel.Teardown(ctx)

			By("creating the test agent")
			testAgent := &utils.TestAgent{
				Name:                 resourceName,
				SystemPrompt:         "Test agent",
				LLM:                  llmName,
				HumanContactChannels: []string{contactChannel.Name},
			}
			testAgent.Setup(ctx, k8sClient)
			defer testAgent.Teardown(ctx)

			By("reconciling the agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &AgentReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the agent status")
			updatedAgent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedAgent)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedAgent.Status.Ready).To(BeTrue())
			Expect(updatedAgent.Status.Status).To(Equal("Ready"))
			Expect(updatedAgent.Status.StatusDetail).To(Equal("All dependencies validated successfully"))
			Expect(updatedAgent.Status.ValidHumanContactChannels).To(ContainElement(acp.ResolvedContactChannel{
				Name: contactChannel.Name,
				Type: "email",
			}))

			By("checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})

	Context("'':'' -> Error:Error", func() {
		It("moves to Error:Error when LLM is not found", func() {
			By("creating a test contact channel")
			contactChannel := &utils.TestContactChannel{
				Name:        "test-humancontactchannel",
				ChannelType: acp.ContactChannelTypeEmail,
				SecretName:  "test-secret",
			}
			contactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:        true,
				Status:       "Ready",
				StatusDetail: "Ready for testing",
			})
			defer contactChannel.Teardown(ctx)

			By("creating the test agent with invalid LLM")
			testAgent := &utils.TestAgent{
				Name:                 resourceName,
				SystemPrompt:         "Test agent",
				LLM:                  "nonexistent-llm",
				HumanContactChannels: []string{contactChannel.Name},
			}
			testAgent.Setup(ctx, k8sClient)
			defer testAgent.Teardown(ctx)

			By("reconciling the agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &AgentReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`"nonexistent-llm" not found`))

			By("checking the agent status")
			updatedAgent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedAgent)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedAgent.Status.Ready).To(BeFalse())
			Expect(updatedAgent.Status.Status).To(Equal("Error"))
			Expect(updatedAgent.Status.StatusDetail).To(ContainSubstring(`"nonexistent-llm" not found`))

			By("checking that a failure event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationFailed")
		})

		It("moves to Error:Error when MCP server is not found", func() {
			By("creating a test LLM")
			llm.SetupWithStatus(ctx, k8sClient, acp.LLMStatus{
				Ready:        true,
				Status:       "Ready",
				StatusDetail: "Ready for testing",
			})
			defer llm.Teardown(ctx)

			By("creating a test contact channel")
			contactChannel := &utils.TestContactChannel{
				Name:        "test-humancontactchannel",
				ChannelType: acp.ContactChannelTypeEmail,
				SecretName:  "test-secret",
			}
			contactChannel.SetupWithStatus(ctx, k8sClient, acp.ContactChannelStatus{
				Ready:        true,
				Status:       "Ready",
				StatusDetail: "Ready for testing",
			})
			defer contactChannel.Teardown(ctx)

			By("creating the test agent with invalid MCP server")
			testAgent := &utils.TestAgent{
				Name:                 resourceName,
				SystemPrompt:         "Test agent",
				LLM:                  llmName,
				HumanContactChannels: []string{contactChannel.Name},
				MCPServers:           []string{"nonexistent-mcp-server"},
			}
			testAgent.Setup(ctx, k8sClient)
			defer testAgent.Teardown(ctx)

			By("reconciling the agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &AgentReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				recorder:   eventRecorder,
				MCPManager: &mcpmanager.MCPServerManager{},
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`"nonexistent-mcp-server" not found`))

			By("checking the agent status")
			updatedAgent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedAgent)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedAgent.Status.Ready).To(BeFalse())
			Expect(updatedAgent.Status.Status).To(Equal("Error"))
			Expect(updatedAgent.Status.StatusDetail).To(ContainSubstring(`"nonexistent-mcp-server" not found`))

			By("checking that a failure event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationFailed")
		})

		It("moves to Error:Error when contact channel is not found", func() {
			By("creating a test LLM")
			llm.SetupWithStatus(ctx, k8sClient, acp.LLMStatus{
				Ready:        true,
				Status:       "Ready",
				StatusDetail: "Ready for testing",
			})
			defer llm.Teardown(ctx)
			By("creating the test agent with invalid contact channel")
			testAgent := &utils.TestAgent{
				Name:                 resourceName,
				SystemPrompt:         "Test agent",
				LLM:                  llmName,
				HumanContactChannels: []string{"nonexistent-humancontactchannel"},
			}
			testAgent.Setup(ctx, k8sClient)
			defer testAgent.Teardown(ctx)

			By("reconciling the agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &AgentReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`"nonexistent-humancontactchannel" not found`))

			By("checking the agent status")
			updatedAgent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedAgent)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedAgent.Status.Ready).To(BeFalse())
			Expect(updatedAgent.Status.Status).To(Equal("Error"))
			Expect(updatedAgent.Status.StatusDetail).To(ContainSubstring(`"nonexistent-humancontactchannel" not found`))

			By("checking that a failure event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationFailed")
		})
	})

	Context("'':'' -> Pending:Pending", func() {
		It("moves to Pending:Pending when sub-agent is not ready", func() {
			By("creating a test LLM")
			llm.SetupWithStatus(ctx, k8sClient, acp.LLMStatus{
				Ready:        true,
				Status:       "Ready",
				StatusDetail: "Ready for testing",
			})
			defer llm.Teardown(ctx)
			By("creating the sub-agent first")
			subAgentObj := &utils.TestAgent{
				Name:         "sub-agent",
				SystemPrompt: "Sub agent",
				LLM:          llmName,
			}
			subAgentObj.SetupWithStatus(ctx, k8sClient, acp.AgentStatus{
				Ready:        false,
				Status:       "Pending",
				StatusDetail: "Not ready yet",
			})
			defer subAgentObj.Teardown(ctx)

			By("creating the parent agent with a reference to the sub-agent")
			parentAgentObj := &utils.TestAgent{
				Name:         resourceName,
				SystemPrompt: "Parent agent",
				LLM:          llmName,
				SubAgents:    []string{"sub-agent"},
				Description:  "A parent agent that delegates to sub-agents",
			}
			parentAgentObj.Setup(ctx, k8sClient)

			By("reconciling the parent agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &AgentReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second))

			By("checking the parent agent status")
			updatedParent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedParent)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedParent.Status.Ready).To(BeFalse())
			Expect(updatedParent.Status.Status).To(Equal("Pending"))
			Expect(updatedParent.Status.StatusDetail).To(ContainSubstring("waiting for sub-agent"))
			Expect(updatedParent.Status.StatusDetail).To(ContainSubstring("not ready"))
			Expect(updatedParent.Status.ValidSubAgents).To(BeEmpty())

			By("checking that a pending event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("SubAgentsPending")
		})
	})

	Context("Pending:Pending -> Ready:Ready", func() {
		FIt("moves to Ready:Ready when sub-agent becomes ready", func() {
			By("creating a test LLM")
			llm.SetupWithStatus(ctx, k8sClient, acp.LLMStatus{
				Ready:        true,
				Status:       "Ready",
				StatusDetail: "Ready for testing",
			})
			defer llm.Teardown(ctx)
			By("creating the sub-agent first")
			subAgentObj := &utils.TestAgent{
				Name:         "sub-agent",
				SystemPrompt: "Sub agent",
				LLM:          llmName,
			}
			subAgentObj.SetupWithStatus(ctx, k8sClient, acp.AgentStatus{
				Ready:        true,
				Status:       acp.AgentStatusReady,
				StatusDetail: "All dependencies validated successfully",
			})
			defer subAgentObj.Teardown(ctx)

			By("creating the parent agent with a reference to the sub-agent")
			parentAgentObj := &utils.TestAgent{
				Name:         resourceName,
				SystemPrompt: "Parent agent",
				LLM:          llmName,
				SubAgents:    []string{"sub-agent"},
				Description:  "A parent agent that delegates to sub-agents",
			}
			parentAgentObj.SetupWithStatus(ctx, k8sClient, acp.AgentStatus{
				Ready:        false,
				Status:       acp.AgentStatusPending,
				StatusDetail: "Waiting for sub-agent to become ready",
			})
			defer parentAgentObj.Teardown(ctx)

			By("reconciling the parent agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &AgentReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the parent agent status")
			updatedParent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedParent)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedParent.Status.Ready).To(BeTrue())
			Expect(updatedParent.Status.Status).To(Equal(acp.AgentStatusReady))
			Expect(updatedParent.Status.StatusDetail).To(Equal("All dependencies validated successfully"))
			Expect(updatedParent.Status.ValidSubAgents).To(ContainElement(acp.ResolvedSubAgent{
				Name: "sub-agent",
			}))

			By("checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})
})
