package task

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
)

var _ = Describe("Collect Tools", func() {
	Context("When collecting tools from an agent with sub-agents", func() {
		const parentAgentName = "parent-agent"
		const subAgentName = "sub-agent"
		const subAgentDescription = "A specialized sub-agent for testing"
		const secretName = "test-secret"
		const llmName = "test-llm"

		ctx := context.Background()

		// Test objects
		var secret *corev1.Secret
		var llm *acp.LLM
		var subAgent *acp.Agent
		var parentAgent *acp.Agent

		BeforeEach(func() {
			// Create a test secret for the LLM
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"api-key": []byte("test-api-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			// Create a test LLM
			llm = &acp.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      llmName,
					Namespace: "default",
				},
				Spec: acp.LLMSpec{
					Provider: "openai",
					APIKeyFrom: &acp.APIKeySource{
						SecretKeyRef: acp.SecretKeyRef{
							Name: secretName,
							Key:  "api-key",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, llm)).To(Succeed())
			llm.Status.Status = "Ready"
			llm.Status.Ready = true
			Expect(k8sClient.Status().Update(ctx, llm)).To(Succeed())

			// Create the sub-agent with a description
			subAgent = &acp.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      subAgentName,
					Namespace: "default",
				},
				Spec: acp.AgentSpec{
					LLMRef: acp.LocalObjectReference{
						Name: llmName,
					},
					System:      "Sub agent system prompt",
					Description: subAgentDescription,
				},
			}
			Expect(k8sClient.Create(ctx, subAgent)).To(Succeed())
			subAgent.Status.Ready = true
			subAgent.Status.Status = "Ready"
			subAgent.Status.StatusDetail = "Ready for testing"
			Expect(k8sClient.Status().Update(ctx, subAgent)).To(Succeed())

			// Create the parent agent that references the sub-agent
			parentAgent = &acp.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      parentAgentName,
					Namespace: "default",
				},
				Spec: acp.AgentSpec{
					LLMRef: acp.LocalObjectReference{
						Name: llmName,
					},
					System: "Parent agent system prompt",
					SubAgents: []acp.LocalObjectReference{
						{Name: subAgentName},
					},
				},
			}
			Expect(k8sClient.Create(ctx, parentAgent)).To(Succeed())
			parentAgent.Status.Ready = true
			parentAgent.Status.Status = "Ready"
			parentAgent.Status.StatusDetail = "Ready for testing"
			Expect(k8sClient.Status().Update(ctx, parentAgent)).To(Succeed())
		})

		AfterEach(func() {
			// Clean up resources in reverse order
			if parentAgent != nil {
				Expect(k8sClient.Delete(ctx, parentAgent)).To(Succeed())
			}
			if subAgent != nil {
				Expect(k8sClient.Delete(ctx, subAgent)).To(Succeed())
			}
			if llm != nil {
				Expect(k8sClient.Delete(ctx, llm)).To(Succeed())
			}
			if secret != nil {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}
		})

		It("should include sub-agents as delegate tools", func() {
			// Create a task reconciler
			reconciler := &TaskReconciler{
				Client:     k8sClient,
				Scheme:     k8sClient.Scheme(),
				recorder:   record.NewFakeRecorder(10),
				MCPManager: mcpmanager.NewMCPServerManager(),
			}

			// Collect tools from the parent agent
			tools := reconciler.collectTools(ctx, parentAgent)

			fmt.Printf("Found tools: %+v\n", tools)

			// We expect at least one tool for the sub-agent
			Expect(tools).NotTo(BeEmpty())

			// Find the delegate tool for the sub-agent
			var delegateTool *llmclient.Tool
			for _, tool := range tools {
				if tool.ACPToolType == acp.ToolTypeDelegateToAgent {
					delegateTool = &tool
					break
				}
			}

			// The delegate tool should exist
			Expect(delegateTool).NotTo(BeNil())

			// The delegate tool should have the correct name with the sub-agent's name
			Expect(delegateTool.Function.Name).To(Equal("delegate_to_agent__" + subAgentName))

			// The delegate tool should have the sub-agent's description in its description
			Expect(delegateTool.Function.Description).To(ContainSubstring(subAgentDescription))

			// The delegate tool should have the correct ACPToolType
			Expect(delegateTool.ACPToolType).To(Equal(acp.ToolTypeDelegateToAgent))
		})
	})
})
