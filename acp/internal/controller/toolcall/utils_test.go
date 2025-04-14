package toolcall

import (
	"context"
	"fmt"
	"time"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/trace/noop"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

var fakeSpanContext = &acp.SpanContext{TraceID: "0123456789abcdef", SpanID: "fedcba9876543210"}

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

var testSecret = &TestSecret{
	name: "test-secret",
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
	if t.channelType == acp.ContactChannelTypeSlack {
		contactChannel.Spec.Slack = &acp.SlackChannelConfig{
			ChannelOrUserID: "C12345678",
		}
	} else if t.channelType == acp.ContactChannelTypeEmail {
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

// TestToolCall represents a test ToolCall resource
type TestToolCall struct {
	name      string
	toolName  string
	arguments string
	toolType  acp.ToolType
	toolCall  *acp.ToolCall
}

func (t *TestToolCall) SetupWithStatus(ctx context.Context, status acp.ToolCallStatus) *acp.ToolCall {
	toolCall := t.Setup(ctx)
	toolCall.Status = status
	Expect(k8sClient.Status().Update(ctx, toolCall)).To(Succeed())
	t.toolCall = toolCall
	return toolCall
}

func (t *TestToolCall) Setup(ctx context.Context) *acp.ToolCall {
	By("creating the toolcall")
	toolCall := &acp.ToolCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: acp.ToolCallSpec{
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
	_ = k8sClient.Delete(ctx, toolCall) // Delete if exists
	err := k8sClient.Create(ctx, toolCall)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, toolCall)).To(Succeed())
	t.toolCall = toolCall
	return toolCall
}

func (t *TestToolCall) Teardown(ctx context.Context) {
	By("deleting the taskruntoolcall")
	_ = k8sClient.Delete(ctx, t.toolCall)
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

// reconciler creates a new reconciler for testing
func reconciler() (*ToolCallReconciler, *record.FakeRecorder) {
	By("creating a test reconciler")
	recorder := record.NewFakeRecorder(10)

	reconciler := &ToolCallReconciler{
		Client:   k8sClient,
		Scheme:   k8sClient.Scheme(),
		recorder: recorder,
		Tracer:   noop.NewTracerProvider().Tracer("test"),
	}

	// Set the MCPManager field directly using type assertion
	reconciler.MCPManager = &MockMCPManager{
		NeedsApproval: false,
	}

	return reconciler, recorder
}

// SetupTestApprovalConfig contains optional configuration for setupTestApprovalResources
type SetupTestApprovalConfig struct {
	ToolCallStatus     *acp.ToolCallStatus
	ToolCallName       string
	ToolCallArgs       string
	ContactChannelType acp.ContactChannelType
}

// setupTestApprovalResources sets up all resources needed for testing approval
func setupTestApprovalResources(ctx context.Context, config *SetupTestApprovalConfig) (*acp.ToolCall, func()) {
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

	name := "test-mcp-with-approval-trtc"
	args := `{"url": "https://swapi.dev/api/people/1"}`
	if config != nil {
		if config.ToolCallName != "" {
			name = config.ToolCallName
		}
		if config.ToolCallArgs != "" {
			args = config.ToolCallArgs
		}
	}

	toolCall := &TestToolCall{
		name:      name,
		toolName:  testMCPServer.name + "__fetch",
		arguments: args,
		toolType:  acp.ToolTypeMCP,
	}

	status := acp.ToolCallStatus{
		Phase:        acp.ToolCallPhasePending,
		Status:       acp.ToolCallStatusTypeReady,
		StatusDetail: "Setup complete",
		StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
		SpanContext:  fakeSpanContext,
	}

	if config != nil && config.ToolCallStatus != nil {
		config.ToolCallStatus.SpanContext = fakeSpanContext
		status = *config.ToolCallStatus
	}

	tc := toolCall.SetupWithStatus(ctx, status)

	return tc, func() {
		toolCall.Teardown(ctx)
		testMCPServer.Teardown(ctx)
		testContactChannel.Teardown(ctx)
		testSecret.Teardown(ctx)
	}
}

// TODO: Combine the below config with the above mcp/approval flow config, extremely similar
var testHumanContactTool = &TestToolCall{
	name:     "test-human-contact-tool",
	toolName: "test-human-contact-tool",
	toolType: acp.ToolTypeHumanContact,
}

// SetupTestHumanContactConfig contains optional configuration for setupTestHumanContactResources
type SetupTestHumanContactConfig struct {
	ToolCallStatus     *acp.ToolCallStatus
	ToolCallName       string
	ToolCallArgs       string
	ContactChannelType acp.ContactChannelType
}

// setupTestHumanContactResources sets up all resources needed for testing human contact
func setupTestHumanContactResources(ctx context.Context, config *SetupTestHumanContactConfig) (*acp.ToolCall, func()) {
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

	By("creating the human contact tool")
	testHumanContactTool.SetupWithStatus(ctx, acp.ToolCallStatus{
		Ready:  true,
		Status: "Ready",
	})

	name := "test-human-contact-tc"
	args := `{"message": "This is a test human contact message", "options": ["Yes", "No"]}`
	if config != nil {
		if config.ToolCallName != "" {
			name = config.ToolCallName
		}
		if config.ToolCallArgs != "" {
			args = config.ToolCallArgs
		}
	}

	toolCall := &TestToolCall{
		name:      name,
		toolName:  fmt.Sprintf("%s__%s", testContactChannel.name, testHumanContactTool.name),
		arguments: args,
		toolType:  acp.ToolTypeHumanContact,
	}

	status := acp.ToolCallStatus{
		Phase:        acp.ToolCallPhasePending,
		Status:       acp.ToolCallStatusTypeReady,
		StatusDetail: "Setup complete",
		StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
		SpanContext:  fakeSpanContext,
	}

	if config != nil && config.ToolCallStatus != nil {
		status = *config.ToolCallStatus
	}

	tc := toolCall.SetupWithStatus(ctx, status)

	return tc, func() {
		testHumanContactTool.Teardown(ctx)
		testContactChannel.Teardown(ctx)
		testSecret.Teardown(ctx)
	}
}
