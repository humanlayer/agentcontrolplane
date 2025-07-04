package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	"github.com/humanlayer/agentcontrolplane/acp/internal/validation"
)

// StateMachine handles all Task state transitions following the ToolCallController pattern
type StateMachine struct {
	client            client.Client
	recorder          record.EventRecorder
	llmClientFactory  LLMClientFactory
	mcpManager        MCPManager
	humanLayerFactory HumanLayerClientFactory
	toolAdapter       ToolAdapter
	tracer            trace.Tracer
	// Task-level mutexes to prevent concurrent LLM requests (single-pod optimization)
	taskMutexes  map[string]*sync.Mutex
	mutexMapLock sync.RWMutex
	// Distributed locking for multi-pod deployments
	namespace     string
	podName       string
	leaseDuration time.Duration
}

// NewStateMachine creates a new state machine with all dependencies
func NewStateMachine(
	client client.Client,
	recorder record.EventRecorder,
	llmClientFactory LLMClientFactory,
	mcpManager MCPManager,
	humanLayerFactory HumanLayerClientFactory,
	toolAdapter ToolAdapter,
	tracer trace.Tracer,
) *StateMachine {
	// Get pod identity for distributed locking
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		suffix, _ := validation.GenerateK8sRandomString(8)
		podName = "acp-controller-manager-" + suffix
	}

	return &StateMachine{
		client:            client,
		recorder:          recorder,
		llmClientFactory:  llmClientFactory,
		mcpManager:        mcpManager,
		humanLayerFactory: humanLayerFactory,
		toolAdapter:       toolAdapter,
		tracer:            tracer,
		taskMutexes:       make(map[string]*sync.Mutex),
		mutexMapLock:      sync.RWMutex{},
		namespace:         namespace,
		podName:           podName,
		leaseDuration:     30 * time.Second, // 30 second lease duration
	}
}

// Process handles a Task and returns the next action
func (sm *StateMachine) Process(ctx context.Context, task *acp.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Processing Task", "name", task.Name, "phase", task.Status.Phase, "status", task.Status.Status)

	// Handle terminal states first
	if sm.isTerminal(task) {
		return sm.handleTerminal(ctx, task)
	}

	// Initialize span context if needed
	if task.Status.Phase == "" || task.Status.SpanContext == nil {
		return sm.initialize(ctx, task)
	}

	// Route to appropriate phase handler
	switch task.Status.Phase {
	case acp.TaskPhaseFinalAnswer:
		return sm.handleTerminal(ctx, task)
	case acp.TaskPhaseFailed:
		return sm.handleTerminal(ctx, task)
	case acp.TaskPhaseInitializing, acp.TaskPhasePending:
		return sm.validateAgent(ctx, task)
	case acp.TaskPhaseReadyForLLM:
		return sm.sendLLMRequest(ctx, task)
	case acp.TaskPhaseToolCallsPending:
		return sm.checkToolCalls(ctx, task)
	default:
		return sm.handleUnknownPhase(ctx, task)
	}
}

// State transition methods

// initialize handles empty -> "Initializing" transition
func (sm *StateMachine) initialize(ctx context.Context, task *acp.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Create a new *root* span for the Task
	spanCtx, span := sm.tracer.Start(ctx, "Task",
		trace.WithSpanKind(trace.SpanKindServer), // optional
	)
	// Do NOT 'span.End()' here—this is your single "root" for the entire Task lifetime.

	// Set initial phase
	task.Status.Phase = acp.TaskPhaseInitializing
	task.Status.Status = acp.TaskStatusTypePending
	task.Status.StatusDetail = "Initializing Task"

	// Save span context for future use
	task.Status.SpanContext = &acp.SpanContext{
		TraceID: span.SpanContext().TraceID().String(),
		SpanID:  span.SpanContext().SpanID().String(),
	}

	if err := sm.client.Status().Update(spanCtx, task); err != nil {
		logger.Error(err, "Failed to update Task status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// validateAgent handles "Initializing"/"Pending" -> "ReadyForLLM"/"Pending"/"Failed" transitions
func (sm *StateMachine) validateAgent(ctx context.Context, task *acp.Task) (ctrl.Result, error) {
	statusUpdate := task.DeepCopy()

	// First validate task and agent existence/readiness
	agent, result, err := sm.validateTaskAndAgent(ctx, task, statusUpdate)
	if err != nil || !result.IsZero() {
		return result, err
	}

	// If validation passes, prepare for LLM
	return sm.prepareForLLM(ctx, task, statusUpdate, agent)
}

// sendLLMRequest handles "ReadyForLLM" -> "FinalAnswer"/"ToolCallsPending"/"Failed" transitions
func (sm *StateMachine) sendLLMRequest(ctx context.Context, task *acp.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	statusUpdate := task.DeepCopy()

	// Acquire task-specific mutex to serialize LLM requests (single-pod optimization)
	mutex := sm.getTaskMutex(task.Name)
	mutex.Lock()
	defer mutex.Unlock()

	// Acquire distributed lease for multi-pod coordination
	lease, acquired, err := sm.acquireTaskLease(ctx, task.Name)
	if err != nil {
		logger.Error(err, "Failed to acquire distributed task lease")
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	if !acquired {
		logger.V(1).Info("Task lease held by another pod, requeuing", "task", task.Name)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	defer sm.releaseTaskLease(ctx, lease)

	// Get agent and credentials
	agent, result, err := sm.validateTaskAndAgent(ctx, task, statusUpdate)
	if err != nil || !result.IsZero() {
		return result, err
	}

	llm, apiKey, err := sm.getLLMAndCredentials(ctx, agent, task, statusUpdate)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create LLM client
	llmClient, err := sm.llmClientFactory.CreateClient(ctx, llm, apiKey)
	if err != nil {
		logger.Error(err, "Failed to create LLM client")

		update := TaskStatusUpdate{
			Ready:        false,
			Status:       acp.TaskStatusTypeError,
			Phase:        acp.TaskPhaseFailed,
			StatusDetail: "Failed to create LLM client: " + err.Error(),
			Error:        err.Error(),
			EventType:    corev1.EventTypeWarning,
			EventReason:  "LLMClientCreationFailed",
			EventMessage: err.Error(),
		}

		sm.endTaskTrace(ctx, statusUpdate, codes.Error, "Failed to create LLM client: "+err.Error())

		if updateErr := sm.updateTaskStatus(ctx, statusUpdate, update); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil
	}

	// Collect tools and send LLM request
	tools := sm.collectTools(ctx, agent)

	// Only send event if not already in this phase to prevent duplicates
	if task.Status.Phase != acp.TaskPhaseReadyForLLM || statusUpdate.Status.StatusDetail != "Sending request to LLM" {
		sm.recorder.Event(task, corev1.EventTypeNormal, "SendingContextWindowToLLM", "Sending context window to LLM")
		// Update status to indicate we're sending to LLM
		statusUpdate.Status.StatusDetail = "Sending request to LLM"
		if err := sm.client.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update Task status before LLM request")
		}
	}

	// Create child span for LLM call
	llmCtx, llmSpan := sm.createLLMRequestSpan(ctx, task, len(task.Status.ContextWindow), len(tools))
	if llmSpan != nil {
		defer llmSpan.End()
	}

	output, err := llmClient.SendRequest(llmCtx, task.Status.ContextWindow, tools)
	if err != nil {
		return sm.handleLLMError(ctx, statusUpdate, err, llmSpan)
	}

	// Mark span as successful
	if llmSpan != nil {
		llmSpan.SetStatus(codes.Ok, "LLM request succeeded")
		llmSpan.SetAttributes(
			attribute.String("llm.request.model", llm.Spec.Parameters.Model),
			attribute.Int("llm.response.tool_calls.count", len(output.ToolCalls)),
			attribute.Bool("llm.response.has_content", output.Content != ""),
		)
	}

	llmResult, err := sm.processLLMResponse(ctx, output, task, statusUpdate, tools)
	if err != nil {
		logger.Error(err, "Failed to process LLM response")

		update := TaskStatusUpdate{
			Ready:        false,
			Status:       acp.TaskStatusTypeError,
			Phase:        acp.TaskPhaseFailed,
			StatusDetail: fmt.Sprintf("Failed to process LLM response: %v", err),
			Error:        err.Error(),
			EventType:    corev1.EventTypeWarning,
			EventReason:  "LLMResponseProcessingFailed",
			EventMessage: err.Error(),
		}

		if updateErr := sm.updateTaskStatus(ctx, statusUpdate, update); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status after LLM response processing error")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil
	}

	if !llmResult.IsZero() {
		return llmResult, nil
	}

	// Update final status
	if err := sm.client.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update Task status")
		return ctrl.Result{}, err
	}

	logger.Info("Task reconciled", "phase", statusUpdate.Status.Phase)

	return ctrl.Result{}, nil
}

// checkToolCalls handles "ToolCallsPending" -> "ReadyForLLM"/"ToolCallsPending" transitions
func (sm *StateMachine) checkToolCalls(ctx context.Context, task *acp.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// List all tool calls for this Task
	toolCalls := &acp.ToolCallList{}
	if err := sm.client.List(ctx, toolCalls, client.InNamespace(task.Namespace), client.MatchingLabels{
		"acp.humanlayer.dev/task":            task.Name,
		"acp.humanlayer.dev/toolcallrequest": task.Status.ToolCallRequestID,
	}); err != nil {
		logger.Error(err, "Failed to list tool calls")
		return ctrl.Result{}, err
	}

	// Check if all tool calls are completed
	allCompleted := true
	for _, tc := range toolCalls.Items {
		if tc.Status.Status != acp.ToolCallStatusTypeSucceeded &&
			// todo separate between send-to-model failures and tool-is-retrying failures
			tc.Status.Status != acp.ToolCallStatusTypeError {
			allCompleted = false
			break
		}
	}

	if !allCompleted {
		return ctrl.Result{RequeueAfter: DefaultRequeueDelay}, nil
	}

	// All tool calls are completed, append results to context window
	for _, tc := range toolCalls.Items {
		task.Status.ContextWindow = append(task.Status.ContextWindow, acp.Message{
			Role:       "tool",
			Content:    tc.Status.Result,
			ToolCallID: tc.Spec.ToolCallID,
		})
	}

	// Update status
	task.Status.Phase = acp.TaskPhaseReadyForLLM
	task.Status.Status = acp.TaskStatusTypeReady
	task.Status.StatusDetail = "All tool calls completed, ready to send tool results to LLM"
	task.Status.Error = "" // Clear previous error
	sm.recorder.Event(task, corev1.EventTypeNormal, "AllToolCallsCompleted", "All tool calls completed")

	if err := sm.client.Status().Update(ctx, task); err != nil {
		logger.Error(err, "Failed to update Task status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// handleTerminal handles terminal states like "FinalAnswer" and "Failed"
func (sm *StateMachine) handleTerminal(ctx context.Context, task *acp.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Ending trace", "phase", task.Status.Phase)

	switch task.Status.Phase {
	case acp.TaskPhaseFinalAnswer:
		sm.endTaskTrace(ctx, task, codes.Ok, "Task completed successfully with final answer")
	case acp.TaskPhaseFailed:
		message := task.Status.Error
		if message == "" {
			message = "Task failed"
		}
		sm.endTaskTrace(ctx, task, codes.Error, message)
	}

	return ctrl.Result{}, nil
}

// Helper methods

// isTerminal checks if the Task is in a terminal state
func (sm *StateMachine) isTerminal(task *acp.Task) bool {
	return task.Status.Phase == acp.TaskPhaseFinalAnswer ||
		task.Status.Phase == acp.TaskPhaseFailed
}

// handleUnknownPhase handles tasks in unknown phases
func (sm *StateMachine) handleUnknownPhase(ctx context.Context, task *acp.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Task in unknown phase", "phase", task.Status.Phase)
	return ctrl.Result{}, nil
}

// Helper methods extracted from original controller

func (sm *StateMachine) validateTaskAndAgent(ctx context.Context, task *acp.Task, statusUpdate *acp.Task) (*acp.Agent, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var agent acp.Agent
	if err := sm.client.Get(ctx, client.ObjectKey{Namespace: task.Namespace, Name: task.Spec.AgentRef.Name}, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Agent not found, waiting for it to exist")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = acp.TaskStatusTypePending
			statusUpdate.Status.Phase = acp.TaskPhasePending
			statusUpdate.Status.StatusDetail = "Waiting for Agent to exist"
			statusUpdate.Status.Error = "" // Clear previous error
			sm.recorder.Event(task, corev1.EventTypeNormal, "Waiting", "Waiting for Agent to exist")
		} else {
			logger.Error(err, "Failed to get Agent")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = acp.TaskStatusTypeError
			statusUpdate.Status.Phase = acp.TaskPhaseFailed
			statusUpdate.Status.Error = err.Error()
			sm.recorder.Event(task, corev1.EventTypeWarning, "AgentFetchFailed", err.Error())
		}
		if updateErr := sm.client.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return nil, ctrl.Result{}, updateErr
		}
		return nil, ctrl.Result{RequeueAfter: DefaultRequeueDelay}, nil
	}

	// Check if agent is ready
	if !agent.Status.Ready {
		logger.Info("Agent exists but is not ready", "agent", agent.Name)
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = acp.TaskStatusTypePending
		statusUpdate.Status.Phase = acp.TaskPhasePending
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Waiting for agent %q to become ready", agent.Name)
		statusUpdate.Status.Error = "" // Clear previous error
		sm.recorder.Event(task, corev1.EventTypeNormal, "Waiting", fmt.Sprintf("Waiting for agent %q to become ready", agent.Name))
		if err := sm.client.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update Task status")
			return nil, ctrl.Result{}, err
		}
		return nil, ctrl.Result{RequeueAfter: DefaultRequeueDelay}, nil
	}

	return &agent, ctrl.Result{}, nil
}

func (sm *StateMachine) prepareForLLM(ctx context.Context, task *acp.Task, statusUpdate *acp.Task, agent *acp.Agent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if statusUpdate.Status.Phase == acp.TaskPhaseInitializing || statusUpdate.Status.Phase == acp.TaskPhasePending {
		if err := validation.ValidateTaskMessageInput(task.Spec.UserMessage, task.Spec.ContextWindow); err != nil {
			return sm.setValidationError(ctx, task, statusUpdate, err)
		}

		if err := validation.ValidateContactChannelRef(ctx, sm.client, task); err != nil {
			return sm.setValidationError(ctx, task, statusUpdate, err)
		}

		initialContextWindow := buildInitialContextWindow(task.Spec.ContextWindow, agent.Spec.System, task.Spec.UserMessage)

		statusUpdate.Status.UserMsgPreview = validation.GetUserMessagePreview(task.Spec.UserMessage, task.Spec.ContextWindow)
		statusUpdate.Status.ContextWindow = initialContextWindow
		statusUpdate.Status.Phase = acp.TaskPhaseReadyForLLM
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = acp.TaskStatusTypeReady
		statusUpdate.Status.StatusDetail = "Ready to send to LLM"
		statusUpdate.Status.Error = ""

		// Only send event if not already validated to prevent duplicates
		if task.Status.Phase != acp.TaskPhaseReadyForLLM {
			sm.recorder.Event(task, corev1.EventTypeNormal, "ValidationSucceeded", "Task validation succeeded")
		}
		if err := sm.client.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update Task status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (sm *StateMachine) setValidationError(ctx context.Context, task *acp.Task, statusUpdate *acp.Task, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "Validation failed")
	statusUpdate.Status.Ready = false
	statusUpdate.Status.Status = acp.TaskStatusTypeError
	statusUpdate.Status.Phase = acp.TaskPhaseFailed
	statusUpdate.Status.StatusDetail = err.Error()
	statusUpdate.Status.Error = err.Error()
	sm.recorder.Event(task, corev1.EventTypeWarning, "ValidationFailed", err.Error())
	if updateErr := sm.client.Status().Update(ctx, statusUpdate); updateErr != nil {
		logger.Error(updateErr, "Failed to update Task status")
		return ctrl.Result{}, updateErr
	}
	return ctrl.Result{}, err
}

// Additional helper methods from original controller

func (sm *StateMachine) getLLMAndCredentials(ctx context.Context, agent *acp.Agent, task *acp.Task, statusUpdate *acp.Task) (acp.LLM, string, error) {
	logger := log.FromContext(ctx)

	// Get the LLM
	var llm acp.LLM
	if err := sm.client.Get(ctx, client.ObjectKey{Namespace: task.Namespace, Name: agent.Spec.LLMRef.Name}, &llm); err != nil {
		logger.Error(err, "Failed to get LLM")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = acp.TaskStatusTypeError
		statusUpdate.Status.Phase = acp.TaskPhaseFailed
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Failed to get LLM: %v", err)
		statusUpdate.Status.Error = err.Error()
		sm.recorder.Event(task, corev1.EventTypeWarning, "LLMFetchFailed", err.Error())
		if updateErr := sm.client.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	// Get the API key from the secret
	var secret corev1.Secret
	if err := sm.client.Get(ctx, client.ObjectKey{
		Namespace: task.Namespace,
		Name:      llm.Spec.APIKeyFrom.SecretKeyRef.Name,
	}, &secret); err != nil {
		logger.Error(err, "Failed to get API key secret")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = acp.TaskStatusTypeError
		statusUpdate.Status.Phase = acp.TaskPhaseFailed
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Failed to get API key secret: %v", err)
		statusUpdate.Status.Error = err.Error()
		sm.recorder.Event(task, corev1.EventTypeWarning, "APIKeySecretFetchFailed", err.Error())
		if updateErr := sm.client.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	apiKey := string(secret.Data[llm.Spec.APIKeyFrom.SecretKeyRef.Key])
	if apiKey == "" {
		err := fmt.Errorf("API key is empty")
		logger.Error(err, "Empty API key")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = acp.TaskStatusTypeError
		statusUpdate.Status.Phase = acp.TaskPhaseFailed
		statusUpdate.Status.StatusDetail = "API key is empty"
		statusUpdate.Status.Error = err.Error()
		sm.recorder.Event(task, corev1.EventTypeWarning, "EmptyAPIKey", "API key is empty")
		if updateErr := sm.client.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	return llm, apiKey, nil
}

func (sm *StateMachine) collectTools(ctx context.Context, agent *acp.Agent) []llmclient.Tool {
	logger := log.FromContext(ctx)
	tools := make([]llmclient.Tool, 0)

	// Iterate through each MCP server directly to maintain server-tool association
	for _, serverRef := range agent.Spec.MCPServers {
		mcpTools, found := sm.mcpManager.GetTools(serverRef.Name)
		if !found {
			logger.Info("Server not found or has no tools", "server", serverRef.Name)
			continue
		}
		// Use the injected tool adapter to convert tools
		tools = append(tools, sm.toolAdapter.ConvertMCPTools(mcpTools, serverRef.Name)...)
		logger.Info("Added MCP server tools", "server", serverRef.Name, "toolCount", len(mcpTools))
	}

	// Collect and convert HumanContactChannel tools
	contactChannels := make([]acp.ContactChannel, 0, len(agent.Status.ValidHumanContactChannels))
	for _, validChannel := range agent.Status.ValidHumanContactChannels {
		channel := &acp.ContactChannel{}
		if err := sm.client.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: validChannel.Name}, channel); err != nil {
			logger.Error(err, "Failed to get ContactChannel", "name", validChannel.Name)
			continue
		}
		contactChannels = append(contactChannels, *channel)
	}
	tools = append(tools, sm.toolAdapter.ConvertContactChannels(contactChannels)...)
	logger.Info("Added contact channel tools", "count", len(contactChannels))

	// Collect and convert sub-agent tools
	subAgents := make([]acp.Agent, 0, len(agent.Spec.SubAgents))
	for _, subAgentRef := range agent.Spec.SubAgents {
		subAgent := &acp.Agent{}
		if err := sm.client.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: subAgentRef.Name}, subAgent); err != nil {
			logger.Error(err, "Failed to get sub-agent", "name", subAgentRef.Name)
			continue
		}
		subAgents = append(subAgents, *subAgent)
	}
	tools = append(tools, sm.toolAdapter.ConvertSubAgents(subAgents)...)
	logger.Info("Added sub-agent delegate tools", "count", len(subAgents))

	return tools
}

func (sm *StateMachine) createLLMRequestSpan(
	ctx context.Context, // This context should already have the root span attached via contextWithTaskSpan
	task *acp.Task,
	numMessages int,
	numTools int,
) (context.Context, trace.Span) {
	// Now that ctx has the *root* span in it (from contextWithTaskSpan), we can start a child:
	childCtx, childSpan := sm.tracer.Start(ctx, "LLMRequest",
		trace.WithSpanKind(trace.SpanKindClient), // Mark as client span for LLM call
	)

	childSpan.SetAttributes(
		attribute.Int("acp.task.context_window.messages", numMessages),
		attribute.Int("acp.task.tools.count", numTools),
		attribute.String("acp.task.name", task.Name), // Add task name for context
	)

	return childCtx, childSpan
}

func (sm *StateMachine) processLLMResponse(ctx context.Context, output *acp.Message, task *acp.Task, statusUpdate *acp.Task, tools []llmclient.Tool) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if output.Content != "" {
		// Check if this is a v1beta3 task - if so, create respond_to_human tool call instead of normal final answer
		if task.Labels != nil && task.Labels["acp.humanlayer.dev/v1beta3"] == "true" {
			return sm.handleV1Beta3FinalAnswer(ctx, output, task, statusUpdate, tools)
		}

		// final answer branch
		statusUpdate.Status.Output = output.Content
		statusUpdate.Status.Phase = acp.TaskPhaseFinalAnswer
		statusUpdate.Status.Ready = true
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, acp.Message{
			Role:    "assistant",
			Content: output.Content,
		})
		statusUpdate.Status.Status = acp.TaskStatusTypeReady
		statusUpdate.Status.StatusDetail = "LLM final response received"
		statusUpdate.Status.Error = ""

		// Only send event if not already in final answer phase to prevent duplicates
		if task.Status.Phase != acp.TaskPhaseFinalAnswer {
			sm.recorder.Event(task, corev1.EventTypeNormal, "LLMFinalAnswer", "LLM response received successfully")
		}

		// If task has contactChannelRef, send the final result via HumanLayer API
		if task.Spec.ContactChannelRef != nil {
			sm.notifyHumanLayerAPIAsync(ctx, task, output.Content)
		}

		// End the task trace with OK status since we have a final answer.
		// The context passed here should ideally be the one from Reconcile after contextWithTaskSpan.
		// r.endTaskTrace(ctx, task, codes.Ok, "Task completed successfully with final answer")
		// NOTE: The plan suggests calling endTaskTrace from Reconcile when phase is FinalAnswer,
		// so we might not need to call it here. Let's follow the plan's structure.
	} else {
		// Generate a unique ID for this set of tool calls
		toolCallRequestId, err := validation.GenerateK8sRandomString(7)
		if err != nil {
			logger.Error(err, "Failed to generate toolCallRequestId")
			return ctrl.Result{}, err
		}
		logger.Info("Generated toolCallRequestId for tool calls", "id", toolCallRequestId)

		// tool call branch: create ToolCall objects for each tool call returned by the LLM.
		statusUpdate.Status.Output = ""
		statusUpdate.Status.Phase = acp.TaskPhaseToolCallsPending
		statusUpdate.Status.ToolCallRequestID = toolCallRequestId
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, acp.Message{
			Role:      "assistant",
			ToolCalls: output.ToolCalls,
		})
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = acp.TaskStatusTypeReady
		statusUpdate.Status.StatusDetail = "LLM response received, tool calls pending"
		statusUpdate.Status.Error = ""
		sm.recorder.Event(task, corev1.EventTypeNormal, "ToolCallsPending", "LLM response received, tool calls pending")

		// Update the parent's status before creating tool call objects.
		if err := sm.client.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Unable to update Task status")
			return ctrl.Result{}, err
		}

		// todo should this technically happen before the status update? is there a chance they get dropped?
		return sm.createToolCalls(ctx, task, statusUpdate, output.ToolCalls, tools)
	}
	return ctrl.Result{}, nil
}

func (sm *StateMachine) createToolCalls(ctx context.Context, task *acp.Task, statusUpdate *acp.Task, toolCalls []acp.MessageToolCall, tools []llmclient.Tool) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if statusUpdate.Status.ToolCallRequestID == "" {
		err := fmt.Errorf("no ToolCallRequestID found in statusUpdate, cannot create tool calls")
		logger.Error(err, "Missing ToolCallRequestID")
		return ctrl.Result{}, err
	}

	// Create a map of tool name to tool type for quick lookup
	toolTypeMap := buildToolTypeMap(tools)

	// For each tool call, create a new ToolCall with a unique name using the ToolCallRequestID
	for i, tc := range toolCalls {
		newName := fmt.Sprintf("%s-%s-tc-%02d", statusUpdate.Name, statusUpdate.Status.ToolCallRequestID, i+1)
		toolType := toolTypeMap[tc.Function.Name]

		newTC := &acp.ToolCall{
			ObjectMeta: metav1.ObjectMeta{
				Name:      newName,
				Namespace: statusUpdate.Namespace,
				Labels: map[string]string{
					"acp.humanlayer.dev/task":            statusUpdate.Name,
					"acp.humanlayer.dev/toolcallrequest": statusUpdate.Status.ToolCallRequestID,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "acp.humanlayer.dev/v1alpha1",
						Kind:       "Task",
						Name:       statusUpdate.Name,
						UID:        statusUpdate.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Spec: acp.ToolCallSpec{
				ToolCallID: tc.ID,
				TaskRef: acp.LocalObjectReference{
					Name: statusUpdate.Name,
				},
				ToolRef: acp.LocalObjectReference{
					Name: tc.Function.Name,
				},
				ToolType:  toolTypeMap[tc.Function.Name],
				Arguments: tc.Function.Arguments,
			},
		}
		if err := sm.client.Create(ctx, newTC); err != nil {
			logger.Error(err, "Failed to create ToolCall", "name", newName)
			return ctrl.Result{}, err
		}
		logger.Info("Created ToolCall", "name", newName, "requestId", statusUpdate.Status.ToolCallRequestID, "toolType", toolType)
		sm.recorder.Event(task, corev1.EventTypeNormal, "ToolCallCreated", "Created ToolCall "+newName)
	}
	return ctrl.Result{RequeueAfter: DefaultRequeueDelay}, nil
}

func (sm *StateMachine) handleLLMError(ctx context.Context, statusUpdate *acp.Task, err error, llmSpan trace.Span) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "LLM request failed")

	// Check for LLMRequestError with 4xx status code
	var llmErr *llmclient.LLMRequestError
	is4xxError := errors.As(err, &llmErr) && llmErr.StatusCode >= 400 && llmErr.StatusCode < 500

	var update TaskStatusUpdate
	if is4xxError {
		logger.Info("LLM request failed with 4xx status code, marking as failed",
			"statusCode", llmErr.StatusCode,
			"message", llmErr.Message)

		update = TaskStatusUpdate{
			Ready:        false,
			Status:       acp.TaskStatusTypeError,
			Phase:        acp.TaskPhaseFailed,
			StatusDetail: fmt.Sprintf("LLM request failed: %v", err),
			Error:        err.Error(),
			EventType:    corev1.EventTypeWarning,
			EventReason:  "LLMRequestFailed4xx",
			EventMessage: fmt.Sprintf("LLM request failed with status %d: %s", llmErr.StatusCode, llmErr.Message),
		}
	} else {
		// For non-4xx errors, preserve current phase (will retry)
		update = TaskStatusUpdate{
			Ready:        false,
			Status:       acp.TaskStatusTypeError,
			Phase:        statusUpdate.Status.Phase, // Preserve current phase
			StatusDetail: fmt.Sprintf("LLM request failed: %v", err),
			Error:        err.Error(),
			EventType:    corev1.EventTypeWarning,
			EventReason:  "LLMRequestFailed",
			EventMessage: err.Error(),
		}
	}

	// Record error in span
	if llmSpan != nil {
		llmSpan.RecordError(err)
		llmSpan.SetStatus(codes.Error, err.Error())
	}

	// Update status
	if updateErr := sm.updateTaskStatus(ctx, statusUpdate, update); updateErr != nil {
		logger.Error(updateErr, "Failed to update Task status after LLM error")
		return ctrl.Result{}, updateErr
	}

	// If 4xx error, don't retry (terminal state)
	if is4xxError {
		return ctrl.Result{}, nil
	}

	// For other errors, return the error so controller-runtime handles retry
	return ctrl.Result{RequeueAfter: DefaultRequeueDelay}, err
}

func (sm *StateMachine) updateTaskStatus(ctx context.Context, task *acp.Task, update TaskStatusUpdate) error {
	task.Status.Ready = update.Ready
	task.Status.Status = update.Status
	task.Status.Phase = update.Phase
	task.Status.StatusDetail = update.StatusDetail
	task.Status.Error = update.Error

	if update.EventType != "" && update.EventReason != "" {
		sm.recorder.Event(task, update.EventType, update.EventReason, update.EventMessage)
	}

	return sm.client.Status().Update(ctx, task)
}

func (sm *StateMachine) endTaskTrace(ctx context.Context, task *acp.Task, code codes.Code, message string) {
	logger := log.FromContext(ctx)
	if task.Status.SpanContext == nil {
		logger.Info("No span context found in task status, cannot end trace")
		return
	}

	// Reattach the parent's context again to ensure the final span is correctly parented.
	ctx = sm.contextWithTaskSpan(ctx, task)

	// Now create a final child span to mark "root" completion.
	_, span := sm.tracer.Start(ctx, "EndTaskSpan")
	defer span.End() // End this specific child span immediately.

	span.SetStatus(code, message)
	// Add any last attributes if needed
	span.SetAttributes(attribute.String("task.name", task.Name))

	logger.V(1).Info("Trace ended", "status", code.String())
}

func (sm *StateMachine) contextWithTaskSpan(ctx context.Context, task *acp.Task) context.Context {
	if task.Status.SpanContext == nil || task.Status.SpanContext.TraceID == "" || task.Status.SpanContext.SpanID == "" {
		return ctx // no root yet or invalid context
	}

	sc, err := reconstructSpanContext(task.Status.SpanContext.TraceID, task.Status.SpanContext.SpanID)
	if err != nil {
		log.FromContext(ctx).V(1).Info("Failed to reconstruct span context", "error", err)
		return ctx
	}

	return trace.ContextWithSpanContext(ctx, sc)
}

func (sm *StateMachine) notifyHumanLayerAPIAsync(ctx context.Context, task *acp.Task, result string) {
	go func() {
		notifyCtx, cancel := context.WithTimeout(ctx, HumanLayerAPITimeout)
		defer cancel()

		taskCopy := task.DeepCopy()

		if err := sm.sendFinalResultViaHumanLayerAPI(notifyCtx, taskCopy, result); err != nil {
			// Use structured logging instead of recorder in goroutine
			contactChannelName := ""
			if taskCopy.Spec.ContactChannelRef != nil {
				contactChannelName = taskCopy.Spec.ContactChannelRef.Name
			}
			log.FromContext(notifyCtx).Error(err, "Failed to send final result via HumanLayer API",
				"taskName", task.Name,
				"contactChannel", contactChannelName)
		}
	}()
}

func (sm *StateMachine) sendFinalResultViaHumanLayerAPI(ctx context.Context, task *acp.Task, result string) error {
	logger := log.FromContext(ctx)

	if task.Spec.ContactChannelRef == nil {
		logger.Info("Skipping result notification, ContactChannelRef not set")
		return nil
	}

	// Get the ContactChannel
	var contactChannel acp.ContactChannel
	if err := sm.client.Get(ctx, client.ObjectKey{
		Namespace: task.Namespace,
		Name:      task.Spec.ContactChannelRef.Name,
	}, &contactChannel); err != nil {
		return fmt.Errorf("failed to get ContactChannel: %w", err)
	}

	// Get the API key from the ContactChannel's secret
	var secret corev1.Secret
	if err := sm.client.Get(ctx, client.ObjectKey{
		Namespace: task.Namespace,
		Name:      contactChannel.Spec.APIKeyFrom.SecretKeyRef.Name,
	}, &secret); err != nil {
		return fmt.Errorf("failed to get ContactChannel API key secret: %w", err)
	}

	apiKey := string(secret.Data[contactChannel.Spec.APIKeyFrom.SecretKeyRef.Key])
	if apiKey == "" {
		return fmt.Errorf("API key is empty in ContactChannel secret")
	}

	// Create HumanLayer client - use a hardcoded URL for now (need to determine baseURL source)
	client, err := sm.humanLayerFactory.NewClient("https://api.humanlayer.dev")
	if err != nil {
		return fmt.Errorf("failed to create HumanLayer client: %w", err)
	}
	client.SetAPIKey(apiKey)                 // Use API key from ContactChannel secret
	client.SetRunID(task.Spec.AgentRef.Name) // Use agent name as runID

	// Generate a random callID
	callID, err := validation.GenerateK8sRandomString(7)
	if err != nil {
		return fmt.Errorf("failed to generate callID: %w", err)
	}
	client.SetCallID(callID)

	// Retry up to 3 times
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Send the request to HumanLayer API
		humanContact, statusCode, err := client.RequestHumanContact(ctx, result)

		// Check for success
		if err == nil && statusCode >= 200 && statusCode < 300 {
			logger.Info("Successfully sent final result via HumanLayer API",
				"contactChannel", task.Spec.ContactChannelRef.Name,
				"statusCode", statusCode,
				"humanContactID", humanContact.GetCallId())
			return nil
		}

		// Log the error
		if err != nil {
			logger.Error(err, "Failed to send human contact request",
				"attempt", attempt+1,
				"contactChannel", task.Spec.ContactChannelRef.Name)
		} else {
			logger.Error(fmt.Errorf("HTTP error %d", statusCode),
				"Failed to send human contact request",
				"attempt", attempt+1,
				"contactChannel", task.Spec.ContactChannelRef.Name)
		}

		// Exponential backoff
		if attempt < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(1<<attempt)) // 1s, 2s, 4s
		}
	}

	return fmt.Errorf("failed to send human contact request after %d attempts", maxRetries)
}

// getTaskMutex returns or creates a mutex for a specific task
func (sm *StateMachine) getTaskMutex(taskName string) *sync.Mutex {
	sm.mutexMapLock.RLock()
	mutex, exists := sm.taskMutexes[taskName]
	sm.mutexMapLock.RUnlock()

	if exists {
		return mutex
	}

	// Need to create a new mutex
	sm.mutexMapLock.Lock()
	defer sm.mutexMapLock.Unlock()

	// Double-check after acquiring write lock
	if mutex, exists := sm.taskMutexes[taskName]; exists {
		return mutex
	}

	mutex = &sync.Mutex{}
	sm.taskMutexes[taskName] = mutex
	return mutex
}

// handleV1Beta3FinalAnswer handles final answers for v1beta3 tasks by creating a respond_to_human tool call
func (sm *StateMachine) handleV1Beta3FinalAnswer(ctx context.Context, output *acp.Message, task *acp.Task, statusUpdate *acp.Task, _ []llmclient.Tool) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling v1beta3 final answer by creating respond_to_human tool call")

	// Generate a unique ID for this tool call using k8s-style random strings
	toolCallRequestId, err := validation.GenerateK8sRandomString(7)
	if err != nil {
		logger.Error(err, "Failed to generate toolCallRequestId")
		return ctrl.Result{}, err
	}
	toolCallID, err := validation.GenerateK8sRandomString(8)
	if err != nil {
		logger.Error(err, "Failed to generate toolCallID")
		return ctrl.Result{}, err
	}

	// Create a respond_to_human tool call instead of final answer
	respondToHumanCall := acp.MessageToolCall{
		ID: toolCallID,
		Function: acp.ToolCallFunction{
			Name:      "respond_to_human",
			Arguments: fmt.Sprintf(`{"content": "%s"}`, output.Content),
		},
		Type: "function",
	}

	// Set status to tool calls pending instead of final answer
	statusUpdate.Status.Output = ""
	statusUpdate.Status.Phase = acp.TaskPhaseToolCallsPending
	statusUpdate.Status.ToolCallRequestID = toolCallRequestId
	statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, acp.Message{
		Role:      "assistant",
		ToolCalls: []acp.MessageToolCall{respondToHumanCall},
	})
	statusUpdate.Status.Ready = true
	statusUpdate.Status.Status = acp.TaskStatusTypeReady
	statusUpdate.Status.StatusDetail = "Creating respond_to_human tool call for v1beta3 final answer"
	statusUpdate.Status.Error = ""
	sm.recorder.Event(task, corev1.EventTypeNormal, "V1Beta3RespondToHuman", "Creating respond_to_human tool call for final answer")

	// Update the status before creating tool call
	if err := sm.client.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update Task status for v1beta3 respond_to_human")
		return ctrl.Result{}, err
	}

	// Create the respond_to_human ToolCall resource
	return sm.createV1Beta3ToolCall(ctx, task, statusUpdate, respondToHumanCall)
}

// createV1Beta3ToolCall creates a special respond_to_human ToolCall for v1beta3 tasks
func (sm *StateMachine) createV1Beta3ToolCall(ctx context.Context, task *acp.Task, statusUpdate *acp.Task, toolCall acp.MessageToolCall) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	newName := fmt.Sprintf("%s-%s-respond-to-human", statusUpdate.Name, statusUpdate.Status.ToolCallRequestID)

	newTC := &acp.ToolCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newName,
			Namespace: statusUpdate.Namespace,
			Labels: map[string]string{
				"acp.humanlayer.dev/task":            statusUpdate.Name,
				"acp.humanlayer.dev/toolcallrequest": statusUpdate.Status.ToolCallRequestID,
				"acp.humanlayer.dev/v1beta3":         "true",
				"acp.humanlayer.dev/tool-type":       "respond_to_human",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "acp.humanlayer.dev/v1alpha1",
					Kind:       "Task",
					Name:       statusUpdate.Name,
					UID:        statusUpdate.UID,
					Controller: ptr.To(true),
				},
			},
		},
		Spec: acp.ToolCallSpec{
			ToolCallID: toolCall.ID,
			TaskRef: acp.LocalObjectReference{
				Name: statusUpdate.Name,
			},
			ToolRef: acp.LocalObjectReference{
				Name: "respond_to_human",
			},
			ToolType:  acp.ToolTypeHumanContact,
			Arguments: toolCall.Function.Arguments,
		},
	}

	if err := sm.client.Create(ctx, newTC); err != nil {
		logger.Error(err, "Failed to create respond_to_human ToolCall", "name", newName)
		return ctrl.Result{}, err
	}

	logger.Info("Created respond_to_human ToolCall for v1beta3 task", "name", newName, "requestId", statusUpdate.Status.ToolCallRequestID)
	sm.recorder.Event(task, corev1.EventTypeNormal, "V1Beta3ToolCallCreated", "Created respond_to_human ToolCall "+newName)

	return ctrl.Result{RequeueAfter: DefaultRequeueDelay}, nil
}

// acquireTaskLease attempts to acquire a distributed lease for a task
func (sm *StateMachine) acquireTaskLease(ctx context.Context, taskName string) (*coordinationv1.Lease, bool, error) {
	leaseName := "task-llm-" + taskName
	now := metav1.NewMicroTime(time.Now())

	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: sm.namespace,
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       &sm.podName,
			LeaseDurationSeconds: ptr.To(int32(sm.leaseDuration.Seconds())),
			AcquireTime:          &now,
			RenewTime:            &now,
		},
	}

	// Try to create the lease
	err := sm.client.Create(ctx, lease)
	if err == nil {
		return lease, true, nil
	}

	// If lease already exists, try to acquire it if expired
	if apierrors.IsAlreadyExists(err) {
		existingLease := &coordinationv1.Lease{}
		if err := sm.client.Get(ctx, client.ObjectKey{
			Namespace: sm.namespace,
			Name:      leaseName,
		}, existingLease); err != nil {
			return nil, false, err
		}

		// Check if lease is expired or we already hold it
		if sm.canAcquireLease(existingLease) {
			existingLease.Spec.HolderIdentity = &sm.podName
			existingLease.Spec.AcquireTime = &now
			existingLease.Spec.RenewTime = &now

			if err := sm.client.Update(ctx, existingLease); err != nil {
				return nil, false, err
			}
			return existingLease, true, nil
		}

		return nil, false, nil // Lease held by another pod
	}

	return nil, false, err
}

// canAcquireLease checks if we can acquire the lease (expired or we already hold it)
func (sm *StateMachine) canAcquireLease(lease *coordinationv1.Lease) bool {
	if lease.Spec.HolderIdentity != nil && *lease.Spec.HolderIdentity == sm.podName {
		return true // We already hold it
	}

	if lease.Spec.RenewTime == nil {
		return true // No renew time, assume expired
	}

	expireTime := lease.Spec.RenewTime.Add(sm.leaseDuration)
	return time.Now().After(expireTime)
}

// releaseTaskLease releases a distributed lease
func (sm *StateMachine) releaseTaskLease(ctx context.Context, lease *coordinationv1.Lease) {
	if lease == nil {
		return
	}

	// Delete the lease to release it
	if err := sm.client.Delete(ctx, lease); err != nil {
		// Log but don't fail - lease will expire naturally
		log.FromContext(ctx).V(1).Info("Failed to delete task lease", "lease", lease.Name, "error", err)
	}
}
