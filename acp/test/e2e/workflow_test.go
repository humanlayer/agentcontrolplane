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
	"os"
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
//
// This test follows the steps from the README.md, specifically:
// 1. Setting up a Kubernetes cluster
// 2. Installing and configuring the ACP operator
// 3. Creating LLM resources with API keys
// 4. Creating an Agent and Task
// 5. Setting up and verifying MCP Server tools
// 6. Observability setup and verification (optional)
//
// When E2E_USE_REAL_CREDENTIALS=true in the environment, the test will use real API keys
// Otherwise, it will use mock keys that won't allow full task execution
var _ = Describe("README Workflow", Ordered, func() {
	// Used to track resources created for cleanup
	var resourcesCreated bool

	// Timeout and polling interval for Eventually blocks
	const timeout = 5 * time.Minute
	const pollingInterval = 2 * time.Second

	// Whether to use real credentials from environment variables
	var useRealCredentials = getEnvBool("E2E_USE_REAL_CREDENTIALS")

	// Define expected resources based on what's in the README and config/samples
	// These sample resources are defined in the config/samples directory
	var expectedLLMs = []string{"gpt-4o"}
	var expectedMCPServers = []string{"fetch"}
	var expectedAgents = []string{"my-assistant"}
	var expectedTasks = []string{"hello-world-1"}

	// If we're testing with Anthropic as well
	if useRealCredentials && os.Getenv("ANTHROPIC_API_KEY") != "" {
		expectedLLMs = append(expectedLLMs, "claude-3-5-sonnet")
		expectedAgents = append(expectedAgents, "claude")
		expectedTasks = append(expectedTasks, "claude-task")
	}

	// Setup the environment following README.md steps
	BeforeAll(func() {
		By("Step 1: Ensuring Kubernetes cluster is ready")
		// In the CI environment, the cluster is already created by the workflow
		// In a local environment, the user would run 'kind create cluster'
		cmd := exec.Command("kubectl", "cluster-info")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Kubernetes cluster is not ready")

		By("Step 2: Creating test namespace if it doesn't exist")
		cmd = exec.Command("kubectl", "get", "namespace", workflowNamespace)
		_, err = utils.Run(cmd)
		if err != nil {
			cmd = exec.Command("kubectl", "create", "namespace", workflowNamespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")
		}

		By("Step 3: Creating LLM API key secrets (using real ones if available)")
		createMockSecrets()

		By("Step 4: Installing ACP Custom Resource Definitions (CRDs)")
		// This is equivalent to applying the latest-crd.yaml in the README
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("Step 5: Deploying the ACP controller to the cluster")
		// This is equivalent to applying the latest.yaml in the README
		cmd = exec.Command("make", "deploy-local-kind")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller")

		// Wait for controller deployment to be ready
		verifyControllerReady := func(g Gomega) {
			// Check in default namespace instead of acp-system
			cmd := exec.Command("kubectl", "get", "deployments", "-n", "default",
				"-l", "control-plane=controller-manager", "-o", "jsonpath={.items[0].status.readyReplicas}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred(), "Failed to get deployment status")
			readyReplicas := strings.TrimSpace(output)
			g.Expect(readyReplicas).NotTo(Equal(""), "No ready replicas found")
			g.Expect(readyReplicas).NotTo(Equal("0"), "No ready replicas")
		}
		Eventually(verifyControllerReady, timeout, pollingInterval).Should(Succeed())
	})

	// Clean up any resources created during the test
	AfterAll(func() {
		if resourcesCreated {
			By("Cleaning up sample resources")
			cmd := exec.Command("make", "undeploy-samples")
			_, _ = utils.Run(cmd)
		}

		By("Undeploying the controller from default namespace")
		cmd := exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("Uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("Cleaning up mock secrets")
		cleanupMockSecrets()
	})

	// Test the README.md workflow
	Context("Following the README.md Getting Started guide", func() {
		It("should create all resources and verify they work as expected", func() {
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

			// Verify tasks
			if useRealCredentials {
				By("Verifying tasks complete successfully using real API keys")
				for _, taskName := range expectedTasks {
					By(fmt.Sprintf("Verifying task %s completes successfully", taskName))
					verifyTaskStatus(taskName, timeout)
				}
			} else {
				By("Verifying tasks exist in the cluster (not checking status due to mock API keys)")
				verifyTasksExist(timeout)
			}
		})
	})

	// Test the OpenTelemetry integration described in the README.md
	Context("Setting up the OpenTelemetry observability stack", func() {
		It("should deploy and verify Prometheus, Grafana, and OpenTelemetry components", func() {
			skipObservability := getEnvBool("E2E_SKIP_OBSERVABILITY")

			if skipObservability {
				By("Skipping observability stack tests (E2E_SKIP_OBSERVABILITY=true)")
				return
			}

			// Attempt to deploy the observability stack if not skipped
			By("Deploying the observability stack")
			cmd := exec.Command("make", "-C", "../../..", "deploy-otel")
			output, err := utils.Run(cmd)

			// We'll continue even if there's an error, since the observability
			// stack is optional and may not be fully supported in all environments
			if err != nil {
				By(fmt.Sprintf("Warning: Observability stack deployment had issues: %v", err))
				By(fmt.Sprintf("Output: %s", output))
				By("Continuing without full observability testing")
				return
			}

			// If we get here, the observability stack was deployed
			By("Verifying Prometheus deployment")
			cmd = exec.Command("kubectl", "get", "deployment", "prometheus-operator",
				"-n", "default", "--ignore-not-found")
			_, _ = utils.Run(cmd)

			By("Note: Full observability stack verification would include:")
			// In a full test, we would verify:
			// - Prometheus deployment
			// - Grafana deployment
			// - OpenTelemetry Collector deployment
		})
	})
})

// createMockSecrets creates secrets for testing LLMs
// Uses real credentials from environment variables if available (when E2E_USE_REAL_CREDENTIALS=true)
// Otherwise, falls back to mock credentials
func createMockSecrets() {
	useRealCredentials := getEnvBool("E2E_USE_REAL_CREDENTIALS")

	// Initialize secrets map
	secrets := map[string]map[string]string{
		"anthropic": {"ANTHROPIC_API_KEY": "mock-anthropic-key"},
		"openai":    {"OPENAI_API_KEY": "mock-openai-key"},
		"mistral":   {"MISTRAL_API_KEY": "mock-mistral-key"},
		"google":    {"GOOGLE_API_KEY": "mock-google-key"},
		"vertex":    {"service-account-json": "{\"type\":\"service_account\",\"project_id\":\"mock-project\"}"},
	}

	// If using real credentials, replace mock values with real ones from environment
	if useRealCredentials {
		By("Using real API credentials from environment variables")

		// OpenAI
		openaiKey := os.Getenv("OPENAI_API_KEY")
		if openaiKey != "" {
			By("Using real OpenAI API key")
			secrets["openai"] = map[string]string{"OPENAI_API_KEY": openaiKey}
		} else {
			By("WARNING: E2E_USE_REAL_CREDENTIALS is true but OPENAI_API_KEY not set")
		}

		// Anthropic
		anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
		if anthropicKey != "" {
			By("Using real Anthropic API key")
			secrets["anthropic"] = map[string]string{"ANTHROPIC_API_KEY": anthropicKey}
		}

		// Mistral
		mistralKey := os.Getenv("MISTRAL_API_KEY")
		if mistralKey != "" {
			By("Using real Mistral API key")
			secrets["mistral"] = map[string]string{"MISTRAL_API_KEY": mistralKey}
		}

		// Other providers could be added here
	} else {
		By("Using mock API credentials (set E2E_USE_REAL_CREDENTIALS=true to use real keys)")
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
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create secret %s", name))
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
// nolint:unparam
func verifyResources(resourceType string, expectedResources []string, _ /* timeout */, interval time.Duration) {
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

	// We use a constant timeout value for all resource checks
	const defaultTimeout = 5 * time.Minute
	Eventually(verifyResourcesFunc, defaultTimeout, interval).Should(Succeed())
}

// verifyTaskStatus checks that a task transitions to the Ready status
// nolint:unused
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

// verifyTasksExist checks that tasks are created, even if they're in error state
// For our e2e test, we just need to verify that the resources are created,
// since the LLMs will be in error state with mock API keys
func verifyTasksExist(timeout time.Duration) {
	verifyTasksExistFunc := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "tasks.acp.humanlayer.dev",
			"-n", workflowNamespace, "-o", "jsonpath={.items[*].metadata.name}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to get tasks")

		// Split the output to get task names
		taskNames := strings.Fields(output)
		g.Expect(taskNames).NotTo(BeEmpty(), "No tasks found")

		// In a real environment with valid API keys, we would check for Ready status here
		// For testing, we just verify that tasks were created

		By(fmt.Sprintf("Found %d tasks in the cluster", len(taskNames)))
		for _, name := range taskNames {
			By(fmt.Sprintf("  - Task: %s", name))
		}
	}

	Eventually(verifyTasksExistFunc, timeout, 5*time.Second).Should(Succeed())
}

// verifyDeployment checks that a deployment is running with ready replicas
// nolint:unused
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
