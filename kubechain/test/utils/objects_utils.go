package utils

import (
	"context"

	. "github.com/onsi/gomega" //nolint:golint,revive
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestScopedAgent struct {
	Name                 string
	SystemPrompt         string
	Tools                []string
	LLM                  string
	HumanContactChannels []string
	client               client.Client
}

func (t *TestScopedAgent) Setup(k8sClient client.Client) {
	t.client = k8sClient
	// Create a test Agent
	agent := &kubechain.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: kubechain.AgentSpec{
			LLMRef: kubechain.LocalObjectReference{
				Name: t.LLM,
			},
			System: t.SystemPrompt,
			Tools: func() []kubechain.LocalObjectReference {
				refs := make([]kubechain.LocalObjectReference, len(t.Tools))
				for i, tool := range t.Tools {
					refs[i] = kubechain.LocalObjectReference{Name: tool}
				}
				return refs
			}(),
			HumanContactChannels: func() []kubechain.LocalObjectReference {
				refs := make([]kubechain.LocalObjectReference, len(t.HumanContactChannels))
				for i, channel := range t.HumanContactChannels {
					refs[i] = kubechain.LocalObjectReference{Name: channel}
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
	agent.Status.ValidTools = func() []kubechain.ResolvedTool {
		tools := make([]kubechain.ResolvedTool, len(t.Tools))
		for i, tool := range t.Tools {
			tools[i] = kubechain.ResolvedTool{
				Kind: "Tool",
				Name: tool,
			}
		}
		return tools
	}()
	agent.Status.ValidHumanContactChannels = func() []kubechain.ResolvedContactChannel {
		channels := make([]kubechain.ResolvedContactChannel, len(t.HumanContactChannels))
		for i, channel := range t.HumanContactChannels {
			channels[i] = kubechain.ResolvedContactChannel{
				Name: channel,
				Type: "email", // Default type for testing
			}
		}
		return channels
	}()
	Expect(t.client.Status().Update(context.Background(), agent)).To(Succeed())

}

func (t *TestScopedAgent) Teardown() {
	agent := &kubechain.Agent{}
	err := t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, agent)
	if err == nil {
		Expect(t.client.Delete(context.Background(), agent)).To(Succeed())
	}
}
