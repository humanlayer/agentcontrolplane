package task

import (
	"context"

	"github.com/humanlayer/agentcontrolplane/acp/test/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
	. "github.com/onsi/ginkgo/v2"
)

var testSecret = &utils.TestSecret{
	Name: "test-secret",
}

var testLLM = &utils.TestLLM{
	Name:       "test-llm",
	SecretName: testSecret.Name,
}

var testAgent = &utils.TestAgent{
	Name:         "test-agent",
	LLM:          testLLM.Name,
	SystemPrompt: "you are a testing assistant",
	MCPServers:   []string{},
}

var testTask = &utils.TestTask{
	Name:        "test-task",
	AgentName:   "test-agent",
	UserMessage: "what is the capital of the moon?",
}

var testToolCall = &utils.TestToolCall{
	Name:     "test-toolcall",
	TaskName: testTask.Name,
	ToolRef:  "test-tool",
	ToolType: acp.ToolTypeMCP,
}

var testToolCallTwo = &utils.TestToolCall{
	Name:     "test-toolcall-two",
	TaskName: testTask.Name,
	ToolRef:  "test-tool",
	ToolType: acp.ToolTypeMCP,
}

// nolint:golint,unparam
func setupSuiteObjects(ctx context.Context) (secret *corev1.Secret, llm *acp.LLM, agent *acp.Agent, teardown func()) {
	secret = testSecret.Setup(ctx, k8sClient)
	llm = testLLM.SetupWithStatus(ctx, k8sClient, acp.LLMStatus{
		Status: "Ready",
		Ready:  true,
	})
	agent = testAgent.SetupWithStatus(ctx, k8sClient, acp.AgentStatus{
		Status: "Ready",
		Ready:  true,
	})
	teardown = func() {
		testSecret.Teardown(ctx)
		testLLM.Teardown(ctx)
		testAgent.Teardown(ctx)
	}
	return secret, llm, agent, teardown
}

// reconciler is a utility function to create a new TaskReconciler
// nolint:unused
func reconciler() (*TaskReconciler, *record.FakeRecorder) {
	By("creating the reconciler")
	recorder := record.NewFakeRecorder(10)
	reconciler := &TaskReconciler{
		Client:      k8sClient,
		Scheme:      k8sClient.Scheme(),
		recorder:    recorder,
		MCPManager:  &mcpmanager.MCPServerManager{},
		toolAdapter: &defaultToolAdapter{},
		Tracer:      noop.NewTracerProvider().Tracer("test"),
	}
	return reconciler, recorder
}
