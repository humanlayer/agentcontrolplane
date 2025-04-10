package agent

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

var _ = Describe("Agent Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-agent"
		const llmName = "test-llm"
		const toolName = "test-tool"
		const humanContactChannelName = "test-humancontactchannel"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			// Create a test LLM
			llm := &acp.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      llmName,
					Namespace: "default",
				},
				Spec: acp.LLMSpec{
					Provider: "openai",
					APIKeyFrom: &acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: "test-secret",
							Key:  "api-key",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, llm)).To(Succeed())

			// Mark LLM as ready
			llm.Status.Status = "Ready"
			llm.Status.StatusDetail = "Ready for testing"
			Expect(k8sClient.Status().Update(ctx, llm)).To(Succeed())

			// Create a test Tool
			tool := &acp.Tool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      toolName,
					Namespace: "default",
				},
				Spec: acp.ToolSpec{
					ToolType: "function",
					Name:     "test",
				},
			}
			Expect(k8sClient.Create(ctx, tool)).To(Succeed())

			// Mark Tool as ready
			tool.Status.Ready = true
			Expect(k8sClient.Status().Update(ctx, tool)).To(Succeed())

			// Create a test ContactChannel
			contactChannel := &acp.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      humanContactChannelName,
					Namespace: "default",
				},
				Spec: acp.ContactChannelSpec{
					Type: acp.ContactChannelTypeEmail,
					APIKeyFrom: acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: "test-secret",
							Key:  "api-key",
						},
					},
					Email: &acp.EmailChannelConfig{
						Address:          "test@example.com",
						ContextAboutUser: "Test user",
					},
				},
			}
			Expect(k8sClient.Create(ctx, contactChannel)).To(Succeed())

			// Mark ContactChannel as ready
			contactChannel.Status.Ready = true
			contactChannel.Status.Status = "Ready"
			contactChannel.Status.StatusDetail = "Ready for testing"
			Expect(k8sClient.Status().Update(ctx, contactChannel)).To(Succeed())
		})

		AfterEach(func() {
			// Cleanup test resources
			By("Cleanup the test LLM")
			llm := &acp.LLM{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: llmName, Namespace: "default"}, llm)
			if err == nil {
				Expect(k8sClient.Delete(ctx, llm)).To(Succeed())
			}

			By("Cleanup the test Tool")
			tool := &acp.Tool{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: toolName, Namespace: "default"}, tool)
			if err == nil {
				Expect(k8sClient.Delete(ctx, tool)).To(Succeed())
			}

			By("Cleanup the test ContactChannel")
			contactChannel := &acp.ContactChannel{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: humanContactChannelName, Namespace: "default"}, contactChannel)
			if err == nil {
				Expect(k8sClient.Delete(ctx, contactChannel)).To(Succeed())
			}

			By("Cleanup the test Agent")
			agent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, agent)
			if err == nil {
				Expect(k8sClient.Delete(ctx, agent)).To(Succeed())
			}
		})

		It("should successfully validate an agent with valid dependencies", func() {
			By("creating the test agent")
			testAgent := &utils.TestScopedAgent{
				Name:                 resourceName,
				SystemPrompt:         "Test agent",
				Tools:                []string{toolName},
				LLM:                  llmName,
				HumanContactChannels: []string{humanContactChannelName},
			}
			testAgent.Setup(k8sClient)
			defer testAgent.Teardown()

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
			Expect(updatedAgent.Status.ValidTools).To(ContainElement(acp.ResolvedTool{
				Kind: "Tool",
				Name: toolName,
			}))
			Expect(updatedAgent.Status.ValidHumanContactChannels).To(ContainElement(acp.ResolvedContactChannel{
				Name: humanContactChannelName,
				Type: "email",
			}))

			By("checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})

		It("should fail validation with non-existent LLM", func() {
			By("creating the test agent with invalid LLM")
			testAgent := &utils.TestScopedAgent{
				Name:                 resourceName,
				SystemPrompt:         "Test agent",
				Tools:                []string{toolName},
				LLM:                  "nonexistent-llm",
				HumanContactChannels: []string{humanContactChannelName},
			}
			testAgent.Setup(k8sClient)
			defer testAgent.Teardown()

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

		It("should fail validation with non-existent Tool", func() {
			By("creating the test agent with invalid Tool")
			testAgent := &utils.TestScopedAgent{
				Name:                 resourceName,
				SystemPrompt:         "Test agent",
				Tools:                []string{"nonexistent-tool"},
				LLM:                  llmName,
				HumanContactChannels: []string{humanContactChannelName},
			}
			testAgent.Setup(k8sClient)
			defer testAgent.Teardown()

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
			Expect(err.Error()).To(ContainSubstring(`"nonexistent-tool" not found`))

			By("checking the agent status")
			updatedAgent := &acp.Agent{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedAgent)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedAgent.Status.Ready).To(BeFalse())
			Expect(updatedAgent.Status.Status).To(Equal("Error"))
			Expect(updatedAgent.Status.StatusDetail).To(ContainSubstring(`"nonexistent-tool" not found`))

			By("checking that a failure event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationFailed")
		})

		It("should fail validation with non-existent HumanContactChannel", func() {
			By("creating the test agent with invalid HumanContactChannel")
			testAgent := &utils.TestScopedAgent{
				Name:                 resourceName,
				SystemPrompt:         "Test agent",
				Tools:                []string{toolName},
				LLM:                  llmName,
				HumanContactChannels: []string{"nonexistent-humancontactchannel"},
			}
			testAgent.Setup(k8sClient)
			defer testAgent.Teardown()

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
})
