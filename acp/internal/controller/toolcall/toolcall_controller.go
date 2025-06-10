package toolcall

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.opentelemetry.io/otel/trace"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayer"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
)

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=toolcalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=toolcalls/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=tools,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// ToolCallReconciler is a clean, simple controller
type ToolCallReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	recorder     record.EventRecorder
	Tracer       trace.Tracer
	stateMachine *StateMachine

	// Dependencies
	MCPManager      mcpmanager.MCPManagerInterface
	HLClientFactory humanlayer.HumanLayerClientFactory
}

// Reconcile is the main reconciliation loop with flat switch dispatch
func (r *ToolCallReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var tc acp.ToolCall
	if err := r.Get(ctx, req.NamespacedName, &tc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle terminal states first
	if r.isTerminal(&tc) {
		return r.handleTerminal(ctx, &tc)
	}

	// Initialize span context if needed
	if tc.Status.SpanContext == nil {
		return r.handleSpanInit(ctx, &tc)
	}

	// Process based on current state
	switch {
	case tc.Status.Phase == "":
		return r.handleInitialize(ctx, &tc)
	case tc.Status.Phase == acp.ToolCallPhasePending && tc.Status.Status == acp.ToolCallStatusTypePending:
		return r.handleSetup(ctx, &tc)
	case tc.Status.Phase == acp.ToolCallPhasePending && tc.Status.Status == acp.ToolCallStatusTypeReady:
		return r.handleCheckApproval(ctx, &tc)
	case tc.Status.Phase == acp.ToolCallPhaseAwaitingHumanApproval:
		return r.handleWaitForApproval(ctx, &tc)
	case tc.Status.Phase == acp.ToolCallPhaseReadyToExecuteApprovedTool:
		return r.handleExecute(ctx, &tc)
	case tc.Status.Phase == acp.ToolCallPhaseAwaitingSubAgent:
		return r.handleWaitForSubAgent(ctx, &tc)
	case tc.Status.Phase == acp.ToolCallPhaseAwaitingHumanInput:
		return r.handleWaitForHumanInput(ctx, &tc)
	default:
		return r.handleUnknownPhase(ctx, &tc)
	}
}

// SetupWithManager sets up the controller with minimal configuration
func (r *ToolCallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("toolcall-controller")

	// Initialize dependencies if not provided
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManagerWithClient(r.Client)
	}
	if r.HLClientFactory == nil {
		factory, err := humanlayer.NewHumanLayerClientFactory("")
		if err != nil {
			return err
		}
		r.HLClientFactory = factory
	}

	// Create executor and state machine
	executor := NewToolExecutor(r.Client, r.MCPManager, r.HLClientFactory)
	r.stateMachine = NewStateMachine(r.Client, executor, r.Tracer, r.recorder)

	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.ToolCall{}).
		Complete(r)
}

// isTerminal checks if the ToolCall is in a terminal state
func (r *ToolCallReconciler) isTerminal(tc *acp.ToolCall) bool {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.isTerminal(tc)
}

// handleTerminal processes terminal states
func (r *ToolCallReconciler) handleTerminal(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// handleSpanInit initializes span context
func (r *ToolCallReconciler) handleSpanInit(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.initializeSpan(ctx, tc)
}

// handleInitialize processes initialization phase
func (r *ToolCallReconciler) handleInitialize(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.initialize(ctx, tc)
}

// handleSetup processes setup phase
func (r *ToolCallReconciler) handleSetup(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.setup(ctx, tc)
}

// handleCheckApproval processes check approval phase
func (r *ToolCallReconciler) handleCheckApproval(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.checkApproval(ctx, tc)
}

// handleWaitForApproval processes wait for approval phase
func (r *ToolCallReconciler) handleWaitForApproval(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.waitForApproval(ctx, tc)
}

// handleExecute processes execution phase
func (r *ToolCallReconciler) handleExecute(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.execute(ctx, tc)
}

// handleWaitForSubAgent processes wait for sub-agent phase
func (r *ToolCallReconciler) handleWaitForSubAgent(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.waitForSubAgent(ctx, tc)
}

// handleWaitForHumanInput processes wait for human input phase
func (r *ToolCallReconciler) handleWaitForHumanInput(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.waitForHumanInput(ctx, tc)
}

// handleUnknownPhase processes unknown phases
func (r *ToolCallReconciler) handleUnknownPhase(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}
	return r.stateMachine.fail(ctx, tc, fmt.Errorf("unknown phase: %s", tc.Status.Phase))
}

// ensureStateMachine initializes the state machine if not already initialized
func (r *ToolCallReconciler) ensureStateMachine() {
	if r.stateMachine != nil {
		return
	}

	// Initialize dependencies if not provided
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManagerWithClient(r.Client)
	}
	if r.HLClientFactory == nil {
		factory, err := humanlayer.NewHumanLayerClientFactory("")
		if err != nil {
			// In test scenarios, this might be a mock, so handle gracefully
			return
		}
		r.HLClientFactory = factory
	}

	// Create executor and state machine
	executor := NewToolExecutor(r.Client, r.MCPManager, r.HLClientFactory)
	r.stateMachine = NewStateMachine(r.Client, executor, r.Tracer, r.recorder)
}
