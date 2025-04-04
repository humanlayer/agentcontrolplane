package taskrun

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
	. "github.com/humanlayer/smallchain/kubechain/test/utils"
)

var _ = Describe("TaskRun Controller", func() {
	Context("Initializing -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if the taskRun is missing a taskRef but has an agentRef and userMessage", func() {
			By("creating the llm and agent")
			testLLM.SetupWithStatus(ctx, kubechain.LLMStatus{
				Status: "Ready",
				Ready:  true,
			})
			defer testLLM.Teardown(ctx)

			agent := testAgent.SetupWithStatus(ctx, kubechain.AgentStatus{
				Status: "Ready",
				Ready:  true,
			})
			defer testAgent.Teardown(ctx)

			By("reconciling the taskrun")
			taskRun := &kubechain.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "direct-taskrun",
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					AgentRef: &kubechain.LocalObjectReference{
						Name: agent.Name,
					},
					UserMessage: "test message",
				},
			}
			err := k8sClient.Create(ctx, taskRun)
			Expect(err).NotTo(HaveOccurred())

			By("creating the reconciler")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRun.Name,
					Namespace: taskRun.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// First reconcile initializes the phase
			result, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRun.Name,
					Namespace: taskRun.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("ensuring the context window is set correctly")
			var updatedTaskRun kubechain.TaskRun
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      taskRun.Name,
				Namespace: taskRun.Namespace,
			}, &updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(2))
			Expect(updatedTaskRun.Status.ContextWindow[0].Role).To(Equal("system"))
			Expect(updatedTaskRun.Status.ContextWindow[1].Role).To(Equal("user"))
			Expect(updatedTaskRun.Status.ContextWindow[1].Content).To(Equal("test message"))
		})
	})
})
