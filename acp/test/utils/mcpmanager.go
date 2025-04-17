package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MockMCPManager implements the MCPManagerInterface for testing
type MockMCPManager struct {
	NeedsApproval bool                       // Flag to control if mock MCP tools need approval
	MockTools     map[string][]acp.MCPTool   // Mock tools for each server
	Connected     map[string]bool            // Track which servers are connected
	mu            sync.RWMutex               // Mutex for thread safety
}

// CallTool implements the MCPManager.CallTool method
func (m *MockMCPManager) CallTool(
	ctx context.Context,
	serverName, toolName string,
	args map[string]interface{},
) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if server exists and is connected
	if !m.isServerConnected(serverName) {
		return "", fmt.Errorf("MCP server not found or not connected: %s", serverName)
	}
	
	// If we're testing the approval flow, return an error to prevent direct execution
	if m.NeedsApproval {
		return "", fmt.Errorf("tool requires approval")
	}

	// For non-approval tests, pretend to add the numbers
	if a, ok := args["a"].(float64); ok {
		if b, ok := args["b"].(float64); ok {
			return fmt.Sprintf("%v", a+b), nil
		}
	}

	return "5", nil // Default result
}

// ConnectServer mocks connecting to an MCP server
func (m *MockMCPManager) ConnectServer(ctx context.Context, mcpServer *acp.MCPServer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Initialize maps if needed
	if m.Connected == nil {
		m.Connected = make(map[string]bool)
	}
	if m.MockTools == nil {
		m.MockTools = make(map[string][]acp.MCPTool)
	}
	
	// Mark server as connected
	m.Connected[mcpServer.Name] = true
	
	// If no tools are defined for this server, create some mock tools
	if _, exists := m.MockTools[mcpServer.Name]; !exists {
		// Create default mock tools
		m.MockTools[mcpServer.Name] = []acp.MCPTool{
			{
				Name:        "fetch",
				Description: "Mock fetch tool for testing",
				InputSchema: runtime.RawExtension{
					Raw: []byte(`{"type":"object","properties":{"url":{"type":"string"}},"required":["url"]}`),
				},
			},
			{
				Name:        "calculate",
				Description: "Mock calculation tool for testing",
				InputSchema: runtime.RawExtension{
					Raw: []byte(`{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}},"required":["a","b"]}`),
				},
			},
		}
	}
	
	return nil
}

// DisconnectServer mocks disconnecting from an MCP server
func (m *MockMCPManager) DisconnectServer(serverName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.Connected != nil {
		delete(m.Connected, serverName)
	}
}

// GetTools returns the mock tools for a server
func (m *MockMCPManager) GetTools(serverName string) ([]acp.MCPTool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if !m.isServerConnected(serverName) {
		return nil, false
	}
	
	tools, exists := m.MockTools[serverName]
	return tools, exists
}

// GetToolsForAgent returns all tools from the MCP servers referenced by the agent
func (m *MockMCPManager) GetToolsForAgent(agent *acp.Agent) []acp.MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var allTools []acp.MCPTool
	for _, serverRef := range agent.Spec.MCPServers {
		if tools, exists := m.GetTools(serverRef.Name); exists {
			allTools = append(allTools, tools...)
		}
	}
	return allTools
}

// FindServerForTool mocks finding which server provides a tool
func (m *MockMCPManager) FindServerForTool(fullToolName string) (serverName string, toolName string, found bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Parse the tool name (expected format: "serverName__toolName")
	parts := strings.Split(fullToolName, "__")
	if len(parts) != 2 {
		return "", "", false
	}
	
	serverName = parts[0]
	toolName = parts[1]
	
	// Check if the server exists and is connected
	if !m.isServerConnected(serverName) {
		return "", "", false
	}
	
	// Check if the tool exists on this server
	tools, exists := m.MockTools[serverName]
	if !exists {
		return "", "", false
	}
	
	for _, tool := range tools {
		if tool.Name == toolName {
			return serverName, toolName, true
		}
	}
	
	return "", "", false
}

// Close mocks closing all connections
func (m *MockMCPManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Connected = make(map[string]bool)
}

// Helper method to check if a server is connected
func (m *MockMCPManager) isServerConnected(serverName string) bool {
	if m.Connected == nil {
		return false
	}
	
	connected, exists := m.Connected[serverName]
	return exists && connected
}