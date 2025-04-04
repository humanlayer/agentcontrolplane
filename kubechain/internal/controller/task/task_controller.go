package task

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=tasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=taskruns,verbs=get;list;create;watch

// TaskReconciler reconciles a Task object
type TaskReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// validateAgent checks if the referenced agent exists and is ready
func (r *TaskReconciler) validateAgent(ctx context.Context, task *kubechainv1alpha1.Task) error {
	agent := &kubechainv1alpha1.Agent{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: task.Namespace,
		Name:      task.Spec.AgentRef.Name,
	}, agent)
	if err != nil {
		return fmt.Errorf("failed to get Agent %q: %w", task.Spec.AgentRef.Name, err)
	}

	if !agent.Status.Ready {
		return fmt.Errorf("agent %q is not ready", task.Spec.AgentRef.Name)
	}

	return nil
}

// Reconcile validates the task's agent reference and parameters
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var task kubechainv1alpha1.Task
	if err := r.Get(ctx, req.NamespacedName, &task); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", task.Name)

	// Create a copy for status update
	statusUpdate := task.DeepCopy()

	// Initialize status if not set
	if statusUpdate.Status.Status == "" {
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.StatusDetail = "Validating agent reference"
		r.recorder.Event(&task, corev1.EventTypeNormal, "Initializing", "Starting validation")
	}

	// Validate agent reference
	if err := r.validateAgent(ctx, &task); err != nil {
		logger.Error(err, "Agent validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = kubechainv1alpha1.TaskStatusError
		statusUpdate.Status.StatusDetail = err.Error()
		r.recorder.Event(&task, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return ctrl.Result{}, fmt.Errorf("failed to update task status: %v", err)
		}
		return ctrl.Result{}, err // requeue
	}

	// Check if we need to create a TaskRun
	taskRunList := &kubechainv1alpha1.TaskRunList{}
	if err := r.List(ctx, taskRunList, client.InNamespace(task.Namespace), client.MatchingLabels{
		"kubechain.humanlayer.dev/task": task.Name,
	}); err != nil {
		logger.Error(err, "Failed to list TaskRuns")
		return ctrl.Result{}, err
	}

	if len(taskRunList.Items) == 0 {
		// Create a new TaskRun
		taskRun := &kubechainv1alpha1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      task.Name + "-1",
				Namespace: task.Namespace,
				Labels: map[string]string{
					"kubechain.humanlayer.dev/task": task.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: kubechainv1alpha1.GroupVersion.String(),
						Kind:       "Task",
						Name:       task.Name,
						UID:        task.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Spec: kubechainv1alpha1.TaskRunSpec{
				TaskRef: &kubechainv1alpha1.LocalObjectReference{
					Name: task.Name,
				},
			},
		}

		if err := r.Create(ctx, taskRun); err != nil {
			logger.Error(err, "Failed to create TaskRun")
			return ctrl.Result{}, err
		}
		logger.Info("Created TaskRun", "name", taskRun.Name)
		r.recorder.Event(&task, corev1.EventTypeNormal, "TaskRunCreated", fmt.Sprintf("Created TaskRun %s", taskRun.Name))
	}

	statusUpdate.Status.Ready = true
	statusUpdate.Status.Status = kubechainv1alpha1.TaskStatusReady
	statusUpdate.Status.StatusDetail = "Task Run Created"
	r.recorder.Event(&task, corev1.EventTypeNormal, "ValidationSucceeded", "Task validation successful")

	// Update status
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update Task status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled task",
		"name", task.Name,
		"ready", statusUpdate.Status.Ready,
		"status", statusUpdate.Status.Status,
		"statusDetail", statusUpdate.Status.StatusDetail)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("task-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.Task{}).
		Complete(r)
}
