package mcpmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// MCPServerManager manages MCP server connections and tools
var _ MCPManagerInterface = &MCPServerManager{}

type MCPServerManager struct {
	connections map[string]*MCPConnection
	mu          sync.RWMutex
	client      ctrlclient.Client // Kubernetes client for accessing resources
}

type MCPManagerInterface interface {
	CallTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (string, error)
}

// MCPConnection represents a connection to an MCP server
type MCPConnection struct {
	// ServerName is the name of the MCPServer resource
	ServerName string
	// ServerType is "stdio" or "http"
	ServerType string
	// Client is the MCP client
	Client mcpclient.MCPClient
	// Tools is the list of tools provided by this server
	Tools []acp.MCPTool
	// LastConnectTime records when this connection was established
	LastConnectTime time.Time
}

// NewMCPServerManager creates a new MCPServerManager
func NewMCPServerManager() *MCPServerManager {
	return &MCPServerManager{
		connections: make(map[string]*MCPConnection),
		mu:          sync.RWMutex{},
	}
}

// NewMCPServerManagerWithClient creates a new MCPServerManager with a Kubernetes client
func NewMCPServerManagerWithClient(c ctrlclient.Client) *MCPServerManager {
	return &MCPServerManager{
		connections: make(map[string]*MCPConnection),
		mu:          sync.RWMutex{},
		client:      c,
	}
}

// GetConnection returns the MCPConnection for the given server name
func (m *MCPServerManager) GetConnection(serverName string) (*MCPConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.connections[serverName]
	return conn, exists
}

// convertEnvVars converts acp EnvVar to string slice of env vars
func (m *MCPServerManager) convertEnvVars(ctx context.Context, envVars []acp.EnvVar, namespace string) ([]string, error) {
	env := make([]string, 0, len(envVars))
	for _, e := range envVars {
		// Case 1: Direct value
		if e.Value != "" {
			env = append(env, fmt.Sprintf("%s=%s", e.Name, e.Value))
			continue
		}

		// Case 2: Value from secret reference
		if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
			secretRef := e.ValueFrom.SecretKeyRef

			// If we don't have a Kubernetes client, we can't resolve secrets
			if m.client == nil {
				return nil, fmt.Errorf("cannot resolve secret reference for env var %s: no Kubernetes client available", e.Name)
			}

			// Fetch the secret from Kubernetes
			var secret corev1.Secret
			if err := m.client.Get(ctx, types.NamespacedName{
				Name:      secretRef.Name,
				Namespace: namespace,
			}, &secret); err != nil {
				return nil, fmt.Errorf("failed to get secret %s for env var %s: %w", secretRef.Name, e.Name, err)
			}

			// Get the value from the secret
			secretValue, exists := secret.Data[secretRef.Key]
			if !exists {
				return nil, fmt.Errorf("key %s not found in secret %s for env var %s", secretRef.Key, secretRef.Name, e.Name)
			}

			// Add the environment variable with the secret value
			env = append(env, fmt.Sprintf("%s=%s", e.Name, string(secretValue)))
		}
	}
	return env, nil
}

// ConnectServer establishes a connection to an MCP server
func (m *MCPServerManager) ConnectServer(ctx context.Context, mcpServer *acp.MCPServer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use a timeout context to prevent hanging during connection
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Check if we already have a connection for this server
	if conn, exists := m.connections[mcpServer.Name]; exists {
		// If the server exists and the specs are the same, reuse the connection
		// TODO: Add logic to detect if specs changed and reconnect if needed
		if conn.ServerType == mcpServer.Spec.Transport {
			return nil
		}

		// Clean up existing connection
		m.disconnectServerLocked(mcpServer.Name)
	}

	var mcpClient mcpclient.MCPClient
	var err error

	if mcpServer.Spec.Transport == "stdio" {
		// Convert environment variables, resolving any secret references
		envVars, err := m.convertEnvVars(connectCtx, mcpServer.Spec.Env, mcpServer.Namespace)
		if err != nil {
			return fmt.Errorf("failed to process environment variables: %w", err)
		}

		// Create a stdio-based MCP client
		// The cmd member will be set inside the NewStdioMCPClient function
		mcpClient, err = mcpclient.NewStdioMCPClient(mcpServer.Spec.Command, envVars, mcpServer.Spec.Args...)
		if err != nil {
			return fmt.Errorf("failed to create stdio MCP client: %w", err)
		}
	} else if mcpServer.Spec.Transport == "http" {
		// Create an SSE-based MCP client for HTTP connections
		mcpClient, err = mcpclient.NewSSEMCPClient(mcpServer.Spec.URL)
		if err != nil {
			return fmt.Errorf("failed to create SSE MCP client: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported MCP server transport: %s", mcpServer.Spec.Transport)
	}

	// Ensure client is cleaned up on any error
	clientCreated := true
	defer func() {
		if err != nil && clientCreated && mcpClient != nil {
			if closeErr := mcpClient.Close(); closeErr != nil {
				fmt.Printf("Error closing mcpClient during error handling: %v\n", closeErr)
			}
		}
	}()

	// Initialize the client with timeout context
	_, err = mcpClient.Initialize(connectCtx, mcp.InitializeRequest{})
	if err != nil {
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// Get the list of tools with timeout context
	toolsResp, err := mcpClient.ListTools(connectCtx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert tools to acp format
	tools := make([]acp.MCPTool, 0, len(toolsResp.Tools))
	for _, tool := range toolsResp.Tools {
		// Handle the InputSchema properly
		var inputSchemaBytes []byte
		var schemaErr error

		if len(tool.RawInputSchema) > 0 {
			// Use RawInputSchema if available (preferred)
			inputSchemaBytes = tool.RawInputSchema
		} else {
			// Otherwise, use the structured InputSchema and ensure required is an array
			schema := tool.InputSchema

			// Ensure required is not null
			if schema.Required == nil {
				schema.Required = []string{}
			}

			inputSchemaBytes, schemaErr = json.Marshal(schema)
			if schemaErr != nil {
				// Log the error but continue
				fmt.Printf("Error marshaling input schema for tool %s: %v\n", tool.Name, schemaErr)
				// Use a minimal valid schema as fallback
				inputSchemaBytes = []byte(`{"type":"object","properties":{},"required":[]}`)
			}
		}

		tools = append(tools, acp.MCPTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: runtime.RawExtension{Raw: inputSchemaBytes},
		})
	}

	// Store the connection
	m.connections[mcpServer.Name] = &MCPConnection{
		ServerName:      mcpServer.Name,
		ServerType:      mcpServer.Spec.Transport,
		Client:          mcpClient,
		Tools:           tools,
		LastConnectTime: time.Now(),
	}

	return nil
}

// DisconnectServer closes the connection to an MCP server
func (m *MCPServerManager) DisconnectServer(serverName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectServerLocked(serverName)
}

// disconnectServerLocked is the internal implementation of DisconnectServer
// that assumes the lock is already held
func (m *MCPServerManager) disconnectServerLocked(serverName string) {
	conn, exists := m.connections[serverName]
	if !exists {
		return
	}

	// Close the connection with a timeout to avoid hanging
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Closing routine with timeout
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Close the client connection
		if conn.Client != nil {
			if err := conn.Client.Close(); err != nil {
				fmt.Printf("Error closing MCP client connection for %s: %v\n", serverName, err)
			}
		}
	}()

	// Wait for close to finish or timeout
	select {
	case <-done:
		// Closed successfully
	case <-closeCtx.Done():
		fmt.Printf("Warning: Timeout while closing MCP client connection for %s\n", serverName)
	}

	// The MCP client's Close method should handle process termination
	// We don't have direct access to the underlying process anymore

	// Remove the connection from the map
	delete(m.connections, serverName)
}

// GetTools returns the tools for the given server
func (m *MCPServerManager) GetTools(serverName string) ([]acp.MCPTool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.connections[serverName]
	if !exists {
		return nil, false
	}
	return conn.Tools, true
}

// GetToolsForAgent returns all tools from the MCP servers referenced by the agent
func (m *MCPServerManager) GetToolsForAgent(agent *acp.Agent) []acp.MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allTools []acp.MCPTool
	for _, serverRef := range agent.Spec.MCPServers {
		conn, exists := m.connections[serverRef.Name]
		if !exists {
			continue
		}
		allTools = append(allTools, conn.Tools...)
	}
	return allTools
}

// CallTool calls a tool on an MCP server
func (m *MCPServerManager) CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (string, error) {
	// Create a timeout context to prevent hanging calls
	callCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	m.mu.RLock()
	conn, exists := m.connections[serverName]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("MCP server not found: %s", serverName)
	}

	// Check if the connection is still alive
	if conn.Client == nil {
		return "", fmt.Errorf("MCP server connection is invalid for %s", serverName)
	}

	result, err := conn.Client.CallTool(callCtx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      toolName,
			Arguments: arguments,
		},
	})
	if err != nil {
		// Check if it was a context timeout
		if callCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timeout calling tool %s on server %s: operation took too long", toolName, serverName)
		}
		return "", fmt.Errorf("error calling tool %s on server %s: %w", toolName, serverName, err)
	}

	// Process the result
	var output string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			output += textContent.Text
		} else {
			// Handle other content types as needed
			output += "[Non-text content]"
		}
	}

	if result.IsError {
		return output, fmt.Errorf("tool execution error: %s", output)
	}

	return output, nil
}

// FindServerForTool finds which MCP server provides a given tool
// Format of the tool name is expected to be "serverName__toolName"
func (m *MCPServerManager) FindServerForTool(fullToolName string) (serverName string, toolName string, found bool) {
	// In our implementation, we'll use serverName__toolName as the format
	parts := strings.SplitN(fullToolName, "__", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	serverName = parts[0]
	toolName = parts[1]

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if the server exists
	conn, exists := m.connections[serverName]
	if !exists {
		return "", "", false
	}

	// Check if the tool exists on this server
	for _, tool := range conn.Tools {
		if tool.Name == toolName {
			return serverName, toolName, true
		}
	}

	return "", "", false
}

// Close closes all connections
func (m *MCPServerManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a list of server names to avoid modifying the map during iteration
	serverNames := make([]string, 0, len(m.connections))
	for serverName := range m.connections {
		serverNames = append(serverNames, serverName)
	}

	// Close each connection
	for _, serverName := range serverNames {
		m.disconnectServerLocked(serverName)
	}

	// Clear the connections map
	m.connections = make(map[string]*MCPConnection)
}
