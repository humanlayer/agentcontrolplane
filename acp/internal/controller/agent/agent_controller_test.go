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
	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

const (
	llmName   = "test-llm"
	agentName = "test-agent"
)

var llm = utils.TestLLM{
	Name:       llmName,
	SecretName: "fake-secret",
}

var _ = Describe("Agent Controller", func() {
	ctx := context.Background()

	typeNamespacedName := types.NamespacedName{
		Name:      agentName,
		Namespace: "default",
	}

	Context("StateMachine Tests", func() {
		Describe("'' -> Ready:Ready", func() {
			It("initializes agent status and validates dependencies successfully", func() {
				By("setting up required dependencies")
				llm.SetupWithStatus(ctx, k8sClient, acp.LLMStatus{
					Ready:        true,
					Status:       "Ready",
					StatusDetail: "Ready for testing",
				})
				defer llm.Teardown(ctx)

				By("creating a test agent with empty status")
				testAgent := &utils.TestAgent{
					Name:         agentName,
					SystemPrompt: "Test agent",
					LLM:          llmName,
				}
				testAgent.Setup(ctx, k8sClient)
				defer testAgent.Teardown(ctx)

				By("getting the agent to verify empty status")
				agent := &acp.Agent{}
				err := k8sClient.Get(ctx, typeNamespacedName, agent)
				Expect(err).NotTo(HaveOccurred())
				Expect(agent.Status.Status).To(BeEmpty())

				By("processing with state machine")
				eventRecorder := record.NewFakeRecorder(10)
				stateMachine := NewStateMachine(k8sClient, eventRecorder)

				_, err = stateMachine.Process(ctx, agent)
				Expect(err).NotTo(HaveOccurred())

				By("verifying status transitions to Ready")
				err = k8sClient.Get(ctx, typeNamespacedName, agent)
				Expect(err).NotTo(HaveOccurred())
				Expect(agent.Status.Status).To(Equal(acp.AgentStatusReady))
				Expect(agent.Status.StatusDetail).To(Equal("All dependencies validated successfully"))

				By("checking that events were created")
				utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("Initializing")
				utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
			})
		})

		Describe("Pending:Pending -> Ready:Ready", func() {
			It("validates dependencies and transitions to Ready", func() {
				By("setting up all required dependencies")
				llm.SetupWithStatus(ctx, k8sClient, acp.LLMStatus{
					Ready:        true,
					Status:       "Ready",
					StatusDetail: "Ready for testing",
				})
				defer llm.Teardown(ctx)

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

				By("creating a test agent with pending status")
				testAgent := &utils.TestAgent{
					Name:                 agentName,
					SystemPrompt:         "Test agent",
					LLM:                  llmName,
					HumanContactChannels: []string{contactChannel.Name},
				}
				testAgent.SetupWithStatus(ctx, k8sClient, acp.AgentStatus{
					Status:       acp.AgentStatusPending,
					StatusDetail: "Validating dependencies",
					Ready:        false,
				})
				defer testAgent.Teardown(ctx)

				By("processing with state machine")
				eventRecorder := record.NewFakeRecorder(10)
				stateMachine := NewStateMachine(k8sClient, eventRecorder)

				agent := &acp.Agent{}
				err := k8sClient.Get(ctx, typeNamespacedName, agent)
				Expect(err).NotTo(HaveOccurred())

				_, err = stateMachine.Process(ctx, agent)
				Expect(err).NotTo(HaveOccurred())

				By("verifying status transitions to Ready")
				err = k8sClient.Get(ctx, typeNamespacedName, agent)
				Expect(err).NotTo(HaveOccurred())
				Expect(agent.Status.Status).To(Equal(acp.AgentStatusReady))
				Expect(agent.Status.Ready).To(BeTrue())
				Expect(agent.Status.StatusDetail).To(Equal("All dependencies validated successfully"))
				Expect(agent.Status.ValidHumanContactChannels).To(ContainElement(acp.ResolvedContactChannel{
					Name: contactChannel.Name,
					Type: "email",
				}))

				By("checking that a success event was created")
				utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
			})
		})
	})

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
				Name:                 agentName,
				SystemPrompt:         "Test agent",
				LLM:                  llmName,
				HumanContactChannels: []string{contactChannel.Name},
			}
			testAgent.Setup(ctx, k8sClient)
			defer testAgent.Teardown(ctx)

			By("reconciling the agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := NewTestAgentReconciler(k8sClient, eventRecorder)

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the agent status")
			updatedAgent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedAgent)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedAgent.Status.Ready).To(BeTrue())
			Expect(updatedAgent.Status.Status).To(Equal(acp.AgentStatusReady))
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
				Name:                 agentName,
				SystemPrompt:         "Test agent",
				LLM:                  "nonexistent-llm",
				HumanContactChannels: []string{contactChannel.Name},
			}
			testAgent.Setup(ctx, k8sClient)
			defer testAgent.Teardown(ctx)

			By("reconciling the agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := NewTestAgentReconciler(k8sClient, eventRecorder)

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
			Expect(updatedAgent.Status.Status).To(Equal(acp.AgentStatusError))
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
				Name:                 agentName,
				SystemPrompt:         "Test agent",
				LLM:                  llmName,
				HumanContactChannels: []string{contactChannel.Name},
				MCPServers:           []string{"nonexistent-mcp-server"},
			}
			testAgent.Setup(ctx, k8sClient)
			defer testAgent.Teardown(ctx)

			By("reconciling the agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := NewTestAgentReconciler(k8sClient, eventRecorder)

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
			Expect(updatedAgent.Status.Status).To(Equal(acp.AgentStatusError))
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
				Name:                 agentName,
				SystemPrompt:         "Test agent",
				LLM:                  llmName,
				HumanContactChannels: []string{"nonexistent-humancontactchannel"},
			}
			testAgent.Setup(ctx, k8sClient)
			defer testAgent.Teardown(ctx)

			By("reconciling the agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := NewTestAgentReconciler(k8sClient, eventRecorder)

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
			Expect(updatedAgent.Status.Status).To(Equal(acp.AgentStatusError))
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
				Name:         agentName,
				SystemPrompt: "Parent agent",
				LLM:          llmName,
				SubAgents:    []string{"sub-agent"},
				Description:  "A parent agent that delegates to sub-agents",
			}
			parentAgentObj.Setup(ctx, k8sClient)
			defer parentAgentObj.Teardown(ctx)

			By("reconciling the parent agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := NewTestAgentReconciler(k8sClient, eventRecorder)

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
			Expect(updatedParent.Status.Status).To(Equal(acp.AgentStatusPending))
			Expect(updatedParent.Status.StatusDetail).To(ContainSubstring("waiting for sub-agent"))
			Expect(updatedParent.Status.StatusDetail).To(ContainSubstring("not ready"))
			Expect(updatedParent.Status.ValidSubAgents).To(BeEmpty())

			By("checking that a pending event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("SubAgentsPending")
		})

		It("moves to Ready:Ready when MCP server is connected with tools", func() {
			By("creating a test LLM")
			llm.SetupWithStatus(ctx, k8sClient, acp.LLMStatus{
				Ready:        true,
				Status:       "Ready",
				StatusDetail: "Ready for testing",
			})
			defer llm.Teardown(ctx)

			By("creating a connected MCP server")
			mcpServer := &utils.TestMCPServer{
				Name:      "test-fetch",
				Transport: "stdio",
				Command:   "uvx",
				Args:      []string{"mcp-server-fetch"},
			}
			mcpServer.SetupWithStatus(ctx, k8sClient, acp.MCPServerStatus{
				Connected:    true,
				Status:       "Ready",
				StatusDetail: "Connected successfully with 1 tools",
				Tools: []acp.MCPTool{{
					Name:        "fetch",
					Description: "Fetch a URL",
				}},
			})
			defer mcpServer.Teardown(ctx)

			By("creating a test agent with MCP server reference")
			testAgent := &utils.TestAgent{
				Name:         agentName,
				SystemPrompt: "Test agent",
				LLM:          llmName,
				MCPServers:   []string{"test-fetch"},
			}
			testAgent.Setup(ctx, k8sClient)
			defer testAgent.Teardown(ctx)

			By("reconciling the agent")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := NewTestAgentReconciler(k8sClient, eventRecorder)

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the agent status")
			updatedAgent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedAgent)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedAgent.Status.Ready).To(BeTrue())
			Expect(updatedAgent.Status.Status).To(Equal(acp.AgentStatusReady))
			Expect(updatedAgent.Status.StatusDetail).To(Equal("All dependencies validated successfully"))
			Expect(updatedAgent.Status.ValidMCPServers).To(ContainElement(acp.ResolvedMCPServer{
				Name:  "test-fetch",
				Tools: []string{"fetch"},
			}))

			By("checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})

	Context("Pending:Pending -> Ready:Ready", func() {
		It("moves to Ready:Ready when sub-agent becomes ready", func() {
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
				Name:         agentName,
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
			reconciler := NewTestAgentReconciler(k8sClient, eventRecorder)

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
