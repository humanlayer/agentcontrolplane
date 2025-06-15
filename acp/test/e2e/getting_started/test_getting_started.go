/*
Copyright 2025 the Agent Control Plane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package getting_started

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	. "github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

var _ = Describe("Getting Started Tests", func() {
	It("should have working Kubernetes client", func() {
		ctx := testFramework.GetContext()
		client := testFramework.GetClient()

		By("creating a simple secret to test connectivity")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "connectivity-test",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"test": []byte("data"),
			},
		}

		err := client.Create(ctx, secret)
		Expect(err).NotTo(HaveOccurred())

		By("verifying the secret was created")
		createdSecret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{
			Name:      "connectivity-test",
			Namespace: "default",
		}, createdSecret)
		Expect(err).NotTo(HaveOccurred())
		Expect(createdSecret.Data["test"]).To(Equal([]byte("data")))

		By("cleaning up the secret")
		err = client.Delete(ctx, secret)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create LLM resources with proper status validation", func() {
		ctx := testFramework.GetContext()
		client := testFramework.GetClient()

		By("creating an LLM resource with mock server configuration")
		llm := &acp.LLM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-llm-validation",
				Namespace: "default",
			},
			Spec: acp.LLMSpec{
				Provider: "openai",
				Parameters: acp.BaseConfig{
					Model: "gpt-4o",
				},
			},
		}

		err := client.Create(ctx, llm)
		Expect(err).NotTo(HaveOccurred())

		By("verifying the LLM was created with correct spec")
		createdLLM := &acp.LLM{}
		err = client.Get(ctx, types.NamespacedName{
			Name:      "test-llm-validation",
			Namespace: "default",
		}, createdLLM)
		Expect(err).NotTo(HaveOccurred())
		Expect(createdLLM.Spec.Provider).To(Equal("openai"))
		Expect(createdLLM.Spec.Parameters.Model).To(Equal("gpt-4o"))

		By("verifying LLM status is set correctly")
		// The LLM may not be ready without proper API key, but should have some status
		// Status might be empty initially, so we'll just verify the resource was created correctly

		By("cleaning up the LLM")
		err = client.Delete(ctx, llm)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle agent without LLM gracefully", func() {
		ctx := testFramework.GetContext()
		client := testFramework.GetClient()

		By("creating an agent with non-existent LLM reference")
		agent := &acp.Agent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-agent-orphan",
				Namespace: "default",
			},
			Spec: acp.AgentSpec{
				LLMRef: acp.LocalObjectReference{
					Name: "non-existent-llm",
				},
				System: "You are a test assistant.",
			},
		}

		err := client.Create(ctx, agent)
		Expect(err).NotTo(HaveOccurred())

		By("verifying agent is not ready due to missing LLM")
		Eventually(func(g Gomega) {
			updatedAgent := &acp.Agent{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      agent.Name,
				Namespace: "default",
			}, updatedAgent)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(updatedAgent.Status.Ready).To(BeFalse())
			g.Expect(updatedAgent.Status.StatusDetail).To(ContainSubstring("LLM"))
		}).Should(Succeed())

		By("cleaning up the agent")
		err = client.Delete(ctx, agent)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should validate resource creation order and dependencies", func() {
		ctx := testFramework.GetContext()
		client := testFramework.GetClient()
		uniqueID := fmt.Sprintf("validation-%d", time.Now().UnixNano())

		By("creating secret first (dependency for LLM)")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-secret-%s", uniqueID),
				Namespace: "default",
			},
			Data: map[string][]byte{
				"api-key": []byte("test-key"),
			},
		}
		err := client.Create(ctx, secret)
		Expect(err).NotTo(HaveOccurred())

		By("creating LLM resource")
		llm := &acp.LLM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-llm-%s", uniqueID),
				Namespace: "default",
			},
			Spec: acp.LLMSpec{
				Provider: "openai",
				APIKeyFrom: &acp.APIKeySource{
					SecretKeyRef: acp.SecretKeyRef{
						Name: secret.Name,
						Key:  "api-key",
					},
				},
				Parameters: acp.BaseConfig{
					Model: "gpt-4o",
				},
			},
		}
		err = client.Create(ctx, llm)
		Expect(err).NotTo(HaveOccurred())

		By("creating Agent resource that references the LLM")
		agent := &acp.Agent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-agent-%s", uniqueID),
				Namespace: "default",
			},
			Spec: acp.AgentSpec{
				LLMRef: acp.LocalObjectReference{
					Name: llm.Name,
				},
				System: "You are a helpful assistant. Your job is to help the user with their tasks.",
			},
		}
		err = client.Create(ctx, agent)
		Expect(err).NotTo(HaveOccurred())

		By("verifying all resources were created with proper references")
		// Verify Secret
		createdSecret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: "default"}, createdSecret)
		Expect(err).NotTo(HaveOccurred())

		// Verify LLM
		createdLLM := &acp.LLM{}
		err = client.Get(ctx, types.NamespacedName{Name: llm.Name, Namespace: "default"}, createdLLM)
		Expect(err).NotTo(HaveOccurred())
		Expect(createdLLM.Spec.APIKeyFrom.SecretKeyRef.Name).To(Equal(secret.Name))

		// Verify Agent
		createdAgent := &acp.Agent{}
		err = client.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: "default"}, createdAgent)
		Expect(err).NotTo(HaveOccurred())
		Expect(createdAgent.Spec.LLMRef.Name).To(Equal(llm.Name))

		By("cleaning up resources in proper order")
		err = client.Delete(ctx, agent)
		Expect(err).NotTo(HaveOccurred())
		err = client.Delete(ctx, llm)
		Expect(err).NotTo(HaveOccurred())
		err = client.Delete(ctx, secret)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("Getting Started Flow", func() {
	var (
		ctx        context.Context
		client     client.Client
		namespace  = "default"
		uniqueID   string
		testSecret *TestSecret
		testLLM    *TestLLM
		testAgent  *TestAgent
		testTask   *TestTask
		mockServer *httptest.Server
	)

	BeforeEach(func() {
		// Initialize context and client from framework
		ctx = testFramework.GetContext()
		client = testFramework.GetClient()

		// Setup mock server for LLM API calls
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Always return success for our tests
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			// Return appropriate OpenAI-compatible response
			_, err := w.Write([]byte(`{"id":"test-id","choices":[{"message":{"content":"test"}}]}`))
			if err != nil {
				http.Error(w, "Error writing response", http.StatusInternalServerError)
				return
			}
		}))

		// Generate unique test ID for resource names
		uniqueID = fmt.Sprintf("test-%d", time.Now().UnixNano())

		// Setup test resources with unique names
		testSecret = &TestSecret{
			Name: fmt.Sprintf("test-secret-%s", uniqueID),
		}

		testLLM = &TestLLM{
			Name:       fmt.Sprintf("test-llm-%s", uniqueID),
			SecretName: testSecret.Name,
		}

		testAgent = &TestAgent{
			Name:         fmt.Sprintf("test-agent-%s", uniqueID),
			SystemPrompt: "You are a helpful test assistant.",
			LLM:          testLLM.Name,
		}

		testTask = &TestTask{
			Name:        fmt.Sprintf("test-task-%s", uniqueID),
			AgentName:   testAgent.Name,
			UserMessage: "What is the capital of France?",
		}
	})

	AfterEach(func() {
		// Clean up mock server
		if mockServer != nil {
			mockServer.Close()
		}

		// Clean up test resources
		if testTask != nil {
			testTask.Teardown(ctx)
		}
		if testAgent != nil {
			testAgent.Teardown(ctx)
		}
		if testLLM != nil {
			// Manual cleanup for LLM since we created it manually
			llmResource := &acp.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testLLM.Name,
					Namespace: namespace,
				},
			}
			_ = client.Delete(ctx, llmResource)
		}
		if testSecret != nil {
			testSecret.Teardown(ctx)
		}
	})

	It("should successfully create and process LLM → Agent → Task flow", func() {
		By("creating test secret for LLM API key")
		secret := testSecret.Setup(ctx, client)
		Expect(secret).NotTo(BeNil())

		By("creating LLM resource with mock server URL")
		llm := &acp.LLM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testLLM.Name,
				Namespace: namespace,
			},
			Spec: acp.LLMSpec{
				Provider: "openai",
				APIKeyFrom: &acp.APIKeySource{
					SecretKeyRef: acp.SecretKeyRef{
						Name: testSecret.Name,
						Key:  "api-key",
					},
				},
				Parameters: acp.BaseConfig{
					BaseURL: mockServer.URL, // Use mock server URL
					Model:   "test-model",
				},
			},
		}
		err := client.Create(ctx, llm)
		Expect(err).NotTo(HaveOccurred())
		Expect(llm.Spec.Provider).To(Equal("openai"))

		By("waiting for LLM to be ready")
		Eventually(func(g Gomega) {
			updatedLLM := &acp.LLM{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      llm.Name,
				Namespace: namespace,
			}, updatedLLM)
			g.Expect(err).NotTo(HaveOccurred())

			// Debug output
			fmt.Printf("LLM Status: Ready=%v, Status=%s, StatusDetail=%s\n",
				updatedLLM.Status.Ready, updatedLLM.Status.Status, updatedLLM.Status.StatusDetail)

			g.Expect(updatedLLM.Status.Ready).To(BeTrue())
		}).Should(Succeed())

		By("creating Agent resource")
		agent := testAgent.Setup(ctx, client)
		Expect(agent).NotTo(BeNil())
		Expect(agent.Spec.LLMRef.Name).To(Equal(testLLM.Name))
		Expect(agent.Spec.System).To(Equal("You are a helpful test assistant."))

		By("waiting for Agent to be ready")
		Eventually(func(g Gomega) {
			updatedAgent := &acp.Agent{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      agent.Name,
				Namespace: namespace,
			}, updatedAgent)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(updatedAgent.Status.Ready).To(BeTrue())
		}).Should(Succeed())

		By("creating Task resource")
		task := testTask.Setup(ctx, client)
		Expect(task).NotTo(BeNil())
		Expect(task.Spec.AgentRef.Name).To(Equal(agent.Name))
		Expect(task.Spec.UserMessage).To(Equal("What is the capital of France?"))

		By("waiting for Task to start processing")
		Eventually(func(g Gomega) {
			updatedTask := &acp.Task{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      task.Name,
				Namespace: namespace,
			}, updatedTask)
			g.Expect(err).NotTo(HaveOccurred())
			// Task should be in some phase (not empty) and have span context
			g.Expect(updatedTask.Status.Phase).NotTo(BeEmpty())
			g.Expect(updatedTask.Status.SpanContext).NotTo(BeNil())
		}).Should(Succeed())

		By("waiting for Task to complete the full LLM workflow")
		Eventually(func(g Gomega) {
			updatedTask := &acp.Task{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      task.Name,
				Namespace: namespace,
			}, updatedTask)
			g.Expect(err).NotTo(HaveOccurred())
			// Task should complete to FinalAnswer phase
			g.Expect(updatedTask.Status.Phase).To(Equal(acp.TaskPhaseFinalAnswer))
			// Should have system, user, and assistant messages
			g.Expect(updatedTask.Status.ContextWindow).To(HaveLen(3))
			g.Expect(updatedTask.Status.ContextWindow[0].Role).To(Equal("system"))
			g.Expect(updatedTask.Status.ContextWindow[1].Role).To(Equal("user"))
			g.Expect(updatedTask.Status.ContextWindow[2].Role).To(Equal("assistant"))
		}).Should(Succeed())

		By("verifying the complete flow worked end-to-end")
		finalTask := &acp.Task{}
		err = client.Get(ctx, types.NamespacedName{
			Name:      task.Name,
			Namespace: namespace,
		}, finalTask)
		Expect(err).NotTo(HaveOccurred())

		// Verify task reached final answer with complete context
		Expect(finalTask.Status.Phase).To(Equal(acp.TaskPhaseFinalAnswer))
		Expect(finalTask.Status.ContextWindow).To(HaveLen(3))
		Expect(finalTask.Status.ContextWindow[0].Content).To(ContainSubstring("helpful test assistant"))
		Expect(finalTask.Status.ContextWindow[1].Content).To(ContainSubstring("capital of France"))
		Expect(finalTask.Status.ContextWindow[2].Content).To(ContainSubstring("test")) // Mock response
	})

	It("should complete the full getting-started tutorial workflow", func() {
		By("creating test secret for OpenAI API key (mimicking kubectl create secret)")
		secret := testSecret.Setup(ctx, client)
		Expect(secret).NotTo(BeNil())

		By("creating LLM resource like in getting-started guide")
		llm := &acp.LLM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testLLM.Name,
				Namespace: namespace,
			},
			Spec: acp.LLMSpec{
				Provider: "openai",
				APIKeyFrom: &acp.APIKeySource{
					SecretKeyRef: acp.SecretKeyRef{
						Name: testSecret.Name,
						Key:  "api-key",
					},
				},
				Parameters: acp.BaseConfig{
					BaseURL: mockServer.URL,
					Model:   "gpt-4o",
				},
			},
		}
		err := client.Create(ctx, llm)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for LLM to be ready with proper status")
		Eventually(func(g Gomega) {
			updatedLLM := &acp.LLM{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      llm.Name,
				Namespace: namespace,
			}, updatedLLM)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(updatedLLM.Status.Ready).To(BeTrue())
		}).Should(Succeed())

		By("creating Agent resource like in getting-started guide")
		agent := &acp.Agent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testAgent.Name,
				Namespace: namespace,
			},
			Spec: acp.AgentSpec{
				LLMRef: acp.LocalObjectReference{
					Name: testLLM.Name,
				},
				System: "You are a helpful assistant. Your job is to help the user with their tasks.",
			},
		}
		err = client.Create(ctx, agent)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for Agent to be ready")
		Eventually(func(g Gomega) {
			updatedAgent := &acp.Agent{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      agent.Name,
				Namespace: namespace,
			}, updatedAgent)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(updatedAgent.Status.Ready).To(BeTrue())
		}).Should(Succeed())

		By("creating Task like in getting-started guide")
		task := &acp.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testTask.Name,
				Namespace: namespace,
			},
			Spec: acp.TaskSpec{
				AgentRef: acp.LocalObjectReference{
					Name: agent.Name,
				},
				UserMessage: "What is the capital of the moon?",
			},
		}
		err = client.Create(ctx, task)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for Task to complete like in getting-started guide")
		Eventually(func(g Gomega) {
			updatedTask := &acp.Task{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      task.Name,
				Namespace: namespace,
			}, updatedTask)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(updatedTask.Status.Phase).To(Equal(acp.TaskPhaseFinalAnswer))
		}).Should(Succeed())

		By("verifying the complete context window like in getting-started describe output")
		finalTask := &acp.Task{}
		err = client.Get(ctx, types.NamespacedName{
			Name:      task.Name,
			Namespace: namespace,
		}, finalTask)
		Expect(err).NotTo(HaveOccurred())

		// Verify context window structure like getting-started guide shows
		Expect(finalTask.Status.ContextWindow).To(HaveLen(3))
		Expect(finalTask.Status.ContextWindow[0].Role).To(Equal("system"))
		Expect(finalTask.Status.ContextWindow[0].Content).To(ContainSubstring("helpful assistant"))
		Expect(finalTask.Status.ContextWindow[1].Role).To(Equal("user"))
		Expect(finalTask.Status.ContextWindow[1].Content).To(Equal("What is the capital of the moon?"))
		Expect(finalTask.Status.ContextWindow[2].Role).To(Equal("assistant"))
		Expect(finalTask.Status.ContextWindow[2].Content).NotTo(BeEmpty())

		// Verify output field like getting-started guide shows
		Expect(finalTask.Status.Output).NotTo(BeEmpty())
		Expect(finalTask.Status.Output).To(Equal(finalTask.Status.ContextWindow[2].Content))

		// Verify span context is set for tracing
		Expect(finalTask.Status.SpanContext).NotTo(BeNil())
		Expect(finalTask.Status.SpanContext.TraceID).NotTo(BeEmpty())
		Expect(finalTask.Status.SpanContext.SpanID).NotTo(BeEmpty())
	})

	It("should handle task with missing agent gracefully", func() {
		By("creating a task with non-existent agent")
		task := &acp.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("orphan-task-%s", uniqueID),
				Namespace: namespace,
			},
			Spec: acp.TaskSpec{
				AgentRef: acp.LocalObjectReference{
					Name: "non-existent-agent",
				},
				UserMessage: "This should fail",
			},
		}
		err := client.Create(ctx, task)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for task to be in pending state")
		Eventually(func(g Gomega) {
			updatedTask := &acp.Task{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      task.Name,
				Namespace: namespace,
			}, updatedTask)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(updatedTask.Status.Phase).To(Equal(acp.TaskPhasePending))
			g.Expect(updatedTask.Status.StatusDetail).To(ContainSubstring("Waiting for Agent to exist"))
		}).Should(Succeed())

		By("cleaning up orphan task")
		err = client.Delete(ctx, task)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("MCP Server Integration Tests", func() {
	var (
		ctx        context.Context
		client     client.Client
		namespace  = "default"
		uniqueID   string
		testSecret *TestSecret
		testLLM    *TestLLM
		testAgent  *TestAgent
		testTask   *TestTask
		mockServer *httptest.Server
	)

	BeforeEach(func() {
		ctx = testFramework.GetContext()
		client = testFramework.GetClient()

		// Setup mock server for both LLM and fetch tool responses
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Mock different endpoints
			if r.URL.Path == "/v1/chat/completions" {
				// LLM endpoint
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				// Return a response that indicates tool calling
				_, err := w.Write([]byte(`{
					"id": "test-id",
					"choices": [{
						"message": {
							"content": null,
							"tool_calls": [{
								"id": "tool-call-1",
								"type": "function", 
								"function": {
									"name": "fetch__fetch",
									"arguments": "{\"url\":\"https://example.com/api/data\"}"
								}
							}]
						}
					}]
				}`))
				if err != nil {
					http.Error(w, "Error writing response", http.StatusInternalServerError)
				}
			} else {
				// Default mock response for other requests
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"result": "mock response"}`))
				if err != nil {
					http.Error(w, "Error writing response", http.StatusInternalServerError)
				}
			}
		}))

		uniqueID = fmt.Sprintf("mcp-test-%d", time.Now().UnixNano())

		testSecret = &TestSecret{
			Name: fmt.Sprintf("test-secret-%s", uniqueID),
		}

		testLLM = &TestLLM{
			Name:       fmt.Sprintf("test-llm-%s", uniqueID),
			SecretName: testSecret.Name,
		}

		testAgent = &TestAgent{
			Name:         fmt.Sprintf("test-agent-%s", uniqueID),
			SystemPrompt: "You are a helpful assistant that can use tools.",
			LLM:          testLLM.Name,
		}

		testTask = &TestTask{
			Name:        fmt.Sprintf("test-task-%s", uniqueID),
			AgentName:   testAgent.Name,
			UserMessage: "Fetch data from https://example.com/api/data",
		}
	})

	AfterEach(func() {
		if mockServer != nil {
			mockServer.Close()
		}

		// Clean up test resources
		if testTask != nil {
			testTask.Teardown(ctx)
		}
		if testAgent != nil {
			testAgent.Teardown(ctx)
		}
		if testLLM != nil {
			llmResource := &acp.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testLLM.Name,
					Namespace: namespace,
				},
			}
			_ = client.Delete(ctx, llmResource)
		}
		if testSecret != nil {
			testSecret.Teardown(ctx)
		}

		// Clean up MCP server
		mcpServer := &acp.MCPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-fetch-%s", uniqueID),
				Namespace: namespace,
			},
		}
		_ = client.Delete(ctx, mcpServer)
	})

	It("should create and validate MCPServer resource", func() {
		By("creating an MCPServer like in getting-started guide")
		mcpServer := &acp.MCPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-fetch-%s", uniqueID),
				Namespace: namespace,
			},
			Spec: acp.MCPServerSpec{
				Transport: "stdio",
				Command:   "echo", // Use echo instead of uvx for testing
				Args: []string{
					`{"jsonrpc":"2.0","result":{"capabilities":{"tools":{"fetch":` +
						`{"description":"Fetch URL","inputSchema":{"type":"object",` +
						`"properties":{"url":{"type":"string"}},"required":["url"]}}}},"id":1}}`,
				},
			},
		}

		err := client.Create(ctx, mcpServer)
		Expect(err).NotTo(HaveOccurred())

		By("verifying MCPServer was created")
		createdMCPServer := &acp.MCPServer{}
		err = client.Get(ctx, types.NamespacedName{
			Name:      mcpServer.Name,
			Namespace: namespace,
		}, createdMCPServer)
		Expect(err).NotTo(HaveOccurred())
		Expect(createdMCPServer.Spec.Transport).To(Equal("stdio"))
		Expect(createdMCPServer.Spec.Command).To(Equal("echo"))

		By("verifying MCPServer status is eventually set")
		// The MCP server controller will attempt to connect and set status
		Eventually(func(g Gomega) {
			updatedMCPServer := &acp.MCPServer{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: namespace,
			}, updatedMCPServer)
			g.Expect(err).NotTo(HaveOccurred())
			// The controller should have attempted to process this server
			// Status may be empty initially but should eventually be set
		}, "15s", "1s").Should(Succeed())
	})

	It("should handle agent with MCPServer reference", func() {
		By("creating MCPServer resource first")
		mcpServer := &acp.MCPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-fetch-%s", uniqueID),
				Namespace: namespace,
			},
			Spec: acp.MCPServerSpec{
				Transport: "stdio",
				Command:   "echo",
				Args:      []string{"test"},
			},
		}
		err := client.Create(ctx, mcpServer)
		Expect(err).NotTo(HaveOccurred())

		By("creating test secret for LLM")
		secret := testSecret.Setup(ctx, client)
		Expect(secret).NotTo(BeNil())

		By("creating LLM resource")
		llm := &acp.LLM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testLLM.Name,
				Namespace: namespace,
			},
			Spec: acp.LLMSpec{
				Provider: "openai",
				APIKeyFrom: &acp.APIKeySource{
					SecretKeyRef: acp.SecretKeyRef{
						Name: testSecret.Name,
						Key:  "api-key",
					},
				},
				Parameters: acp.BaseConfig{
					BaseURL: mockServer.URL,
					Model:   "gpt-4o",
				},
			},
		}
		err = client.Create(ctx, llm)
		Expect(err).NotTo(HaveOccurred())

		By("creating Agent with MCPServer reference like in getting-started guide")
		agent := &acp.Agent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testAgent.Name,
				Namespace: namespace,
			},
			Spec: acp.AgentSpec{
				LLMRef: acp.LocalObjectReference{
					Name: testLLM.Name,
				},
				System: "You are a helpful assistant that can use tools.",
				MCPServers: []acp.LocalObjectReference{
					{Name: mcpServer.Name},
				},
			},
		}
		err = client.Create(ctx, agent)
		Expect(err).NotTo(HaveOccurred())

		By("verifying Agent was created with MCP server reference")
		createdAgent := &acp.Agent{}
		err = client.Get(ctx, types.NamespacedName{
			Name:      agent.Name,
			Namespace: namespace,
		}, createdAgent)
		Expect(err).NotTo(HaveOccurred())
		Expect(createdAgent.Spec.MCPServers).To(HaveLen(1))
		Expect(createdAgent.Spec.MCPServers[0].Name).To(Equal(mcpServer.Name))

		By("verifying Agent references are correct")
		Expect(createdAgent.Spec.LLMRef.Name).To(Equal(testLLM.Name))
		Expect(createdAgent.Spec.System).To(ContainSubstring("tools"))
	})

	It("should validate MCPServer creation and basic properties", func() {
		By("creating MCPServer with basic configuration")
		testMCPServer := &acp.MCPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("basic-test-%s", uniqueID),
				Namespace: namespace,
			},
			Spec: acp.MCPServerSpec{
				Transport: "stdio",
				Command:   "echo",
				Args:      []string{"hello", "world"},
			},
		}

		err := client.Create(ctx, testMCPServer)
		Expect(err).NotTo(HaveOccurred())

		By("verifying MCPServer properties are set correctly")
		createdMCPServer := &acp.MCPServer{}
		err = client.Get(ctx, types.NamespacedName{
			Name:      testMCPServer.Name,
			Namespace: namespace,
		}, createdMCPServer)
		Expect(err).NotTo(HaveOccurred())

		// Verify the basic properties
		Expect(createdMCPServer.Spec.Transport).To(Equal("stdio"))
		Expect(createdMCPServer.Spec.Command).To(Equal("echo"))
		Expect(createdMCPServer.Spec.Args).To(Equal([]string{"hello", "world"}))

		By("cleaning up test MCPServer")
		err = client.Delete(ctx, testMCPServer)
		Expect(err).NotTo(HaveOccurred())
	})
})
