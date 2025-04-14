package utils

import (
	"context"
	"github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestLLM struct {
	Name       string
	SecretName string
	LLM        *v1alpha1.LLM
	k8sClient  client.Client
}

func (t *TestLLM) Setup(ctx context.Context, k8sClient client.Client) *v1alpha1.LLM {
	t.k8sClient = k8sClient
	By("creating the llm")
	llm := &v1alpha1.LLM{
		ObjectMeta: v1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: v1alpha1.LLMSpec{
			Provider: "openai",
			APIKeyFrom: &v1alpha1.APIKeySource{
				SecretKeyRef: v1alpha1.SecretKeyRef{
					Name: t.SecretName,
					Key:  "api-key",
				},
			},
		},
	}
	err := k8sClient.Create(ctx, llm)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.Name, Namespace: "default"}, llm)).To(Succeed())
	t.LLM = llm
	return llm
}

func (t *TestLLM) SetupWithStatus(ctx context.Context, k8sClient client.Client, status v1alpha1.LLMStatus) *v1alpha1.LLM {
	llm := t.Setup(ctx, k8sClient)
	llm.Status = status
	Expect(k8sClient.Status().Update(ctx, llm)).To(Succeed())
	t.LLM = llm
	return llm
}

func (t *TestLLM) Teardown(ctx context.Context) {
	if t.k8sClient == nil {
		return
	}
	By("deleting the llm")
	Expect(t.k8sClient.Delete(ctx, t.LLM)).To(Succeed())
}
