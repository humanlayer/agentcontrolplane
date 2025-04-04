package taskrun

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// todo this file should probably live in a shared package, but for now...
type TestSecret struct {
	name   string
	secret *corev1.Secret
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
	err := k8sClient.Create(ctx, secret)
	Expect(err).NotTo(HaveOccurred())
	t.secret = secret
	return secret
}

func (t *TestSecret) Teardown(ctx context.Context) {
	By("deleting the secret")
	Expect(k8sClient.Delete(ctx, t.secret)).To(Succeed())
}

var testSecret = &TestSecret{
	name: "test-secret",
}

type TestLLM struct {
	name string
	llm  *kubechain.LLM
}

func (t *TestLLM) Setup(ctx context.Context) *kubechain.LLM {
	By("creating the llm")
	llm := &kubechain.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechain.LLMSpec{
			Provider: "openai",
			APIKeyFrom: kubechain.APIKeySource{
				SecretKeyRef: kubechain.SecretKeyRef{
					Name: testSecret.name,
					Key:  "api-key",
				},
			},
		},
	}
	err := k8sClient.Create(ctx, llm)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, llm)).To(Succeed())
	t.llm = llm
	return llm
}

func (t *TestLLM) SetupWithStatus(ctx context.Context, status kubechain.LLMStatus) *kubechain.LLM {
	llm := t.Setup(ctx)
	llm.Status = status
	Expect(k8sClient.Status().Update(ctx, llm)).To(Succeed())
	t.llm = llm
	return llm
}

func (t *TestLLM) Teardown(ctx context.Context) {
	By("deleting the llm")
	Expect(k8sClient.Delete(ctx, t.llm)).To(Succeed())
}

var testLLM = &TestLLM{
	name: "test-llm",
}

type TestAgent struct {
	name       string
	llmName    string
	system     string
	mcpServers []kubechain.LocalObjectReference
	agent      *kubechain.Agent
}

func (t *TestAgent) Setup(ctx context.Context) *kubechain.Agent {
	By("creating the agent")
	agent := &kubechain.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechain.AgentSpec{
			LLMRef: kubechain.LocalObjectReference{
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

func (t *TestAgent) SetupWithStatus(ctx context.Context, status kubechain.AgentStatus) *kubechain.Agent {
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
	llmName:    testLLM.name,
	system:     "you are a testing assistant",
	mcpServers: []kubechain.LocalObjectReference{},
}

type TestTask struct {
	name      string
	agentName string
	message   string
	task      *kubechain.Task
}

func (t *TestTask) Setup(ctx context.Context) *kubechain.Task {
	By("creating the task")
	task := &kubechain.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechain.TaskSpec{
			AgentRef: kubechain.LocalObjectReference{
				Name: t.agentName,
			},
			Message: t.message,
		},
	}
	err := k8sClient.Create(ctx, task)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, task)).To(Succeed())
	t.task = task
	return task
}

func (t *TestTask) SetupWithStatus(ctx context.Context, status kubechain.TaskStatus) *kubechain.Task {
	task := t.Setup(ctx)
	task.Status = status
	Expect(k8sClient.Status().Update(ctx, task)).To(Succeed())
	t.task = task
	return task
}

func (t *TestTask) Teardown(ctx context.Context) {
	By("deleting the task")
	Expect(k8sClient.Delete(ctx, t.task)).To(Succeed())
}

type TestTaskRun struct {
	name     string
	taskName string
	taskRun  *kubechain.TaskRun
}

func (t *TestTaskRun) Setup(ctx context.Context) *kubechain.TaskRun {
	By("creating the taskrun")
	taskRun := &kubechain.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechain.TaskRunSpec{
			TaskRef: &kubechain.LocalObjectReference{
				Name: t.taskName,
			},
		},
	}
	err := k8sClient.Create(ctx, taskRun)
	Expect(err).NotTo(HaveOccurred())

	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, taskRun)).To(Succeed())
	t.taskRun = taskRun
	return taskRun
}

func (t *TestTaskRun) SetupWithStatus(ctx context.Context, status kubechain.TaskRunStatus) *kubechain.TaskRun {
	taskRun := t.Setup(ctx)
	taskRun.Status = status
	Expect(k8sClient.Status().Update(ctx, taskRun)).To(Succeed())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, taskRun)).To(Succeed())
	t.taskRun = taskRun
	return taskRun
}

func (t *TestTaskRun) Teardown(ctx context.Context) {
	By("deleting the taskrun")
	Expect(k8sClient.Delete(ctx, t.taskRun)).To(Succeed())
}

var testTask = &TestTask{
	name:      "test-task",
	agentName: "test-agent",
	message:   "what is the capital of the moon?",
}

var testTaskRun = &TestTaskRun{
	name:     "test-taskrun",
	taskName: testTask.name,
}

type TestTaskRunToolCall struct {
	name            string
	taskRunToolCall *kubechain.TaskRunToolCall
}

func (t *TestTaskRunToolCall) Setup(ctx context.Context) *kubechain.TaskRunToolCall {
	By("creating the taskrun toolcall")
	taskRunToolCall := &kubechain.TaskRunToolCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
			Labels: map[string]string{
				"kubechain.humanlayer.dev/taskruntoolcall": testTaskRun.name,
				"kubechain.humanlayer.dev/toolcallrequest": "test123",
			},
		},
		Spec: kubechain.TaskRunToolCallSpec{
			TaskRunRef: &kubechain.LocalObjectReference{
				Name: testTaskRun.name,
			},
			ToolRef: &kubechain.LocalObjectReference{
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

func (t *TestTaskRunToolCall) SetupWithStatus(ctx context.Context, status kubechain.TaskRunToolCallStatus) *kubechain.TaskRunToolCall {
	taskRunToolCall := t.Setup(ctx)
	taskRunToolCall.Status = status
	Expect(k8sClient.Status().Update(ctx, taskRunToolCall)).To(Succeed())
	t.taskRunToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestTaskRunToolCall) Teardown(ctx context.Context) {
	By("deleting the taskrun toolcall")
	Expect(k8sClient.Delete(ctx, t.taskRunToolCall)).To(Succeed())
}

var testTaskRunToolCall = &TestTaskRunToolCall{
	name: "test-taskrun-toolcall",
}

// nolint:golint,unparam
func setupSuiteObjects(ctx context.Context) (secret *corev1.Secret, llm *kubechain.LLM, agent *kubechain.Agent, task *kubechain.Task, teardown func()) {
	secret = testSecret.Setup(ctx)
	llm = testLLM.SetupWithStatus(ctx, kubechain.LLMStatus{
		Status: "Ready",
		Ready:  true,
	})
	agent = testAgent.SetupWithStatus(ctx, kubechain.AgentStatus{
		Status: "Ready",
		Ready:  true,
	})
	task = testTask.SetupWithStatus(ctx, kubechain.TaskStatus{
		Status: "Ready",
		Ready:  true,
	})
	teardown = func() {
		testSecret.Teardown(ctx)
		testLLM.Teardown(ctx)
		testAgent.Teardown(ctx)
		testTask.Teardown(ctx)
	}
	return secret, llm, agent, task, teardown
}

func reconciler() (*TaskRunReconciler, *record.FakeRecorder) {
	By("creating the reconciler")
	recorder := record.NewFakeRecorder(10)
	reconciler := &TaskRunReconciler{
		Client:   k8sClient,
		Scheme:   k8sClient.Scheme(),
		recorder: recorder,
	}
	return reconciler, recorder
}
