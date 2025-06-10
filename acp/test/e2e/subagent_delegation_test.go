/*
Copyright 2025.

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

package e2e

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

var _ = Describe("Sub-Agent Delegation", Ordered, func() {
	const testNamespace = "acp-testing"

	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", testNamespace)
		_, err := utils.Run(cmd)
		if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
			Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")
		}

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", testNamespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy-local-kind")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("creating OpenAI API key secret")
		secretYaml := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: openai
  namespace: %s
type: Opaque
data:
  OPENAI_API_KEY: %s
`, testNamespace, base64.StdEncoding.EncodeToString([]byte(os.Getenv("OPENAI_API_KEY"))))

		cmd = exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(secretYaml)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create OpenAI API key secret")

		By("creating an LLM")
		llmYaml := fmt.Sprintf(`
apiVersion: acp.humanlayer.dev/v1alpha1
kind: LLM
metadata:
  name: gpt-4o
  namespace: %s
spec:
  provider: openai
  parameters:
    model: gpt-4o
  apiKeyFrom:
    secretKeyRef:
      name: openai
      key: OPENAI_API_KEY
`, testNamespace)

		cmd = exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(llmYaml)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create LLM")

		By("waiting for LLM to be ready")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "llm", "gpt-4o", "-n", testNamespace, "-o", "jsonpath={.status.status}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("Ready"))
		}).Should(Succeed())
	})

	AfterAll(func() {
		By("removing manager namespace")
		cmd := exec.Command("kubectl", "delete", "ns", testNamespace)
		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		By("cleaning up test resources")
		cmd := exec.Command("kubectl", "delete", "task", "--all", "-n", testNamespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
		cmd = exec.Command("kubectl", "delete", "agent", "--all", "-n", testNamespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
		cmd = exec.Command("kubectl", "delete", "mcpserver", "--all", "-n", testNamespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
	})

	SetDefaultEventuallyTimeout(1 * time.Minute)
	SetDefaultEventuallyPollingInterval(5 * time.Second)

	Context("When creating a sub-agent delegation scenario", func() {
		FIt("should fail due to missing tool responses in context window (reproduces bug)", func() {
			By("creating a fetch MCP server")
			fetchServerYaml := fmt.Sprintf(`
apiVersion: acp.humanlayer.dev/v1alpha1
kind: MCPServer
metadata:
  name: fetch
  namespace: %s
spec:
  transport: "stdio"
  command: "uvx"
  args: ["mcp-server-fetch"]
`, testNamespace)
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(fetchServerYaml)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create fetch MCP server")

			By("waiting for fetch MCP server to be ready")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "mcpserver", "fetch", "-n", testNamespace, "-o", "jsonpath={.status.status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Ready"))
			}).Should(Succeed())

			By("creating a web-search agent with fetch MCP server")
			webSearchAgentYaml := fmt.Sprintf(`
apiVersion: acp.humanlayer.dev/v1alpha1
kind: Agent
metadata:
  name: web-search
  namespace: %s
spec:
  llmRef:
    name: gpt-4o
  system: |
    You are a helpful assistant. Your job is to help the user with their tasks.
  mcpServers:
    - name: fetch
`, testNamespace)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(webSearchAgentYaml)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create web-search agent")

			By("creating a manager agent with sub-agent delegation")
			managerAgentYaml := fmt.Sprintf(`
apiVersion: acp.humanlayer.dev/v1alpha1
kind: Agent
metadata:
  name: manager
  namespace: %s
spec:
  llmRef:
    name: gpt-4o
  system: |
    You are a helpful assistant. Your job is to help the user with their tasks.
  subAgents:
    - name: web-search
`, testNamespace)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(managerAgentYaml)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create manager agent")

			By("waiting for agents to be ready")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "agent", "web-search", "-n", testNamespace, "-o", "jsonpath={.status.status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Ready"))
			}).Should(Succeed())

			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "agent", "manager", "-n", testNamespace, "-o", "jsonpath={.status.status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Ready"))
			}).Should(Succeed())

			By("creating a task that triggers sub-agent delegation")
			managerTaskYaml := fmt.Sprintf(`
apiVersion: acp.humanlayer.dev/v1alpha1
kind: Task
metadata:
  name: manager-task
  namespace: %s
spec:
  agentRef:
    name: manager
  userMessage: "what is the data at https://lotrapi.co/api/v1/characters/2?"
`, testNamespace)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(managerTaskYaml)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create manager task")

			By("monitoring task execution and checking for proper tool response handling")
			var delegateTaskName string
			Eventually(func(g Gomega) {
				// First, find any delegated tasks
				cmd := exec.Command("kubectl", "get", "task", "-n", testNamespace, "-l", "acp.humanlayer.dev/parent-toolcall", "-o", "name")
				output, err := utils.Run(cmd)
				if err == nil && output != "" {
					lines := strings.Split(strings.TrimSpace(output), "\n")
					if len(lines) > 0 {
						delegateTaskName = strings.TrimPrefix(lines[0], "task.acp.humanlayer.dev/")
					}
				}

				// Check main task status
				cmd = exec.Command("kubectl", "get", "task", "manager-task", "-n", testNamespace, "-o", "jsonpath={.status.phase}")
				output, err = utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())

				// Should not be in error state
				g.Expect(output).NotTo(Equal("Failed"), "Task should not fail")
				g.Expect(output).NotTo(Equal("ErrorBackoff"), "Task should not be in error backoff")
			}, 3*time.Minute).Should(Succeed())

			By("checking if delegate task has proper context window with tool responses")
			Eventually(func(g Gomega) {
				// Find delegate tasks
				cmd := exec.Command("kubectl", "get", "task", "-n", testNamespace, "-l", "acp.humanlayer.dev/parent-toolcall", "-o", "name")
				output, err := utils.Run(cmd)
				if err == nil && output != "" {
					lines := strings.Split(strings.TrimSpace(output), "\n")
					if len(lines) > 0 {
						delegateTaskName = strings.TrimPrefix(lines[0], "task.acp.humanlayer.dev/")
					}
				}

				if delegateTaskName != "" {
					// Get the delegate task details
					cmd = exec.Command("kubectl", "get", "task", delegateTaskName, "-n", testNamespace, "-o", "yaml")
					output, err = utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())

					// Check if the task has an error related to tool_call_id
					if strings.Contains(output, "tool_call_id") && strings.Contains(output, "400") {
						// Print the context window for debugging
						_, _ = fmt.Fprintf(GinkgoWriter, "REPRODUCING BUG: Delegate task with 400 error:\n%s\n", output)

						// Check for tool calls in the task
						cmd = exec.Command("kubectl", "get", "toolcall", "-n", testNamespace, "-l", fmt.Sprintf("acp.humanlayer.dev/task=%s", delegateTaskName), "-o", "yaml")
						toolCallOutput, err := utils.Run(cmd)
						if err == nil {
							_, _ = fmt.Fprintf(GinkgoWriter, "Related ToolCalls:\n%s\n", toolCallOutput)
						}

						// This should fail the test - we found the 400 error
						Fail("REPRODUCED BUG: Tool call response not added to context window, causing 400 error: " +
							"An assistant message with 'tool_calls' must be followed by tool messages responding to each 'tool_call_id'")
					}
				}
			}, 3*time.Minute).Should(Succeed())

			By("waiting for the bug to manifest - this test should fail when the 400 error occurs")
			// This test is designed to fail when the bug occurs, demonstrating the issue exists
			// The bug should be: tool call succeeds but tool response is not added to context window
		})
	})
})
