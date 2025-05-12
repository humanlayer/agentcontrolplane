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
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

// workflowNamespace is the namespace where samples are deployed
const workflowNamespace = "default"

// E2E test for the complete workflow as described in the README.md
var _ = Describe("Complete Workflow", Ordered, func() {
	// Used to track resources created for cleanup
	var resourcesCreated bool

	// Timeout and polling interval for Eventually blocks
	const timeout = 5 * time.Minute
	const pollingInterval = 2 * time.Second

	// Define expected resources
	var expectedLLMs = []string{"claude-3-5-sonnet", "gpt-4o", "mistral-large", "gemini-pro", "vertex-gemini"}
	var expectedMCPServers = []string{"fetch-server"}
	var expectedAgents = []string{"claude-fetch-agent"}
	var expectedTasks = []string{"claude-fetch-example"}

	// Mock secrets for testing
	BeforeAll(func() {
		By("Creating test namespace if it doesn't exist")
		cmd := exec.Command("kubectl", "get", "namespace", workflowNamespace)
		_, err := utils.Run(cmd)
		if err != nil {
			cmd = exec.Command("kubectl", "create", "namespace", workflowNamespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")
		}

		By("Creating mock secrets required for LLMs")
		createMockSecrets()
	})

	// Clean up any resources created during the test
	AfterAll(func() {
		if resourcesCreated {
			By("Cleaning up sample resources")
			cmd := exec.Command("make", "undeploy-samples")
			_, _ = utils.Run(cmd)
		}

		By("Cleaning up mock secrets")
		cleanupMockSecrets()
	})

	// Test the complete workflow
	Context("Setting up a complete environment", func() {
		It("should deploy and verify all components", func() {
			By("Deploying sample resources")
			cmd := exec.Command("make", "deploy-samples")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to deploy sample resources")
			resourcesCreated = true

			By("Verifying LLMs are created correctly")
			verifyResources("llms.acp.humanlayer.dev", expectedLLMs, timeout, pollingInterval)

			By("Verifying MCP Servers are created correctly")
			verifyResources("mcpservers.acp.humanlayer.dev", expectedMCPServers, timeout, pollingInterval)

			By("Verifying Agents are created correctly")
			verifyResources("agents.acp.humanlayer.dev", expectedAgents, timeout, pollingInterval)

			By("Verifying Tasks are created correctly")
			verifyResources("tasks.acp.humanlayer.dev", expectedTasks, timeout, pollingInterval)

			By("Verifying Task status transitions to successful state")
			verifyTaskStatus("claude-fetch-example", timeout)
		})
	})

	// Test the observability stack
	Context("Setting up the observability stack", func() {
		It("should deploy and verify observability components", func() {
			By("Deploying the observability stack")
			cmd := exec.Command("make", "deploy-otel")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to deploy observability stack")

			By("Verifying Prometheus is running")
			verifyDeployment("prometheus-kube-prometheus-prometheus", "monitoring", timeout, pollingInterval)

			By("Verifying Grafana is running")
			verifyDeployment("prometheus-grafana", "monitoring", timeout, pollingInterval)

			By("Verifying OpenTelemetry Collector is running")
			verifyDeployment("opentelemetry-collector", "monitoring", timeout, pollingInterval)
		})
	})
})

// createMockSecrets creates mock secrets for testing LLMs
func createMockSecrets() {
	// Define mock secrets
	secrets := map[string]map[string]string{
		"anthropic": {"ANTHROPIC_API_KEY": "mock-anthropic-key"},
		"openai":    {"OPENAI_API_KEY": "mock-openai-key"},
		"mistral":   {"MISTRAL_API_KEY": "mock-mistral-key"},
		"google":    {"GOOGLE_API_KEY": "mock-google-key"},
		"vertex":    {"service-account-json": "{\"type\":\"service_account\",\"project_id\":\"mock-project\"}"},
	}

	// Create each secret
	for name, data := range secrets {
		args := []string{"create", "secret", "generic", name, "-n", workflowNamespace}
		for key, value := range data {
			args = append(args, fmt.Sprintf("--from-literal=%s=%s", key, value))
		}

		cmd := exec.Command("kubectl", args...)
		_, err := utils.Run(cmd)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create mock secret %s", name))
		}
	}
}

// cleanupMockSecrets removes the mock secrets
func cleanupMockSecrets() {
	secrets := []string{"anthropic", "openai", "mistral", "google", "vertex"}
	for _, name := range secrets {
		cmd := exec.Command("kubectl", "delete", "secret", name, "-n", workflowNamespace, "--ignore-not-found")
		_, _ = utils.Run(cmd)
	}
}

// verifyResources checks that the expected resources are created
func verifyResources(resourceType string, expectedResources []string, timeout, interval time.Duration) {
	verifyResourcesFunc := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", resourceType, "-n", workflowNamespace, "-o", "json")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to get %s", resourceType))

		var resourceList map[string]interface{}
		err = json.Unmarshal([]byte(output), &resourceList)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to parse resource list")

		items, ok := resourceList["items"].([]interface{})
		g.Expect(ok).To(BeTrue(), "Failed to parse items from resource list")

		// Create a map of found resources
		foundResources := make(map[string]bool)
		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			g.Expect(ok).To(BeTrue(), "Failed to parse item")

			metadata, ok := itemMap["metadata"].(map[string]interface{})
			g.Expect(ok).To(BeTrue(), "Failed to parse metadata")

			name, ok := metadata["name"].(string)
			g.Expect(ok).To(BeTrue(), "Failed to parse name")

			foundResources[name] = true
		}

		// Check that all expected resources are found
		for _, expected := range expectedResources {
			g.Expect(foundResources).To(HaveKey(expected), fmt.Sprintf("Expected %s not found", expected))
		}
	}

	Eventually(verifyResourcesFunc, timeout, interval).Should(Succeed())
}

// verifyTaskStatus checks that the task transitions to a successful state
func verifyTaskStatus(taskName string, timeout time.Duration) {
	verifyTaskStatusFunc := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "tasks.acp.humanlayer.dev", taskName, 
			"-n", workflowNamespace, "-o", "jsonpath={.status.status}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to get task status")
		g.Expect(output).To(Equal("Ready"), "Task is not in Ready status")

		cmd = exec.Command("kubectl", "get", "tasks.acp.humanlayer.dev", taskName,
			"-n", workflowNamespace, "-o", "jsonpath={.status.phase}")
		output, err = utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to get task phase")
		g.Expect(output).To(Equal("Succeeded"), "Task is not in Succeeded phase")
	}

	Eventually(verifyTaskStatusFunc, timeout, 5*time.Second).Should(Succeed())
}

// verifyDeployment checks that a deployment is running
func verifyDeployment(deploymentName, namespace string, timeout, interval time.Duration) {
	verifyDeploymentFunc := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "deployment", deploymentName, 
			"-n", namespace, "-o", "jsonpath={.status.readyReplicas}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to get deployment %s", deploymentName))
		
		readyReplicas := strings.TrimSpace(output)
		g.Expect(readyReplicas).NotTo(Equal(""), "No ready replicas found")
		g.Expect(readyReplicas).NotTo(Equal("0"), "No ready replicas")
	}

	Eventually(verifyDeploymentFunc, timeout, interval).Should(Succeed())
}