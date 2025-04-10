package mcpmanager

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/mark3labs/mcp-go/mcp"
)

// MockMCPClient mocks the mcpclient.MCPClient interface for testing
type MockMCPClient struct {
	// Results
	initResult     *mcp.InitializeResult
	toolsResult    *mcp.ListToolsResult
	callToolResult *mcp.CallToolResult

	// Errors
	initError     error
	toolsError    error
	callToolError error

	// Tracking calls
	initCallCount     int
	toolsCallCount    int
	callToolCallCount int
	closeCallCount    int

	// Last request arguments
	lastCallToolRequest mcp.CallToolRequest
}

// NewMockMCPClient creates a new mock client with default responses
func NewMockMCPClient() *MockMCPClient {
	return &MockMCPClient{
		initResult: &mcp.InitializeResult{},
		toolsResult: &mcp.ListToolsResult{
			Tools: []mcp.Tool{
				{
					Name:           "test_tool",
					Description:    "Test tool for testing",
					RawInputSchema: []byte(`{"type":"object","properties":{"param1":{"type":"string"}}}`),
				},
			},
		},
		callToolResult: &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Mock result",
				},
			},
			IsError: false,
		},
	}
}

// Initialize implements mcpclient.MCPClient
func (m *MockMCPClient) Initialize(ctx context.Context, req mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	m.initCallCount++
	return m.initResult, m.initError
}

// ListTools implements mcpclient.MCPClient
func (m *MockMCPClient) ListTools(ctx context.Context, req mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	m.toolsCallCount++
	return m.toolsResult, m.toolsError
}

// CallTool implements mcpclient.MCPClient
func (m *MockMCPClient) CallTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.callToolCallCount++
	m.lastCallToolRequest = req
	return m.callToolResult, m.callToolError
}

// Close implements mcpclient.MCPClient
func (m *MockMCPClient) Close() error {
	m.closeCallCount++
	return nil
}

// Additional methods required by the interface
// These are stubs to satisfy the interface but aren't used in our tests
func (m *MockMCPClient) Ping(ctx context.Context) error { return nil }

func (m *MockMCPClient) ListResources(ctx context.Context, req mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *MockMCPClient) ListResourceTemplates(ctx context.Context, req mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *MockMCPClient) ReadResource(ctx context.Context, req mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return nil, nil
}
func (m *MockMCPClient) Subscribe(ctx context.Context, req mcp.SubscribeRequest) error { return nil }
func (m *MockMCPClient) Unsubscribe(ctx context.Context, req mcp.UnsubscribeRequest) error {
	return nil
}

func (m *MockMCPClient) ListPrompts(ctx context.Context, req mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

func (m *MockMCPClient) GetPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return nil, nil
}
func (m *MockMCPClient) SetLevel(ctx context.Context, req mcp.SetLevelRequest) error { return nil }
func (m *MockMCPClient) Complete(ctx context.Context, req mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return nil, nil
}
func (m *MockMCPClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {}

// Helper methods for tests
func (m *MockMCPClient) SetToolsResult(tools []mcp.Tool) {
	m.toolsResult = &mcp.ListToolsResult{Tools: tools}
}

func (m *MockMCPClient) SetToolsError(err error) {
	m.toolsError = err
}

func (m *MockMCPClient) SetCallToolResult(result *mcp.CallToolResult) {
	m.callToolResult = result
}

func (m *MockMCPClient) SetCallToolError(err error) {
	m.callToolError = err
}

func (m *MockMCPClient) GetCallToolCount() int {
	return m.callToolCallCount
}

func (m *MockMCPClient) GetLastCallToolRequest() mcp.CallToolRequest {
	return m.lastCallToolRequest
}

func (m *MockMCPClient) GetInitializeCount() int {
	return m.initCallCount
}

func (m *MockMCPClient) GetListToolsCount() int {
	return m.toolsCallCount
}

func (m *MockMCPClient) GetCloseCount() int {
	return m.closeCallCount
}

// A minimal dummy client just to test client assignment
type dummyClient struct {
	ctrlclient.Client
}

func TestMCPManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MCP Manager Suite")
}

var _ = Describe("MCPServerManager", func() {
	var (
		manager    *MCPServerManager
		mockClient *MockMCPClient
		ctx        context.Context
		cancelFunc context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancelFunc = context.WithCancel(context.Background())
		mockClient = NewMockMCPClient()

		// For the main test, we'll use a nil client since we're not testing secret retrieval here
		// The secret retrieval is tested in envvar_test.go

		// Create the manager with nil client
		manager = NewMCPServerManager()

		// Add a test server directly to the connections map
		manager.connections["test-server"] = &MCPConnection{
			ServerName: "test-server",
			ServerType: "stdio",
			Client:     mockClient,
			Tools: []acp.MCPTool{
				{
					Name:        "test_tool",
					Description: "A test tool",
					InputSchema: runtime.RawExtension{Raw: []byte(`{"type":"object"}`)},
				},
			},
		}
	})

	AfterEach(func() {
		cancelFunc()
		manager.Close()
	})

	Describe("Constructor functions", func() {
		It("should create a new MCPServerManager with no client", func() {
			m := NewMCPServerManager()
			Expect(m).NotTo(BeNil())
			Expect(m.connections).NotTo(BeNil())
			Expect(m.connections).To(BeEmpty())
			Expect(m.client).To(BeNil())
		})

		It("should create a new MCPServerManager with a client", func() {
			// Create a dummy client
			dummyClient := &dummyClient{}

			// Create manager with mock client
			clientManager := NewMCPServerManagerWithClient(dummyClient)

			Expect(clientManager).NotTo(BeNil())
			Expect(clientManager.connections).NotTo(BeNil())
			Expect(clientManager.client).NotTo(BeNil())
			Expect(clientManager.client).To(Equal(dummyClient))
		})
	})

	Describe("GetConnection", func() {
		It("should return an existing connection", func() {
			conn, exists := manager.GetConnection("test-server")
			Expect(exists).To(BeTrue())
			Expect(conn).NotTo(BeNil())
			Expect(conn.ServerName).To(Equal("test-server"))
		})

		It("should return false for non-existent connections", func() {
			conn, exists := manager.GetConnection("non-existent")
			Expect(exists).To(BeFalse())
			Expect(conn).To(BeNil())
		})
	})

	Describe("GetTools", func() {
		It("should return tools for an existing server", func() {
			tools, exists := manager.GetTools("test-server")
			Expect(exists).To(BeTrue())
			Expect(tools).To(HaveLen(1))
			Expect(tools[0].Name).To(Equal("test_tool"))
		})

		It("should return false for non-existent servers", func() {
			tools, exists := manager.GetTools("non-existent")
			Expect(exists).To(BeFalse())
			Expect(tools).To(BeNil())
		})
	})

	Describe("GetToolsForAgent", func() {
		It("should return tools from all referenced servers", func() {
			// Add another server
			anotherMock := NewMockMCPClient()
			manager.connections["another-server"] = &MCPConnection{
				ServerName: "another-server",
				ServerType: "stdio",
				Client:     anotherMock,
				Tools: []acp.MCPTool{
					{
						Name:        "another_tool",
						Description: "Another test tool",
						InputSchema: runtime.RawExtension{Raw: []byte(`{"type":"object"}`)},
					},
				},
			}

			// Create a test agent that references both servers
			agent := &acp.Agent{
				Spec: acp.AgentSpec{
					MCPServers: []acp.LocalObjectReference{
						{Name: "test-server"},
						{Name: "another-server"},
					},
				},
			}

			// Get tools for the agent
			tools := manager.GetToolsForAgent(agent)
			Expect(tools).To(HaveLen(2))

			// Check both tools are present
			foundTools := make(map[string]bool)
			for _, tool := range tools {
				foundTools[tool.Name] = true
			}
			Expect(foundTools).To(HaveKey("test_tool"))
			Expect(foundTools).To(HaveKey("another_tool"))
		})

		It("should ignore references to non-existent servers", func() {
			agent := &acp.Agent{
				Spec: acp.AgentSpec{
					MCPServers: []acp.LocalObjectReference{
						{Name: "test-server"},
						{Name: "non-existent"},
					},
				},
			}

			tools := manager.GetToolsForAgent(agent)
			Expect(tools).To(HaveLen(1))
			Expect(tools[0].Name).To(Equal("test_tool"))
		})
	})

	Describe("CallTool", func() {
		It("should successfully call a tool on an MCP server", func() {
			// Set up response
			mockClient.SetCallToolResult(&mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Success",
					},
				},
				IsError: false,
			})

			// Call the tool
			result, err := manager.CallTool(ctx, "test-server", "test_tool", map[string]interface{}{
				"param1": "value1",
			})

			// Verify results
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("Success"))
			Expect(mockClient.GetCallToolCount()).To(Equal(1))

			// Check request details
			req := mockClient.GetLastCallToolRequest()
			Expect(req.Params.Name).To(Equal("test_tool"))
			Expect(req.Params.Arguments).To(HaveKeyWithValue("param1", "value1"))
		})

		It("should return an error when the server doesn't exist", func() {
			_, err := manager.CallTool(ctx, "non-existent", "tool", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MCP server not found"))
		})

		It("should return an error when the tool call fails", func() {
			mockClient.SetCallToolError(errors.New("call failed"))

			_, err := manager.CallTool(ctx, "test-server", "test_tool", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("call failed"))
		})

		It("should return an error when the tool returns IsError=true", func() {
			mockClient.SetCallToolResult(&mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error message",
					},
				},
				IsError: true,
			})

			_, err := manager.CallTool(ctx, "test-server", "test_tool", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Error message"))
		})
	})

	Describe("FindServerForTool", func() {
		It("should find the server and tool for a valid formatted name", func() {
			serverName, toolName, found := manager.FindServerForTool("test-server__test_tool")
			Expect(found).To(BeTrue())
			Expect(serverName).To(Equal("test-server"))
			Expect(toolName).To(Equal("test_tool"))
		})

		It("should return false for an invalid format", func() {
			_, _, found := manager.FindServerForTool("invalid-format")
			Expect(found).To(BeFalse())
		})

		It("should return false for a non-existent server", func() {
			_, _, found := manager.FindServerForTool("non-existent__tool")
			Expect(found).To(BeFalse())
		})

		It("should return false for a non-existent tool", func() {
			_, _, found := manager.FindServerForTool("test-server__non-existent")
			Expect(found).To(BeFalse())
		})
	})

	Describe("DisconnectServer", func() {
		It("should remove the server from connections", func() {
			// Verify connection exists
			_, exists := manager.GetConnection("test-server")
			Expect(exists).To(BeTrue())

			// Disconnect server
			manager.DisconnectServer("test-server")

			// Verify connection is removed
			_, exists = manager.GetConnection("test-server")
			Expect(exists).To(BeFalse())

			// Verify Close was called on client
			Expect(mockClient.GetCloseCount()).To(Equal(1))
		})

		It("should do nothing for non-existent servers", func() {
			// This shouldn't panic
			manager.DisconnectServer("non-existent")
		})
	})

	Describe("Close", func() {
		It("should close all connections", func() {
			// Add another connection
			anotherMock := NewMockMCPClient()
			manager.connections["another-server"] = &MCPConnection{
				ServerName: "another-server",
				ServerType: "stdio",
				Client:     anotherMock,
			}

			// Verify two connections exist
			Expect(manager.connections).To(HaveLen(2))

			// Close all connections
			manager.Close()

			// Verify connections map is empty
			Expect(manager.connections).To(BeEmpty())

			// Verify Close was called on both clients
			Expect(mockClient.GetCloseCount()).To(Equal(1))
			Expect(anotherMock.GetCloseCount()).To(Equal(1))
		})
	})

	// convertEnvVars tests are in envvar_test.go

	// Testing ConnectServer would require additional mocking of NewStdioMCPClient
	// and NewSSEMCPClient, which would require refactoring the production code
	// to allow dependency injection
})
