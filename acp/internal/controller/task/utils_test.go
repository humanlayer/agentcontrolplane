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

type TestAgent struct {
	name       string
	llmName    string
	system     string
	mcpServers []acp.LocalObjectReference
	agent      *acp.Agent
}

func (t *TestAgent) Setup(ctx context.Context) *acp.Agent {
	By("creating the agent")
	agent := &acp.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name: t.name,

			Namespace: "default",
		},
		Spec: acp.AgentSpec{
			LLMRef: acp.LocalObjectReference{
				Name: t.llmName,
			},
			System:     t.system,
			MCPServers: t.mcpServers,
		},
	}
	err := k8sClient.Create(ctx, agent)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, agent)).To(Succeed())
	t.agent = agent
	return agent
}

func (t *TestAgent) SetupWithStatus(ctx context.Context, status acp.AgentStatus) *acp.Agent {
	agent := t.Setup(ctx)
	agent.Status = status
	Expect(k8sClient.Status().Update(ctx, agent)).To(Succeed())
	t.agent = agent
	return agent
}

func (t *TestAgent) Teardown(ctx context.Context) {
	By("deleting the agent")
	Expect(k8sClient.Delete(ctx, t.agent)).To(Succeed())
}

var testAgent = &TestAgent{
	name:       "test-agent",
	llmName:    testLLM.Name,
	system:     "you are a testing assistant",
	mcpServers: []acp.LocalObjectReference{},
}

type TestTask struct {
	name        string
	agentName   string
	userMessage string
	task        *acp.Task
}

func (t *TestTask) Setup(ctx context.Context) *acp.Task {
	By("creating the task")
	task := &acp.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: acp.TaskSpec{},
	}
	if t.agentName != "" {
		task.Spec.AgentRef = acp.LocalObjectReference{
			Name: t.agentName,
		}
	}
	if t.userMessage != "" {
		task.Spec.UserMessage = t.userMessage
	}

	err := k8sClient.Create(ctx, task)
	Expect(err).NotTo(HaveOccurred())

	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, task)).To(Succeed())
	t.task = task
	return task
}

func (t *TestTask) SetupWithStatus(ctx context.Context, status acp.TaskStatus) *acp.Task {
	task := t.Setup(ctx)
	task.Status = status
	Expect(k8sClient.Status().Update(ctx, task)).To(Succeed())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, task)).To(Succeed())
	t.task = task
	return task
}

func (t *TestTask) Teardown(ctx context.Context) {
	By("deleting the task")
	Expect(k8sClient.Delete(ctx, t.task)).To(Succeed())
}

var testTask = &TestTask{
	name:        "test-task",
	agentName:   "test-agent",
	userMessage: "what is the capital of the moon?",
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
				"acp.humanlayer.dev/task":            testTask.name,
				"acp.humanlayer.dev/toolcallrequest": "test123",
			},
		},
		Spec: acp.ToolCallSpec{
			TaskRef: acp.LocalObjectReference{
				Name: testTask.name,
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
	secret = testSecret.Setup(ctx)
	llm = testLLM.SetupWithStatus(ctx, acp.LLMStatus{
		Status: "Ready",
		Ready:  true,
	})
	agent = testAgent.SetupWithStatus(ctx, acp.AgentStatus{
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
