package toolcall

import (
	"context"
	"fmt"
	"time"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// StateMachine handles all ToolCall state transitions in one place
type StateMachine struct {
	client   client.Client
	executor *ToolExecutor
	tracer   trace.Tracer
	recorder record.EventRecorder
}

// NewStateMachine creates a new state machine
func NewStateMachine(client client.Client, executor *ToolExecutor, tracer trace.Tracer, recorder record.EventRecorder) *StateMachine {
	return &StateMachine{
		client:   client,
		executor: executor,
		tracer:   tracer,
		recorder: recorder,
	}
}

// Process handles a ToolCall and returns the next action
func (sm *StateMachine) Process(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing ToolCall", "name", tc.Name, "phase", tc.Status.Phase, "status", tc.Status.Status)

	// Handle terminal states first
	if sm.isTerminal(tc) {
		return ctrl.Result{}, nil
	}

	// Initialize span context if needed
	if tc.Status.SpanContext == nil {
		return sm.initializeSpan(ctx, tc)
	}

	// Process based on current state
	switch {
	case tc.Status.Phase == "":
		return sm.initialize(ctx, tc)
	case tc.Status.Phase == acp.ToolCallPhasePending && tc.Status.Status == acp.ToolCallStatusTypePending:
		return sm.setup(ctx, tc)
	case tc.Status.Phase == acp.ToolCallPhasePending && tc.Status.Status == acp.ToolCallStatusTypeReady:
		return sm.checkApproval(ctx, tc)
	case tc.Status.Phase == acp.ToolCallPhaseAwaitingHumanApproval:
		return sm.waitForApproval(ctx, tc)
	case tc.Status.Phase == acp.ToolCallPhaseReadyToExecuteApprovedTool:
		return sm.execute(ctx, tc)
	case tc.Status.Phase == acp.ToolCallPhaseAwaitingSubAgent:
		return sm.waitForSubAgent(ctx, tc)
	case tc.Status.Phase == acp.ToolCallPhaseAwaitingHumanInput:
		return sm.waitForHumanInput(ctx, tc)
	default:
		return sm.fail(ctx, tc, fmt.Errorf("unknown phase: %s", tc.Status.Phase))
	}
}

// State transition methods

func (sm *StateMachine) initialize(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	tc.Status.Phase = acp.ToolCallPhasePending
	tc.Status.Status = acp.ToolCallStatusTypePending
	tc.Status.StatusDetail = "Initializing"
	tc.Status.StartTime = &metav1.Time{Time: time.Now()}

	return sm.updateAndRequeue(ctx, tc)
}

func (sm *StateMachine) setup(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	tc.Status.Status = acp.ToolCallStatusTypeReady
	tc.Status.StatusDetail = "Ready for execution"

	return sm.updateAndRequeue(ctx, tc)
}

func (sm *StateMachine) checkApproval(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	needsApproval, contactChannel, err := sm.executor.CheckApprovalRequired(ctx, tc)
	if err != nil {
		return sm.fail(ctx, tc, fmt.Errorf("failed to check approval requirement: %w", err))
	}

	if !needsApproval {
		// No approval needed, execute directly
		return sm.execute(ctx, tc)
	}

	// Request approval
	callID, err := sm.executor.RequestApproval(ctx, tc, contactChannel)
	if err != nil {
		return sm.failWithSpecificPhase(ctx, tc, acp.ToolCallPhaseErrorRequestingHumanApproval, fmt.Errorf("failed to request approval: %w", err))
	}

	tc.Status.Phase = acp.ToolCallPhaseAwaitingHumanApproval
	tc.Status.StatusDetail = fmt.Sprintf("Awaiting approval via %s", contactChannel.Name)
	tc.Status.ExternalCallID = callID

	// Emit event for the transition
	if sm.recorder != nil {
		sm.recorder.Event(tc, corev1.EventTypeNormal, "AwaitingHumanApproval",
			fmt.Sprintf("Awaiting human approval via %s", contactChannel.Name))
	}

	return sm.updateAndRequeue(ctx, tc, 5*time.Second)
}

func (sm *StateMachine) waitForApproval(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if tc.Status.ExternalCallID == "" {
		return sm.fail(ctx, tc, fmt.Errorf("missing external call ID"))
	}

	// Get contact channel for API key
	needsApproval, contactChannel, err := sm.executor.CheckApprovalRequired(ctx, tc)
	if err != nil || !needsApproval {
		return sm.fail(ctx, tc, fmt.Errorf("failed to get contact channel: %w", err))
	}

	functionCall, err := sm.executor.CheckApprovalStatus(ctx, tc.Status.ExternalCallID, contactChannel, tc.Namespace)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to check approval status")
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}

	if functionCall == nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	status := functionCall.GetStatus()
	approved, ok := status.GetApprovedOk()
	if !ok || approved == nil {
		// Still pending
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if *approved {
		tc.Status.Phase = acp.ToolCallPhaseReadyToExecuteApprovedTool
		tc.Status.StatusDetail = "Ready to execute approved tool"
		return sm.updateAndComplete(ctx, tc) // Complete - no requeue, let next reconcile handle execution
	} else {
		tc.Status.Phase = acp.ToolCallPhaseToolCallRejected
		tc.Status.Status = acp.ToolCallStatusTypeSucceeded
		tc.Status.StatusDetail = "Tool execution rejected"
		tc.Status.Result = fmt.Sprintf("Rejected: %s", status.GetComment())
		tc.Status.CompletionTime = &metav1.Time{Time: time.Now()}
		return sm.updateAndComplete(ctx, tc) // Complete - no requeue for rejected tools
	}
}

func (sm *StateMachine) execute(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	result, err := sm.executor.Execute(ctx, tc)
	if err != nil {
		// Handle specific error cases based on tool type
		if tc.Spec.ToolType == acp.ToolTypeHumanContact {
			return sm.failWithSpecificPhase(ctx, tc, acp.ToolCallPhaseErrorRequestingHumanInput, err)
		}
		return sm.fail(ctx, tc, fmt.Errorf("execution failed: %w", err))
	}

	// Handle special cases
	if tc.Spec.ToolType == acp.ToolTypeDelegateToAgent {
		tc.Status.Phase = acp.ToolCallPhaseAwaitingSubAgent
		tc.Status.StatusDetail = "Delegating to sub-agent"

		// Emit event for the transition
		if sm.recorder != nil {
			sm.recorder.Event(tc, corev1.EventTypeNormal, "DelegatingToSubAgent",
				"Delegating tool execution to sub-agent")
		}

		return sm.updateAndRequeue(ctx, tc, 5*time.Second)
	}

	if tc.Spec.ToolType == acp.ToolTypeHumanContact {
		tc.Status.Phase = acp.ToolCallPhaseAwaitingHumanInput
		tc.Status.StatusDetail = "Awaiting human input"

		// Emit event for the transition
		if sm.recorder != nil {
			sm.recorder.Event(tc, corev1.EventTypeNormal, "AwaitingHumanContact",
				"Awaiting human contact input")
		}

		return sm.updateAndRequeue(ctx, tc, 5*time.Second)
	}

	// Normal completion
	tc.Status.Phase = acp.ToolCallPhaseSucceeded
	tc.Status.Status = acp.ToolCallStatusTypeSucceeded
	tc.Status.StatusDetail = "Tool executed successfully"
	tc.Status.Result = result
	tc.Status.CompletionTime = &metav1.Time{Time: time.Now()}

	return sm.updateAndComplete(ctx, tc)
}

func (sm *StateMachine) waitForSubAgent(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	// Find child tasks
	var taskList acp.TaskList
	if err := sm.client.List(ctx, &taskList, client.InNamespace(tc.Namespace),
		client.MatchingLabels{"acp.humanlayer.dev/parent-toolcall": tc.Name}); err != nil {
		return sm.fail(ctx, tc, fmt.Errorf("failed to list child tasks: %w", err))
	}

	if len(taskList.Items) == 0 {
		return sm.fail(ctx, tc, fmt.Errorf("no child tasks found"))
	}

	childTask := &taskList.Items[0]

	if childTask.Status.Phase == acp.TaskPhaseFinalAnswer {
		tc.Status.Phase = acp.ToolCallPhaseSucceeded
		tc.Status.Status = acp.ToolCallStatusTypeSucceeded
		tc.Status.StatusDetail = "Sub-agent completed successfully"
		tc.Status.Result = childTask.Status.Output
		tc.Status.CompletionTime = &metav1.Time{Time: time.Now()}

		// Emit event for successful sub-agent completion
		if sm.recorder != nil {
			sm.recorder.Event(tc, corev1.EventTypeNormal, "SubAgentCompleted",
				"Sub-agent task completed successfully")
		}

		return sm.updateAndComplete(ctx, tc)
	}

	if childTask.Status.Phase == acp.TaskPhaseFailed {
		// Emit event for failed sub-agent
		if sm.recorder != nil {
			sm.recorder.Event(tc, corev1.EventTypeWarning, "SubAgentFailed",
				"Sub-agent task failed")
		}

		// Set custom status for sub-agent failure
		tc.Status.Phase = acp.ToolCallPhaseFailed
		tc.Status.Status = acp.ToolCallStatusTypeError
		tc.Status.StatusDetail = "Sub-agent task failed"
		tc.Status.Error = childTask.Status.Error
		tc.Status.CompletionTime = &metav1.Time{Time: time.Now()}

		return sm.updateAndComplete(ctx, tc)
	}

	// Still in progress
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (sm *StateMachine) waitForHumanInput(_ context.Context, _ *acp.ToolCall) (ctrl.Result, error) {
	// This would check HumanLayer API for human input completion
	// For now, just requeue
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (sm *StateMachine) initializeSpan(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	// Initialize span context
	_, span := sm.tracer.Start(ctx, "ToolCall")
	defer span.End()

	tc.Status.SpanContext = &acp.SpanContext{
		TraceID: span.SpanContext().TraceID().String(),
		SpanID:  span.SpanContext().SpanID().String(),
	}

	return sm.updateAndRequeue(ctx, tc)
}

// Helper methods

func (sm *StateMachine) isTerminal(tc *acp.ToolCall) bool {
	return tc.Status.Status == acp.ToolCallStatusTypeError ||
		tc.Status.Status == acp.ToolCallStatusTypeSucceeded
}

func (sm *StateMachine) fail(ctx context.Context, tc *acp.ToolCall, err error) (ctrl.Result, error) {
	return sm.failWithSpecificPhase(ctx, tc, acp.ToolCallPhaseFailed, err)
}

func (sm *StateMachine) failWithSpecificPhase(ctx context.Context, tc *acp.ToolCall, phase acp.ToolCallPhase, err error) (ctrl.Result, error) {
	// Fetch the latest version to avoid UID conflicts
	namespacedName := client.ObjectKey{Name: tc.Name, Namespace: tc.Namespace}
	latestTC := &acp.ToolCall{}
	if getErr := sm.client.Get(ctx, namespacedName, latestTC); getErr != nil {
		return ctrl.Result{}, getErr
	}

	latestTC.Status.Phase = phase
	latestTC.Status.Status = acp.ToolCallStatusTypeError
	latestTC.Status.StatusDetail = err.Error()
	latestTC.Status.Error = err.Error()
	latestTC.Status.CompletionTime = &metav1.Time{Time: time.Now()}

	// Record event
	if updateErr := sm.client.Status().Update(ctx, latestTC); updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{}, nil
}

func (sm *StateMachine) updateAndRequeue(ctx context.Context, tc *acp.ToolCall, after ...time.Duration) (ctrl.Result, error) {
	// Fetch the latest version to avoid UID conflicts
	namespacedName := client.ObjectKey{Name: tc.Name, Namespace: tc.Namespace}
	latestTC := &acp.ToolCall{}
	if err := sm.client.Get(ctx, namespacedName, latestTC); err != nil {
		return ctrl.Result{}, err
	}

	// Copy status fields to latest version
	latestTC.Status = tc.Status

	if err := sm.client.Status().Update(ctx, latestTC); err != nil {
		return ctrl.Result{}, err
	}

	if len(after) > 0 {
		return ctrl.Result{RequeueAfter: after[0]}, nil
	}
	return ctrl.Result{Requeue: true}, nil
}

func (sm *StateMachine) updateStatus(ctx context.Context, tc *acp.ToolCall) error {
	// Fetch the latest version to avoid UID conflicts
	namespacedName := client.ObjectKey{Name: tc.Name, Namespace: tc.Namespace}
	latestTC := &acp.ToolCall{}
	if err := sm.client.Get(ctx, namespacedName, latestTC); err != nil {
		return err
	}

	// Copy status fields to latest version
	latestTC.Status = tc.Status

	return sm.client.Status().Update(ctx, latestTC)
}

func (sm *StateMachine) updateAndComplete(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	if err := sm.updateStatus(ctx, tc); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
