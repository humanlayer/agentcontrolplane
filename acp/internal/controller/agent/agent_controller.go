package agent

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=agents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=agents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=llms,verbs=get;list;watch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=mcpservers,verbs=get;list;watch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=contactchannels,verbs=get;list;watch

// AgentReconciler reconciles a Agent object with simple, direct validation
type AgentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	recorder     record.EventRecorder
	stateMachine *StateMachine
}

// NewAgentReconciler creates a new AgentReconciler with simple dependencies
func NewAgentReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
) *AgentReconciler {
	stateMachine := NewStateMachine(client, recorder)
	return &AgentReconciler{
		Client:       client,
		Scheme:       scheme,
		recorder:     recorder,
		stateMachine: stateMachine,
	}
}

// Reconcile handles agent reconciliation using StateMachine
func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var agent acp.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Delegate to StateMachine
	return r.stateMachine.Process(ctx, &agent)
}

// SetupWithManager sets up the controller with the Manager
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.Agent{}).
		Complete(r)
}

// NewAgentReconcilerForManager creates a fully configured AgentReconciler with simple dependencies
func NewAgentReconcilerForManager(mgr ctrl.Manager) (*AgentReconciler, error) {
	client := mgr.GetClient()
	scheme := mgr.GetScheme()
	recorder := mgr.GetEventRecorderFor("agent-controller")

	return NewAgentReconciler(client, scheme, recorder), nil
}
