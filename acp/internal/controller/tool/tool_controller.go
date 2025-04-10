package tool

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/openai/openai-go"
)

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=tools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=tools/status,verbs=get;update;patch

// ToolReconciler reconciles a Tool object
type ToolReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile fetches a Tool resource, validates required fields, and marks it ready.
func (r *ToolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var tool acp.Tool
	if err := r.Get(ctx, req.NamespacedName, &tool); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", tool.Name, "type", tool.Spec.ToolType)

	// Create a copy for status update
	statusUpdate := tool.DeepCopy()

	// Initialize status if not set
	if statusUpdate.Status.Status == "" {
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.StatusDetail = "Validating configuration"
		r.recorder.Event(&tool, corev1.EventTypeNormal, "Initializing", "Starting validation")
	}

	// Validate Parameters JSON if present
	if tool.Spec.Parameters.Raw != nil {
		var params openai.FunctionParameters
		if err := json.Unmarshal(tool.Spec.Parameters.Raw, &params); err != nil {
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = "Error"
			statusUpdate.Status.StatusDetail = fmt.Sprintf("Invalid Parameters JSON: %v", err)
			r.recorder.Event(&tool, corev1.EventTypeWarning, "ValidationFailed", fmt.Sprintf("Invalid Parameters JSON: %v", err))
			if err := r.Status().Update(ctx, statusUpdate); err != nil {
				logger.Error(err, "Unable to update Tool status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
	}

	// All validations passed
	statusUpdate.Status.Ready = true
	statusUpdate.Status.Status = "Ready"
	statusUpdate.Status.StatusDetail = "Tool validation successful"
	r.recorder.Event(&tool, corev1.EventTypeNormal, "ValidationSucceeded", "Tool validation successful")

	// Update the status subresource.
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update Tool status", "name", tool.Name)
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled tool", "name", tool.Name, "type", tool.Spec.ToolType, "ready", statusUpdate.Status.Ready)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ToolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("tool-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.Tool{}).
		Complete(r)
}
