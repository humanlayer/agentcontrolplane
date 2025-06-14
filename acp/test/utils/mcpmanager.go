package utils

import (
	"context"
	"fmt"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

type MockMCPManager struct {
	NeedsApproval bool // Flag to control if mock MCP tools need approval
}

// CallTool implements the MCPManager.CallTool method
func (m *MockMCPManager) CallTool(
	ctx context.Context,
	serverName, toolName string,
	args map[string]interface{},
) (string, error) {
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

// GetTools implements the MCPManager.GetTools method
func (m *MockMCPManager) GetTools(serverName string) ([]acp.MCPTool, bool) {
	// Return some mock tools for testing
	tools := []acp.MCPTool{
		{
			Name:        "add",
			Description: "Add two numbers",
		},
	}
	return tools, true
}
