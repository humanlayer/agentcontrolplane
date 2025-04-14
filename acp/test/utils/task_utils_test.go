package utils

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

type TestTask struct {
	name        string
	agentName   string
	userMessage string
	task        *acp.Task
	k8sClient   client.Client
}

func NewTestTask(k8sClient client.Client, name, agentName, userMessage string) *TestTask {
	return &TestTask{
		name:        name,
		agentName:   agentName,
		userMessage: userMessage,
		k8sClient:   k8sClient,
	}
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

	err := t.k8sClient.Create(ctx, task)
	Expect(err).NotTo(HaveOccurred())

	Expect(t.k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, task)).To(Succeed())
	t.task = task
	return task
}

func (t *TestTask) SetupWithStatus(ctx context.Context, status acp.TaskStatus) *acp.Task {
	task := t.Setup(ctx)
	task.Status = status
	Expect(t.k8sClient.Status().Update(ctx, task)).To(Succeed())
	Expect(t.k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, task)).To(Succeed())
	t.task = task
	return task
}

func (t *TestTask) Teardown(ctx context.Context) {
	By("deleting the task")
	Expect(t.k8sClient.Delete(ctx, t.task)).To(Succeed())
}

type TestToolCall struct {
	name            string
	taskName        string
	taskRunToolCall *acp.ToolCall
	k8sClient       client.Client
}

func NewTestToolCall(k8sClient client.Client, name, taskName string) *TestToolCall {
	return &TestToolCall{
		name:      name,
		taskName:  taskName,
		k8sClient: k8sClient,
	}
}

func (t *TestToolCall) Setup(ctx context.Context) *acp.ToolCall {
	By("creating the toolcall")
	taskRunToolCall := &acp.ToolCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
			Labels: map[string]string{
				"acp.humanlayer.dev/task":            t.taskName,
				"acp.humanlayer.dev/toolcallrequest": "test123",
			},
		},
		Spec: acp.ToolCallSpec{
			TaskRef: acp.LocalObjectReference{
				Name: t.taskName,
			},
			ToolRef: acp.LocalObjectReference{
				Name: "test-tool",
			},
			Arguments: `{"url": "https://api.example.com/data"}`,
		},
	}
	err := t.k8sClient.Create(ctx, taskRunToolCall)
	Expect(err).NotTo(HaveOccurred())
	Expect(t.k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, taskRunToolCall)).To(Succeed())
	t.taskRunToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestToolCall) SetupWithStatus(ctx context.Context, status acp.ToolCallStatus) *acp.ToolCall {
	taskRunToolCall := t.Setup(ctx)
	taskRunToolCall.Status = status
	Expect(t.k8sClient.Status().Update(ctx, taskRunToolCall)).To(Succeed())
	t.taskRunToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestToolCall) Teardown(ctx context.Context) {
	By("deleting the toolcall")
	Expect(t.k8sClient.Delete(ctx, t.taskRunToolCall)).To(Succeed())
}
