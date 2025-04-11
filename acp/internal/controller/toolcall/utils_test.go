package taskruntoolcall

import (
	"context"
	"fmt"
	"time"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

// Test tool instances for different types
var addTool = &TestTool{
	name:     "add",
	toolType: "function",
}

var testContactChannel = &TestContactChannel{
	name:        "test-contact-channel",
	channelType: acp.ContactChannelTypeSlack,
	secretName:  testSecret.name,
}

var testMCPServer = &TestMCPServer{
	name:                   "test-mcp-server",
	needsApproval:          true,
	approvalContactChannel: testContactChannel.name,
}

var testMCPTool = &TestMCPTool{
	name:        "test-mcp-server-test-tool",
	mcpServer:   testMCPServer.name,
	mcpToolName: "test-tool",
}

var trtcForAddTool = &TestTaskRunToolCall{
	name:      "test-taskruntoolcall",
	toolName:  addTool.name,
	arguments: `{"a": 2, "b": 3}`,
}

var testSecret = &TestSecret{
	name: "test-secret",
}

// TestTool represents a test Tool resource
type TestTool struct {
	name     string
	toolType string
	tool     *acp.Tool
}

// TestSecret represents a test secret for storing API keys
type TestSecret struct {
	name   string
	secret *corev1.Secret
}

// TestContactChannel represents a test ContactChannel resource
type TestContactChannel struct {
	name           string
	channelType    acp.ContactChannelType
	secretName     string
	contactChannel *acp.ContactChannel
}

func (t *TestContactChannel) Setup(ctx context.Context) *acp.ContactChannel {
	By("creating the contact channel")
	contactChannel := &acp.ContactChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: acp.ContactChannelSpec{
			Type: t.channelType,
			APIKeyFrom: acp.APIKeySource{
				SecretKeyRef: acp.SecretKeyRef{
					Name: t.secretName,
					Key:  "api-key",
				},
			},
		},
	}

	// Add specific config based on channel type
	if t.channelType == "slack" {
		contactChannel.Spec.Slack = &acp.SlackChannelConfig{
			ChannelOrUserID: "C12345678",
		}
	} else if t.channelType == "email" {
		contactChannel.Spec.Email = &acp.EmailChannelConfig{
			Address: "test@example.com",
		}
	}

	_ = k8sClient.Delete(ctx, contactChannel) // Delete if exists
	err := k8sClient.Create(ctx, contactChannel)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, contactChannel)).To(Succeed())
	t.contactChannel = contactChannel
	return contactChannel
}

func (t *TestContactChannel) SetupWithStatus(ctx context.Context, status acp.ContactChannelStatus) *acp.ContactChannel {
	contactChannel := t.Setup(ctx)
	contactChannel.Status = status
	Expect(k8sClient.Status().Update(ctx, contactChannel)).To(Succeed())
	t.contactChannel = contactChannel
	return contactChannel
}

func (t *TestContactChannel) Teardown(ctx context.Context) {
	By("deleting the contact channel")
	_ = k8sClient.Delete(ctx, t.contactChannel)
}

func (t *TestSecret) Setup(ctx context.Context) *corev1.Secret {
	By("creating the secret")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}
	_ = k8sClient.Delete(ctx, secret) // Delete if exists
	err := k8sClient.Create(ctx, secret)
	Expect(err).NotTo(HaveOccurred())
	t.secret = secret
	return secret
}

func (t *TestSecret) Teardown(ctx context.Context) {
	By("deleting the secret")
	_ = k8sClient.Delete(ctx, t.secret)
}

// TestTaskRunToolCall represents a test TaskRunToolCall resource
type TestTaskRunToolCall struct {
	name            string
	toolName        string
	arguments       string
	toolType        acp.ToolType
	taskRunToolCall *acp.TaskRunToolCall
}

func (t *TestTaskRunToolCall) SetupWithStatus(ctx context.Context, status acp.TaskRunToolCallStatus) *acp.TaskRunToolCall {
	taskRunToolCall := t.Setup(ctx)
	taskRunToolCall.Status = status
	Expect(k8sClient.Status().Update(ctx, taskRunToolCall)).To(Succeed())
	t.taskRunToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestTaskRunToolCall) Setup(ctx context.Context) *acp.TaskRunToolCall {
	By("creating the taskruntoolcall")
	taskRunToolCall := &acp.TaskRunToolCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: acp.TaskRunToolCallSpec{
			TaskRef: acp.LocalObjectReference{
				Name: "parent-task",
			},
			ToolRef: acp.LocalObjectReference{
				Name: t.toolName,
			},
			ToolType:  t.toolType,
			Arguments: t.arguments,
		},
	}
	_ = k8sClient.Delete(ctx, taskRunToolCall) // Delete if exists
	err := k8sClient.Create(ctx, taskRunToolCall)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, taskRunToolCall)).To(Succeed())
	t.taskRunToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestTaskRunToolCall) Teardown(ctx context.Context) {
	By("deleting the taskruntoolcall")
	_ = k8sClient.Delete(ctx, t.taskRunToolCall)
}

func (t *TestTool) SetupWithStatus(ctx context.Context, status acp.ToolStatus) *acp.Tool {
	tool := t.Setup(ctx)
	tool.Status = status
	Expect(k8sClient.Status().Update(ctx, tool)).To(Succeed())
	t.tool = tool
	return tool
}

func (t *TestTool) Setup(ctx context.Context) *acp.Tool {
	By("creating the tool")
	tool := &acp.Tool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: acp.ToolSpec{
			ToolType:    t.toolType,
			Name:        t.name,
			Description: "Test tool for " + t.toolType,
			Execute: acp.ToolExecute{
				Builtin: &acp.BuiltinToolSpec{
					Name: t.name,
				},
			},
		},
	}
	_ = k8sClient.Delete(ctx, tool) // Delete if exists
	err := k8sClient.Create(ctx, tool)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, tool)).To(Succeed())
	t.tool = tool
	return tool
}

func (t *TestTool) Teardown(ctx context.Context) {
	By("deleting the tool")
	_ = k8sClient.Delete(ctx, t.tool)
}

// setupTestAddTools sets up all the tools needed for testing
func setupTestAddTool(ctx context.Context) func() {
	addTool.SetupWithStatus(ctx, acp.ToolStatus{
		Ready:  true,
		Status: "Ready",
	})

	return func() {
		addTool.Teardown(ctx)
	}
}

// TestMCPServer represents a test MCPServer resource
type TestMCPServer struct {
	name                   string
	needsApproval          bool
	approvalContactChannel string
	mcpServer              *acp.MCPServer
}

func (t *TestMCPServer) Setup(ctx context.Context) *acp.MCPServer {
	By("creating the MCP server")
	mcpServer := &acp.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: acp.MCPServerSpec{
			Transport: "stdio",
		},
	}

	if t.needsApproval && t.approvalContactChannel != "" {
		mcpServer.Spec.ApprovalContactChannel = &acp.LocalObjectReference{
			Name: t.approvalContactChannel,
		}
	}

	_ = k8sClient.Delete(ctx, mcpServer) // Delete if exists
	err := k8sClient.Create(ctx, mcpServer)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, mcpServer)).To(Succeed())
	t.mcpServer = mcpServer
	return mcpServer
}

func (t *TestMCPServer) SetupWithStatus(ctx context.Context, status acp.MCPServerStatus) *acp.MCPServer {
	mcpServer := t.Setup(ctx)
	mcpServer.Status = status
	Expect(k8sClient.Status().Update(ctx, mcpServer)).To(Succeed())
	t.mcpServer = mcpServer
	return mcpServer
}

func (t *TestMCPServer) Teardown(ctx context.Context) {
	By("deleting the MCP server")
	_ = k8sClient.Delete(ctx, t.mcpServer)
}

// MockMCPManager is a struct that mocks the essential MCPServerManager functionality for testing
type MockMCPManager struct {
	NeedsApproval bool // Flag to control if mock MCP tools need approval
}

// CallTool implements the MCPManager.CallTool method
func (m *MockMCPManager) CallTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (string, error) {
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

// TestMCPTool represents a test Tool resource for MCP
type TestMCPTool struct {
	name        string
	mcpServer   string
	mcpToolName string
	tool        *acp.Tool
}

func (t *TestMCPTool) Setup(ctx context.Context) *acp.Tool {
	By("creating the MCP tool")
	toolName := t.mcpServer + "__" + t.mcpToolName
	tool := &acp.Tool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: acp.ToolSpec{
			ToolType:    "function",
			Name:        toolName,
			Description: "Test MCP tool",
			Execute: acp.ToolExecute{
				Builtin: &acp.BuiltinToolSpec{
					Name: "add",
				},
			},
		},
	}
	_ = k8sClient.Delete(ctx, tool) // Delete if exists
	err := k8sClient.Create(ctx, tool)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, tool)).To(Succeed())
	t.tool = tool
	return tool
}

func (t *TestMCPTool) SetupWithStatus(ctx context.Context, status acp.ToolStatus) *acp.Tool {
	tool := t.Setup(ctx)
	tool.Status = status
	Expect(k8sClient.Status().Update(ctx, tool)).To(Succeed())
	t.tool = tool
	return tool
}

func (t *TestMCPTool) Teardown(ctx context.Context) {
	By("deleting the MCP tool")
	_ = k8sClient.Delete(ctx, t.tool)
}

// reconciler creates a new reconciler for testing
func reconciler() (*TaskRunToolCallReconciler, *record.FakeRecorder) {
	By("creating a test reconciler")
	recorder := record.NewFakeRecorder(10)

	reconciler := &TaskRunToolCallReconciler{
		Client:   k8sClient,
		Scheme:   k8sClient.Scheme(),
		recorder: recorder,
	}

	// Set the MCPManager field directly using type assertion
	reconciler.MCPManager = &MockMCPManager{
		NeedsApproval: false,
	}

	return reconciler, recorder
}

// SetupTestApprovalConfig contains optional configuration for setupTestApprovalResources
type SetupTestApprovalConfig struct {
	TaskRunToolCallStatus *acp.TaskRunToolCallStatus
	TaskRunToolCallName   string
	TaskRunToolCallArgs   string
	ContactChannelType    acp.ContactChannelType
}

// setupTestApprovalResources sets up all resources needed for testing approval
func setupTestApprovalResources(ctx context.Context, config *SetupTestApprovalConfig) (*acp.TaskRunToolCall, func()) {
	By("creating the secret")
	testSecret.Setup(ctx)
	By("creating the contact channel")

	// Set contact channel type based on config or default to ContactChannelTypeSlack
	channelType := acp.ContactChannelTypeSlack
	if config != nil && config.ContactChannelType != "" {
		switch config.ContactChannelType {
		case "email":
			channelType = acp.ContactChannelTypeEmail
		default:
			channelType = acp.ContactChannelTypeSlack
		}
	}

	testContactChannel.channelType = channelType
	testContactChannel.SetupWithStatus(ctx, acp.ContactChannelStatus{
		Ready:  true,
		Status: "Ready",
	})
	By("creating the MCP server")
	testMCPServer.SetupWithStatus(ctx, acp.MCPServerStatus{
		Connected: true,
		Status:    "Ready",
	})
	By("creating the MCP tool")
	mcpTool := testMCPTool.SetupWithStatus(ctx, acp.ToolStatus{
		Ready:  true,
		Status: "Ready",
	})

	name := "test-mcp-with-approval-trtc"
	args := `{"a": 2, "b": 3}`
	if config != nil {
		if config.TaskRunToolCallName != "" {
			name = config.TaskRunToolCallName
		}
		if config.TaskRunToolCallArgs != "" {
			args = config.TaskRunToolCallArgs
		}
	}

	taskRunToolCall := &TestTaskRunToolCall{
		name:      name,
		toolName:  mcpTool.Spec.Name,
		arguments: args,
		toolType:  acp.ToolTypeMCP,
	}

	status := acp.TaskRunToolCallStatus{
		Phase:        acp.TaskRunToolCallPhasePending,
		Status:       acp.TaskRunToolCallStatusTypeReady,
		StatusDetail: "Setup complete",
		StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
	}

	if config != nil && config.TaskRunToolCallStatus != nil {
		status = *config.TaskRunToolCallStatus
	}

	trtc := taskRunToolCall.SetupWithStatus(ctx, status)

	return trtc, func() {
		testMCPTool.Teardown(ctx)
		testMCPServer.Teardown(ctx)
		testContactChannel.Teardown(ctx)
		testSecret.Teardown(ctx)
	}
}
