package task

import (
	"context"

	"github.com/humanlayer/agentcontrolplane/acp/test/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	LLMName:      testLLM.Name,
	SystemPrompt: "you are a testing assistant",
	MCPServers:   []acp.LocalObjectReference{},
}

var testTask = &utils.TestTask{
	Name:        "test-task",
	AgentName:   "test-agent",
	UserMessage: "what is the capital of the moon?",
}

type TestToolCall struct {
	name            string
	taskRunToolCall *acp.ToolCall
}

func (t *TestToolCall) Setup(ctx context.Context) *acp.ToolCall {
	By("creating the toolcall")
	taskRunToolCall := &acp.ToolCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
			Labels: map[string]string{
				"acp.humanlayer.dev/task":            testTask.Name,
				"acp.humanlayer.dev/toolcallrequest": "test123",
			},
		},
		Spec: acp.ToolCallSpec{
			TaskRef: acp.LocalObjectReference{
				Name: testTask.Name,
			},
			ToolRef: acp.LocalObjectReference{
				Name: "test-tool",
			},
			Arguments: `{"url": "https://api.example.com/data"}`,
		},
	}
	err := k8sClient.Create(ctx, taskRunToolCall)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, taskRunToolCall)).To(Succeed())
	t.taskRunToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestToolCall) SetupWithStatus(ctx context.Context, status acp.ToolCallStatus) *acp.ToolCall {
	taskRunToolCall := t.Setup(ctx)
	taskRunToolCall.Status = status
	Expect(k8sClient.Status().Update(ctx, taskRunToolCall)).To(Succeed())
	t.taskRunToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestToolCall) Teardown(ctx context.Context) {
	By("deleting the toolcall")
	Expect(k8sClient.Delete(ctx, t.taskRunToolCall)).To(Succeed())
}

var testToolCall = &TestToolCall{
	name: "test-toolcall",
}

var testToolCallTwo = &TestToolCall{
	name: "test-toolcall-two",
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
		Client:     k8sClient,
		Scheme:     k8sClient.Scheme(),
		recorder:   recorder,
		MCPManager: &mcpmanager.MCPServerManager{},
		Tracer:     noop.NewTracerProvider().Tracer("test"),
	}
	return reconciler, recorder
}
