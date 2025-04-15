package utils

import (
	"context"

	. "github.com/onsi/gomega" //nolint:golint,revive
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestUtils struct {
	K8sClient client.Client
}

type TestScopedAgent struct {
	Name                 string
	SystemPrompt         string
	LLM                  string
	HumanContactChannels []string
	MCPServers           []string
	SubAgents            []string
	Description          string
	client               client.Client
}

func (t *TestScopedAgent) Setup(k8sClient client.Client) {
	t.client = k8sClient
	// Create a test Agent
	agent := &acp.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: acp.AgentSpec{
			LLMRef: acp.LocalObjectReference{
				Name: t.LLM,
			},
			System:      t.SystemPrompt,
			Description: t.Description,
			HumanContactChannels: func() []acp.LocalObjectReference {
				refs := make([]acp.LocalObjectReference, len(t.HumanContactChannels))
				for i, channel := range t.HumanContactChannels {
					refs[i] = acp.LocalObjectReference{Name: channel}
				}
				return refs
			}(),
			MCPServers: func() []acp.LocalObjectReference {
				refs := make([]acp.LocalObjectReference, len(t.MCPServers))
				for i, server := range t.MCPServers {
					refs[i] = acp.LocalObjectReference{Name: server}
				}
				return refs
			}(),
			SubAgents: func() []acp.LocalObjectReference {
				refs := make([]acp.LocalObjectReference, len(t.SubAgents))
				for i, agent := range t.SubAgents {
					refs[i] = acp.LocalObjectReference{Name: agent}
				}
				return refs
			}(),
		},
	}
	Expect(t.client.Create(context.Background(), agent)).To(Succeed())

	// Mark Agent as ready
	agent.Status.Ready = true
	agent.Status.Status = "Ready"
	agent.Status.StatusDetail = "Ready for testing"
	agent.Status.ValidHumanContactChannels = func() []acp.ResolvedContactChannel {
		channels := make([]acp.ResolvedContactChannel, len(t.HumanContactChannels))
		for i, channel := range t.HumanContactChannels {
			channels[i] = acp.ResolvedContactChannel{
				Name: channel,
				Type: "email", // Default type for testing
			}
		}
		return channels
	}()
	Expect(t.client.Status().Update(context.Background(), agent)).To(Succeed())
}

func (t *TestScopedAgent) Teardown() {
	agent := &acp.Agent{}
	err := t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, agent)
	if err == nil {
		Expect(t.client.Delete(context.Background(), agent)).To(Succeed())
	}
}
