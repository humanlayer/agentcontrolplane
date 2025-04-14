package task

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
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
	llm  *acp.LLM
}

func (t *TestLLM) Setup(ctx context.Context) *acp.LLM {
	By("creating the llm")
	llm := &acp.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: acp.LLMSpec{
			Provider: "openai",
			APIKeyFrom: &acp.APIKeySource{
				SecretKeyRef: acp.SecretKeyRef{
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

func (t *TestLLM) SetupWithStatus(ctx context.Context, status acp.LLMStatus) *acp.LLM {
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
	mcpServers []acp.LocalObjectReference
	agent      *acp.Agent
}

func (t *TestAgent) Setup(ctx context.Context) *acp.Agent {
	By("creating the agent")
	agent := &acp.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
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
	llmName:    testLLM.name,
	system:     "you are a testing assistant",
	mcpServers: []acp.LocalObjectReference{},
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

var (
	testTaskObj     = utils.NewTestTask(k8sClient, "test-task", "test-agent", "what is the capital of the moon?")
	testToolCallObj = utils.NewTestToolCall(k8sClient, "test-toolcall", testTaskObj.name)
)
