package taskruntoolcall

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/gomega" //nolint:stylecheck

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayer"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "default"
	timeout       = time.Second * 10
	duration      = time.Second * 10
	interval      = time.Millisecond * 250
)

func createTestTool(name string) *acp.Tool {
	return &acp.Tool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: acp.ToolSpec{
			Description: "A test tool",
			ToolType:    "function",
			Execute: &acp.ToolExecute{
				Builtin: &acp.BuiltinToolSpec{
					FunctionName: "test-function",
				},
			},
		},
	}
}

func createTestContactChannel(name string, channelType acp.ContactChannelType) *acp.ContactChannel {
	cc := &acp.ContactChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: acp.ContactChannelSpec{
			Type: channelType,
			APIKeyFrom: &acp.APIKeySource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name + "-secret",
					},
					Key: "apiKey",
				},
			},
		},
	}
	switch channelType {
	case acp.ContactChannelTypeSlack:
		cc.Spec.Slack = &acp.SlackChannelConfig{ChannelOrUserId: "test-slack-channel"}
	case acp.ContactChannelTypeEmail:
		cc.Spec.Email = &acp.EmailChannelConfig{Address: "test@example.com"}
	}
	return cc
}

func createTestMCPServer(name string, approvalChannelName *string) *acp.MCPServer {
	mcp := &acp.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: acp.MCPServerSpec{
			Address: "test-mcp-server:8080",
		},
	}
	if approvalChannelName != nil {
		mcp.Spec.ApprovalContactChannel = &corev1.LocalObjectReference{Name: *approvalChannelName}
	}
	return mcp
}

func createTestMCPTool(mcpServerName, toolName string) *acp.Tool {
	return &acp.Tool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName + "__" + toolName,
			Namespace: testNamespace,
		},
		Spec: acp.ToolSpec{
			Description: "An MCP test tool",
			ToolType:    acp.ToolTypeMCP,
		},
	}
}

func createTestTaskRunToolCall(name, toolName string, toolType acp.ToolType, args map[string]interface{}) *acp.TaskRunToolCall {
	argsBytes, _ := json.Marshal(args)
	return &acp.TaskRunToolCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: acp.TaskRunToolCallSpec{
			ToolRef: corev1.LocalObjectReference{
				Name: toolName,
			},
			ToolType:  toolType,
			Arguments: string(argsBytes),
		},
		Status: acp.TaskRunToolCallStatus{
			Phase:  acp.TaskRunToolCallPhasePending,
			Status: acp.TaskRunToolCallStatusTypePending,
		},
	}
}

func createTestSecret(name, apiKey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"apiKey": []byte(apiKey),
		},
	}
}

func Setup(ctx context.Context, k8sClient client.Client, obj client.Object) {
	Expect(k8sClient.Create(ctx, obj)).Should(Succeed())

	Eventually(func() error {
		return k8sClient.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
	}, timeout, interval).Should(Succeed())
}

func SetupWithStatus(ctx context.Context, k8sClient client.Client, obj client.Object, statusUpdater func(client.Object)) {
	Setup(ctx, k8sClient, obj)

	statusUpdater(obj)

	Expect(k8sClient.Status().Update(ctx, obj)).Should(Succeed())

	Eventually(func() bool {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
		if err != nil {
			return false
		}
		switch o := obj.(type) {
		case *acp.ContactChannel:
			return o.Status.Ready
		case *acp.MCPServer:
			return o.Status.Ready
		case *acp.Tool:
			return o.Status.Ready
		}
		return true
	}, timeout, interval).Should(BeTrue())
}

func Teardown(ctx context.Context, k8sClient client.Client, obj client.Object) {
	Expect(k8sClient.Delete(ctx, obj)).Should(Succeed())

	Eventually(func() error {
		return k8sClient.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
	}, timeout, interval).ShouldNot(Succeed())
}

func setupTestAddTool(ctx context.Context, k8sClient client.Client, tool *acp.Tool, ready bool) {
	if ready {
		SetupWithStatus(ctx, k8sClient, tool, func(obj client.Object) {
			t := obj.(*acp.Tool)
			t.Status = acp.ToolStatus{Ready: true, StatusDetail: "Ready"}
		})
	} else {
		Setup(ctx, k8sClient, tool)
	}
}

func setupTestApprovalResources(ctx context.Context, k8sClient client.Client, baseName string) (*corev1.Secret, *acp.ContactChannel, *acp.MCPServer) {
	secret := createTestSecret(baseName+"-secret", "test-api-key")
	Setup(ctx, k8sClient, secret)

	contactChannel := createTestContactChannel(baseName+"-cc", acp.ContactChannelTypeSlack)
	SetupWithStatus(ctx, k8sClient, contactChannel, func(obj client.Object) {
		cc := obj.(*acp.ContactChannel)
		cc.Status = acp.ContactChannelStatus{Ready: true, StatusDetail: "Ready"}
	})

	mcpServer := createTestMCPServer(baseName+"-mcp", &contactChannel.Name)
	SetupWithStatus(ctx, k8sClient, mcpServer, func(obj client.Object) {
		mcp := obj.(*acp.MCPServer)
		mcp.Status = acp.MCPServerStatus{Status: "Ready", StatusDetail: "Ready"}
	})

	return secret, contactChannel, mcpServer
}

type MockMCPManager struct {
	CallToolFunc func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (string, error)
}

func (m *MockMCPManager) ConnectServer(ctx context.Context, mcpServer *acp.MCPServer) error {
	return nil
}
func (m *MockMCPManager) DisconnectServer(ctx context.Context, serverName string) error {
	return nil
}
func (m *MockMCPManager) GetTools(ctx context.Context, serverName string) ([]acp.MCPTool, error) {
	return nil, nil
}
func (m *MockMCPManager) GetToolsForAgent(ctx context.Context, agent *acp.Agent) ([]acp.MCPTool, error) {
	return nil, nil
}
func (m *MockMCPManager) CallTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (string, error) {
	if m.CallToolFunc != nil {
		return m.CallToolFunc(ctx, serverName, toolName, args)
	}
	return fmt.Sprintf("mock result for %s on %s", toolName, serverName), nil
}
func (m *MockMCPManager) FindServerForTool(ctx context.Context, toolName string) (string, error) {
	return "", nil
}
func (m *MockMCPManager) Close() {
}

func reconciler(k8sClient client.Client, mcpMgr mcpmanager.MCPManagerInterface, hlFactory humanlayer.HumanLayerClientFactory) *TaskRunToolCallReconciler {
	tracer := otel.GetTracerProvider().Tracer("test-taskruntoolcall-reconciler")
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("test-taskruntoolcall-reconciler-noop")
	}

	recorder := record.NewFakeRecorder(10)

	return &TaskRunToolCallReconciler{
		Client:          k8sClient,
		Scheme:          k8sClient.Scheme(),
		recorder:        recorder,
		MCPManager:      mcpMgr,
		HLClientFactory: hlFactory,
		Tracer:          tracer,
	}
}

func SetupTestApprovalConfig(shouldApprove bool, responseComment string, expectedCallID *string) humanlayer.HumanLayerClientFactory {
	mockHLClient := &humanlayer.MockHumanLayerClientWrapper{
		RequestApprovalFunc: func(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
			callID := "test-call-id-" + time.Now().Format("150405")
			if expectedCallID != nil {
				*expectedCallID = callID
			}
			output := humanlayerapi.NewFunctionCallOutputWithDefaults()
			output.SetCallId(callID)
			return output, 200, nil
		},
		GetFunctionCallStatusFunc: func(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
			output := humanlayerapi.NewFunctionCallOutputWithDefaults()
			status := humanlayerapi.NewFunctionCallStatusWithDefaults()
			now := time.Now()
			status.SetRequestedAt(now.Add(-time.Minute))
			status.SetRespondedAt(now)
			status.SetApproved(shouldApprove)
			status.SetComment(responseComment)
			output.SetStatus(*status)
			return output, 200, nil
		},
	}

	mockFactory := &humanlayer.MockHumanLayerClientFactory{
		NewHumanLayerClientFunc: func() humanlayer.HumanLayerClientWrapper {
			return mockHLClient
		},
	}
	return mockFactory
}
