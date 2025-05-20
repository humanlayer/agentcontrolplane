package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAgentEndpointStatus(t *testing.T) {
	// Create a scheme with our API types registered
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add corev1 to scheme: %v", err)
	}
	if err := acp.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add acp to scheme: %v", err)
	}

	// Create a fake client with the agent pre-loaded
	agent := &acp.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "status-test-agent",
			Namespace: "status-test-namespace",
		},
		Spec: acp.AgentSpec{
			LLMRef: acp.LocalObjectReference{Name: "status-test-llm"},
			System: "Agent with status",
		},
		Status: acp.AgentStatus{
			Ready:        true,
			Status:       acp.AgentStatusReady,
			StatusDetail: "Everything is working",
		},
	}

	llm := &acp.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "status-test-llm",
			Namespace: "status-test-namespace",
		},
		Spec: acp.LLMSpec{
			Provider: "test-provider",
			Parameters: acp.BaseConfig{
				Model: "test-model",
			},
		},
	}

	mcpServer := &acp.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "status-test-agent-mcp1",
			Namespace: "status-test-namespace",
		},
		Spec: acp.MCPServerSpec{
			Transport: "stdio",
			Command:   "python",
			Args:      []string{"-m", "script.py"},
		},
		Status: acp.MCPServerStatus{
			Connected:    true,
			Status:       "Ready",
			StatusDetail: "Connected to MCP server",
		},
	}

	// Create namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "status-test-namespace"},
	}

	// Add these objects to our client
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(namespace, llm, agent, mcpServer).
		Build()

	agent.Spec.MCPServers = []acp.LocalObjectReference{
		{Name: "status-test-agent-mcp1"},
	}
	ctx := context.Background()
	if err := k8sClient.Update(ctx, agent); err != nil {
		t.Fatalf("Failed to update agent: %v", err)
	}

	// Create an API server with the client
	apiServer := NewAPIServer(k8sClient, ":8080")
	gin.SetMode(gin.TestMode)

	// Test GET /agents/:name
	t.Run("GET /agents/:name", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/agents/status-test-agent?namespace=status-test-namespace", nil)
		apiServer.Router().ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
			t.Logf("Response body: %s", recorder.Body.String())
			return
		}

		var response AgentResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Verify status fields are included and correct
		if response.Status != string(acp.AgentStatusReady) {
			t.Errorf("Expected status %s, got %s", string(acp.AgentStatusReady), response.Status)
		}
		if response.StatusDetail != "Everything is working" {
			t.Errorf("Expected status detail %q, got %q", "Everything is working", response.StatusDetail)
		}
		if !response.Ready {
			t.Error("Expected ready to be true")
		}

		// Verify MCP server status is embedded in the MCPServer config
		mcpServer, ok := response.MCPServers["mcp1"]
		if !ok {
			t.Error("Expected MCPServers to have key 'mcp1'")
		} else {
			if mcpServer.Status != "Ready" {
				t.Errorf("Expected MCP status %q, got %q", "Ready", mcpServer.Status)
			}
			if mcpServer.StatusDetail != "Connected to MCP server" {
				t.Errorf("Expected MCP status detail %q, got %q", "Connected to MCP server", mcpServer.StatusDetail)
			}
			if !mcpServer.Ready {
				t.Error("Expected MCP ready to be true")
			}
		}
	})

	// Test GET /agents (list)
	t.Run("GET /agents (list)", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/agents?namespace=status-test-namespace", nil)
		apiServer.Router().ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
			t.Logf("Response body: %s", recorder.Body.String())
			return
		}

		var response []AgentResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response) == 0 {
			t.Fatal("Expected at least one agent in response")
		}

		// Find our test agent in the response
		var testAgentResponse AgentResponse
		found := false
		for _, agentResp := range response {
			if agentResp.Name == "status-test-agent" {
				testAgentResponse = agentResp
				found = true
				break
			}
		}

		if !found {
			t.Fatal("Test agent not found in response")
		}

		// Verify status fields are included and correct
		if testAgentResponse.Status != string(acp.AgentStatusReady) {
			t.Errorf("Expected status %s, got %s", string(acp.AgentStatusReady), testAgentResponse.Status)
		}
		if testAgentResponse.StatusDetail != "Everything is working" {
			t.Errorf("Expected status detail %q, got %q", "Everything is working", testAgentResponse.StatusDetail)
		}
		if !testAgentResponse.Ready {
			t.Error("Expected ready to be true")
		}

		// Verify MCP server status is embedded in the MCPServer config
		mcpServer, ok := testAgentResponse.MCPServers["mcp1"]
		if !ok {
			t.Error("Expected MCPServers to have key 'mcp1'")
		} else {
			if mcpServer.Status != "Ready" {
				t.Errorf("Expected MCP status %q, got %q", "Ready", mcpServer.Status)
			}
			if mcpServer.StatusDetail != "Connected to MCP server" {
				t.Errorf("Expected MCP status detail %q, got %q", "Connected to MCP server", mcpServer.StatusDetail)
			}
			if !mcpServer.Ready {
				t.Error("Expected MCP ready to be true")
			}
		}
	})
}
