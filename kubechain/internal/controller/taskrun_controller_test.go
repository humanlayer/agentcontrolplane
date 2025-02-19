package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"go.opentelemetry.io/otel/metric/noop"
)

var _ = Describe("TaskRun Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-taskrun"
		const taskName = "test-task"
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
					Name:      taskName,
					Namespace: "default",
				},
			}
			_ = k8sClient.Delete(ctx, task)
			time.Sleep(100 * time.Millisecond)

			taskRun := &kubechainv1alpha1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
			}
			_ = k8sClient.Delete(ctx, taskRun)
			time.Sleep(100 * time.Millisecond)

			unreadyTask := &kubechainv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unready-task",
					Namespace: "default",
				},
			}
			_ = k8sClient.Delete(ctx, unreadyTask)
			time.Sleep(100 * time.Millisecond)

			// Create test resources
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
			agent.Status.Status = "Ready for testing"
			Expect(k8sClient.Status().Update(ctx, agent)).To(Succeed())

			// Create a test Task
			By("Creating a test Task")
			task = &kubechainv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskName,
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

			// Mark Task as ready
			task.Status.Ready = true
			task.Status.Status = "Agent validated successfully"
			Expect(k8sClient.Status().Update(ctx, task)).To(Succeed())
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

			By("Cleanup the test TaskRun")
			taskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, taskRun)
			if err == nil {
				Expect(k8sClient.Delete(ctx, taskRun)).To(Succeed())
			}

			By("Cleanup the unready task if it exists")
			unreadyTask := &kubechainv1alpha1.Task{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "unready-task", Namespace: "default"}, unreadyTask)
			if err == nil {
				Expect(k8sClient.Delete(ctx, unreadyTask)).To(Succeed())
			}
		})

		It("should successfully execute a task", func() {
			By("creating the taskrun")
			taskRun := &kubechainv1alpha1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("reconciling the taskrun")
			reconciler := &TaskRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Initialize metrics with no-op implementations for testing
			meter := noop.NewMeterProvider().Meter("test")
			var err error
			reconciler.reconcileCounter, err = meter.Int64Counter("taskrun_reconcile_count")
			Expect(err).NotTo(HaveOccurred())
			reconciler.phaseCounter, err = meter.Int64Counter("taskrun_phase_transition")
			Expect(err).NotTo(HaveOccurred())
			reconciler.errorCounter, err = meter.Int64Counter("taskrun_error_count")
			Expect(err).NotTo(HaveOccurred())
			reconciler.reconcileDuration, err = meter.Float64Histogram("taskrun_reconcile_duration")
			Expect(err).NotTo(HaveOccurred())

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the taskrun status")
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseRunning))
		})

		It("should fail when task doesn't exist", func() {
			By("creating the taskrun with non-existent task")
			taskRun := &kubechainv1alpha1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: "nonexistent-task",
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("reconciling the taskrun")
			reconciler := &TaskRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Initialize metrics with no-op implementations for testing
			meter := noop.NewMeterProvider().Meter("test")
			var err error
			reconciler.reconcileCounter, err = meter.Int64Counter("taskrun_reconcile_count")
			Expect(err).NotTo(HaveOccurred())
			reconciler.phaseCounter, err = meter.Int64Counter("taskrun_phase_transition")
			Expect(err).NotTo(HaveOccurred())
			reconciler.errorCounter, err = meter.Int64Counter("taskrun_error_count")
			Expect(err).NotTo(HaveOccurred())
			reconciler.reconcileDuration, err = meter.Float64Histogram("taskrun_reconcile_duration")
			Expect(err).NotTo(HaveOccurred())

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the taskrun status")
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseFailed))
			Expect(updatedTaskRun.Status.Error).To(ContainSubstring("failed to get Task"))
		})

		It("should stay pending when task exists but is not ready", func() {
			By("creating a task that is not ready")
			unreadyTask := &kubechainv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unready-task",
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskSpec{
					AgentRef: kubechainv1alpha1.LocalObjectReference{
						Name: agentName,
					},
					Message: "Test input",
				},
			}
			Expect(k8sClient.Create(ctx, unreadyTask)).To(Succeed())

			By("creating the taskrun referencing the unready task")
			taskRun := &kubechainv1alpha1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: "unready-task",
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("reconciling the taskrun")
			reconciler := &TaskRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Initialize metrics with no-op implementations for testing
			meter := noop.NewMeterProvider().Meter("test")
			var err error
			reconciler.reconcileCounter, err = meter.Int64Counter("taskrun_reconcile_count")
			Expect(err).NotTo(HaveOccurred())
			reconciler.phaseCounter, err = meter.Int64Counter("taskrun_phase_transition")
			Expect(err).NotTo(HaveOccurred())
			reconciler.errorCounter, err = meter.Int64Counter("taskrun_error_count")
			Expect(err).NotTo(HaveOccurred())
			reconciler.reconcileDuration, err = meter.Float64Histogram("taskrun_reconcile_duration")
			Expect(err).NotTo(HaveOccurred())

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("checking the taskrun phase")
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhasePending))
		})
	})
})
