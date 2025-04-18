package utils

import (
	"context"

	"github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestTask struct {
	Name          string
	AgentName     string
	UserMessage   string
	ContextWindow []v1alpha1.Message
	Task          *v1alpha1.Task
	k8sClient     client.Client
}

func (t *TestTask) Setup(ctx context.Context, k8sClient client.Client) *v1alpha1.Task {
	t.k8sClient = k8sClient
	By("creating the task")
	task := &v1alpha1.Task{
		ObjectMeta: v1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: v1alpha1.TaskSpec{},
	}
	if t.AgentName != "" {
		task.Spec.AgentRef = v1alpha1.LocalObjectReference{
			Name: t.AgentName,
		}
	}
	if t.UserMessage != "" {
		task.Spec.UserMessage = t.UserMessage
	}
	if len(t.ContextWindow) > 0 {
		task.Spec.ContextWindow = t.ContextWindow
	}

	err := k8sClient.Create(ctx, task)
	Expect(err).NotTo(HaveOccurred())

	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.Name, Namespace: "default"}, task)).To(Succeed())
	t.Task = task
	return task
}

func (t *TestTask) SetupWithStatus(
	ctx context.Context,
	k8sClient client.Client,
	status v1alpha1.TaskStatus,
) *v1alpha1.Task {
	task := t.Setup(ctx, k8sClient)
	task.Status = status
	Expect(t.k8sClient.Status().Update(ctx, task)).To(Succeed())
	Expect(t.k8sClient.Get(ctx, types.NamespacedName{Name: t.Name, Namespace: "default"}, task)).To(Succeed())
	t.Task = task
	return task
}

func (t *TestTask) Teardown(ctx context.Context) {
	if t.k8sClient == nil {
		return
	}
	By("deleting the task")
	Expect(t.k8sClient.Delete(ctx, t.Task)).To(Succeed())
}
