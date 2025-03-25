package task

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/test/utils"
)

var _ = Describe("Task Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-task"
		const agentName = "test-agent"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			// Clean up any existing resources first
			By("Cleaning up any existing resources")
			agent := &kubechainv1alpha1.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      agentName,
					Namespace: "default",
				},
			}
			_ = k8sClient.Delete(ctx, agent)
			time.Sleep(100 * time.Millisecond)

			task := &kubechainv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
			}
			_ = k8sClient.Delete(ctx, task)
			time.Sleep(100 * time.Millisecond)

			// Create test Agent
			By("Creating a test Agent")
			agent = &kubechainv1alpha1.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      agentName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.AgentSpec{
					LLMRef: kubechainv1alpha1.LocalObjectReference{
						Name: "test-llm",
					},
					System: "Test agent",
				},
			}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())

			// Mark Agent as ready
			agent.Status.Ready = true
			agent.Status.Status = "Ready"
			agent.Status.StatusDetail = "Ready for testing"
			Expect(k8sClient.Status().Update(ctx, agent)).To(Succeed())
		})

		AfterEach(func() {
			// Cleanup test resources
			By("Cleanup the test Agent")
			agent := &kubechainv1alpha1.Agent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: agentName, Namespace: "default"}, agent)
			if err == nil {
				Expect(k8sClient.Delete(ctx, agent)).To(Succeed())
			}

			By("Cleanup the test Task")
			task := &kubechainv1alpha1.Task{}
			err = k8sClient.Get(ctx, typeNamespacedName, task)
			if err == nil {
				Expect(k8sClient.Delete(ctx, task)).To(Succeed())
			}
		})

		It("should successfully validate a task with valid agent", func() {
			By("creating the task with valid agent reference")
			task := &kubechainv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskSpec{
					AgentRef: kubechainv1alpha1.LocalObjectReference{
						Name: agentName,
					},
					Message: "Test input",
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			By("reconciling the task")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the task status")
			updatedTask := &kubechainv1alpha1.Task{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTask)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTask.Status.Ready).To(BeTrue())
			Expect(updatedTask.Status.Status).To(Equal(kubechainv1alpha1.TaskStatusReady))
			Expect(updatedTask.Status.StatusDetail).To(Equal("Task Run Created"))

			By("checking that TaskRun creation event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("TaskRunCreated")

			By("checking that validation success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})

		It("should fail validation with non-existent agent", func() {
			By("creating the task with invalid agent reference")
			task := &kubechainv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskSpec{
					AgentRef: kubechainv1alpha1.LocalObjectReference{
						Name: "nonexistent-agent",
					},
					Message: "Test input",
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			By("reconciling the task")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`"nonexistent-agent" not found`))

			By("checking the task status")
			updatedTask := &kubechainv1alpha1.Task{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTask)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTask.Status.Ready).To(BeFalse())
			Expect(updatedTask.Status.Status).To(Equal(kubechainv1alpha1.TaskStatusError))
			Expect(updatedTask.Status.StatusDetail).To(ContainSubstring(`"nonexistent-agent" not found`))

			By("checking that a failure event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationFailed")
		})
	})
})
