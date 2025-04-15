package utils

import (
	"context"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type TestToolCall struct {
	Name     string
	TaskName string
	ToolRef  string
	ToolType acp.ToolType

	ToolCall  *acp.ToolCall
	k8sClient client.Client
}

func (t *TestToolCall) Setup(ctx context.Context, k8sClient client.Client) *acp.ToolCall {
	t.k8sClient = k8sClient

	By("creating the toolcall")
	toolCall := &acp.ToolCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
			Labels: map[string]string{
				"acp.humanlayer.dev/task":            t.TaskName,
				"acp.humanlayer.dev/toolcallrequest": "test123",
			},
		},
		Spec: acp.ToolCallSpec{
			TaskRef: acp.LocalObjectReference{
				Name: t.TaskName,
			},
			ToolRef: acp.LocalObjectReference{
				Name: t.ToolRef,
			},
			Arguments: `{"url": "https://api.example.com/data"}`,
			ToolType:  t.ToolType,
		},
	}
	err := t.k8sClient.Create(ctx, toolCall)
	Expect(err).NotTo(HaveOccurred())
	Expect(t.k8sClient.Get(ctx, types.NamespacedName{Name: t.Name, Namespace: "default"}, toolCall)).To(Succeed())
	t.ToolCall = toolCall
	return toolCall
}

func (t *TestToolCall) SetupWithStatus(ctx context.Context, k8sClient client.Client, status acp.ToolCallStatus) *acp.ToolCall {
	taskRunToolCall := t.Setup(ctx, k8sClient)
	taskRunToolCall.Status = status
	Expect(k8sClient.Status().Update(ctx, taskRunToolCall)).To(Succeed())
	t.ToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestToolCall) Teardown(ctx context.Context) {
	if t.k8sClient == nil {
		return
	}
	By("deleting the toolcall")
	Expect(t.k8sClient.Delete(ctx, t.ToolCall)).To(Succeed())
}
