package utils

import (
	"context"

	"github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestAgent struct {
	Name         string
	LLMName      string
	SystemPrompt string
	MCPServers   []v1alpha1.LocalObjectReference
	Agent        *v1alpha1.Agent
	k8sClient    client.Client
}

func (t *TestAgent) Setup(ctx context.Context, k8sClient client.Client) *v1alpha1.Agent {
	t.k8sClient = k8sClient
	ginkgo.By("creating the agent")
	agent := &v1alpha1.Agent{
		ObjectMeta: v1.ObjectMeta{
			Name: t.Name,

			Namespace: "default",
		},
		Spec: v1alpha1.AgentSpec{
			LLMRef: v1alpha1.LocalObjectReference{
				Name: t.LLMName,
			},
			System:     t.SystemPrompt,
			MCPServers: t.MCPServers,
		},
	}
	err := k8sClient.Create(ctx, agent)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.Name, Namespace: "default"}, agent)).To(gomega.Succeed())
	t.Agent = agent
	return agent
}

func (t *TestAgent) SetupWithStatus(
	ctx context.Context,
	k8sClient client.Client,
	status v1alpha1.AgentStatus,
) *v1alpha1.Agent {
	agent := t.Setup(ctx, k8sClient)
	agent.Status = status
	gomega.Expect(t.k8sClient.Status().Update(ctx, agent)).To(gomega.Succeed())
	t.Agent = agent
	return agent
}

func (t *TestAgent) Teardown(ctx context.Context) {
	if t.k8sClient == nil {
		return
	}
	ginkgo.By("deleting the agent")
	gomega.Expect(t.k8sClient.Delete(ctx, t.Agent)).To(gomega.Succeed())
}
