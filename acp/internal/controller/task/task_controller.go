package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/humanlayer/agentcontrolplane/acp/internal/adapters"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=tasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=toolcalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=llms,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// TaskReconciler reconciles a Task object
type TaskReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	recorder     record.EventRecorder
	newLLMClient func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error)
	MCPManager   *mcpmanager.MCPServerManager
	Tracer       trace.Tracer
}

// initializePhaseAndSpan initializes the phase and span context for a new Task
func (r *TaskReconciler) initializePhaseAndSpan(ctx context.Context, task *acp.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Create a new *root* span for the Task
	spanCtx, span := r.Tracer.Start(ctx, "Task",
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

	if err := r.Status().Update(spanCtx, task); err != nil {
		logger.Error(err, "Failed to update Task status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// validateTaskAndAgent checks if the agent exists and is ready
func (r *TaskReconciler) attachRootSpan(ctx context.Context, task *acp.Task) context.Context {
	if task.Status.SpanContext == nil || task.Status.SpanContext.TraceID == "" || task.Status.SpanContext.SpanID == "" {
		return ctx // no root yet or invalid context
	}

	traceID, err := trace.TraceIDFromHex(task.Status.SpanContext.TraceID)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to parse TraceID from Task status", "traceID", task.Status.SpanContext.TraceID)
		return ctx
	}
	spanID, err := trace.SpanIDFromHex(task.Status.SpanContext.SpanID)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to parse SpanID from Task status", "spanID", task.Status.SpanContext.SpanID)
		return ctx
	}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})

	if !sc.IsValid() {
		log.FromContext(ctx).Error(fmt.Errorf("reconstructed SpanContext is invalid"), "traceID", traceID, "spanID", spanID)
		return ctx
	}

	return trace.ContextWithSpanContext(ctx, sc)
}

func (r *TaskReconciler) validateTaskAndAgent(ctx context.Context, task *acp.Task, statusUpdate *acp.Task) (*acp.Agent, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var agent acp.Agent
	if err := r.Get(ctx, client.ObjectKey{Namespace: task.Namespace, Name: task.Spec.AgentRef.Name}, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Agent not found, waiting for it to exist")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = acp.TaskStatusTypePending
			statusUpdate.Status.Phase = acp.TaskPhasePending
			statusUpdate.Status.StatusDetail = "Waiting for Agent to exist"
			statusUpdate.Status.Error = "" // Clear previous error
			r.recorder.Event(task, corev1.EventTypeNormal, "Waiting", "Waiting for Agent to exist")
		} else {
			logger.Error(err, "Failed to get Agent")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = acp.TaskStatusTypeError
			statusUpdate.Status.Phase = acp.TaskPhaseFailed
			statusUpdate.Status.Error = err.Error()
			r.recorder.Event(task, corev1.EventTypeWarning, "AgentFetchFailed", err.Error())
		}
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return nil, ctrl.Result{}, updateErr
		}
		return nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Check if agent is ready
	if !agent.Status.Ready {
		logger.Info("Agent exists but is not ready", "agent", agent.Name)
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = acp.TaskStatusTypePending
		statusUpdate.Status.Phase = acp.TaskPhasePending
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Waiting for agent %q to become ready", agent.Name)
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(task, corev1.EventTypeNormal, "Waiting", fmt.Sprintf("Waiting for agent %q to become ready", agent.Name))
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update Task status")
			return nil, ctrl.Result{}, err
		}
		return nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	return &agent, ctrl.Result{}, nil
}

// prepareForLLM sets up the initial state of a Task with the correct context and phase
func (r *TaskReconciler) prepareForLLM(ctx context.Context, task *acp.Task, statusUpdate *acp.Task, agent *acp.Agent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// If we're in Initializing or Pending phase, transition to ReadyForLLM
	if statusUpdate.Status.Phase == acp.TaskPhaseInitializing || statusUpdate.Status.Phase == acp.TaskPhasePending {
		statusUpdate.Status.Phase = acp.TaskPhaseReadyForLLM
		statusUpdate.Status.Ready = true

		if task.Spec.UserMessage == "" {
			err := fmt.Errorf("userMessage is required")
			logger.Error(err, "Missing message")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = acp.TaskStatusTypeError
			statusUpdate.Status.Phase = acp.TaskPhaseFailed
			statusUpdate.Status.StatusDetail = err.Error()
			statusUpdate.Status.Error = err.Error()
			r.recorder.Event(task, corev1.EventTypeWarning, "ValidationFailed", err.Error())
			if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
				logger.Error(updateErr, "Failed to update Task status")
				return ctrl.Result{}, updateErr
			}
			return ctrl.Result{}, err
		}

		// Set the UserMsgPreview - truncate to 50 chars if needed
		preview := task.Spec.UserMessage
		if len(preview) > 50 {
			preview = preview[:47] + "..."
		}
		statusUpdate.Status.UserMsgPreview = preview

		// Set up the context window
		statusUpdate.Status.ContextWindow = []acp.Message{
			{
				Role:    "system",
				Content: agent.Spec.System,
			},
			{
				Role:    "user",
				Content: task.Spec.UserMessage,
			},
		}
		statusUpdate.Status.Status = acp.TaskStatusTypeReady
		statusUpdate.Status.StatusDetail = "Ready to send to LLM"
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(task, corev1.EventTypeNormal, "ValidationSucceeded", "Task validation succeeded")

		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update Task status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// processToolCalls handles the tool calls phase
func (r *TaskReconciler) processToolCalls(ctx context.Context, task *acp.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// List all tool calls for this Task
	toolCalls := &acp.ToolCallList{}
	if err := r.List(ctx, toolCalls, client.InNamespace(task.Namespace), client.MatchingLabels{
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
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
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
	r.recorder.Event(task, corev1.EventTypeNormal, "AllToolCallsCompleted", "All tool calls completed")

	if err := r.Status().Update(ctx, task); err != nil {
		logger.Error(err, "Failed to update Task status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// getLLMAndCredentials gets the LLM and API key for the agent
func (r *TaskReconciler) getLLMAndCredentials(ctx context.Context, agent *acp.Agent, task *acp.Task, statusUpdate *acp.Task) (acp.LLM, string, error) {
	logger := log.FromContext(ctx)

	// Get the LLM
	var llm acp.LLM
	if err := r.Get(ctx, client.ObjectKey{Namespace: task.Namespace, Name: agent.Spec.LLMRef.Name}, &llm); err != nil {
		logger.Error(err, "Failed to get LLM")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = acp.TaskStatusTypeError
		statusUpdate.Status.Phase = acp.TaskPhaseFailed
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Failed to get LLM: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(task, corev1.EventTypeWarning, "LLMFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	// Get the API key from the secret
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: task.Namespace,
		Name:      llm.Spec.APIKeyFrom.SecretKeyRef.Name,
	}, &secret); err != nil {
		logger.Error(err, "Failed to get API key secret")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = acp.TaskStatusTypeError
		statusUpdate.Status.Phase = acp.TaskPhaseFailed
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Failed to get API key secret: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(task, corev1.EventTypeWarning, "APIKeySecretFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
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
		r.recorder.Event(task, corev1.EventTypeWarning, "EmptyAPIKey", "API key is empty")
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	return llm, apiKey, nil
}

// endTaskSpan ends the Task span with the given status
func (r *TaskReconciler) endTaskTrace(ctx context.Context, task *acp.Task, code codes.Code, message string) {
	logger := log.FromContext(ctx)
	if task.Status.SpanContext == nil {
		logger.Info("No span context found in task status, cannot end trace")
		return
	}

	// Reattach the parent's context again to ensure the final span is correctly parented.
	ctx = r.attachRootSpan(ctx, task)

	// Now create a final child span to mark "root" completion.
	_, span := r.Tracer.Start(ctx, "EndTaskSpan")
	defer span.End() // End this specific child span immediately.

	span.SetStatus(code, message)
	// Add any last attributes if needed
	span.SetAttributes(attribute.String("task.name", task.Name))

	logger.Info("Ended task trace with a final child span", "taskName", task.Name, "status", code.String())

	// Optionally clear the SpanContext from the resource status if you don't want subsequent
	// reconciles (e.g., for cleanup) to re-attach to the same trace.
	// task.Status.SpanContext = nil
	// if err := r.Status().Update(context.Background(), task); err != nil { // Use a background context for this update?
	// 	logger.Error(err, "Failed to clear SpanContext after ending trace")
	// }
}

// collectTools collects all tools from the agent's MCP servers
func (r *TaskReconciler) collectTools(ctx context.Context, agent *acp.Agent) []llmclient.Tool {
	logger := log.FromContext(ctx)
	tools := make([]llmclient.Tool, 0)

	// Get tools from MCP manager
	mcpTools := r.MCPManager.GetToolsForAgent(agent)

	// Convert MCP tools to LLM tools
	for _, mcpTool := range mcpTools {
		tools = append(tools, adapters.ConvertMCPToolsToLLMClientTools([]acp.MCPTool{mcpTool}, mcpTool.Name)...)
	}

	// Convert HumanContactChannel tools to LLM tools
	for _, validChannel := range agent.Status.ValidHumanContactChannels {
		channel := &acp.ContactChannel{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: validChannel.Name}, channel); err != nil {
			logger.Error(err, "Failed to get ContactChannel", "name", validChannel.Name)
			continue
		}

		// Convert to LLM client format
		clientTool := llmclient.ToolFromContactChannel(*channel)
		tools = append(tools, *clientTool)
		logger.Info("Added human contact channel tool", "name", channel.Name, "type", channel.Spec.Type)
	}

	// Add delegate tools for sub-agents
	for _, subAgentRef := range agent.Spec.SubAgents {
		subAgent := &acp.Agent{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: subAgentRef.Name}, subAgent); err != nil {
			logger.Error(err, "Failed to get sub-agent", "name", subAgentRef.Name)
			continue
		}

		// Create a delegate tool for the sub-agent
		delegateTool := llmclient.Tool{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        "delegate_to_agent__" + subAgent.Name,
				Description: subAgent.Spec.Description,
				Parameters: llmclient.ToolFunctionParameters{
					Type: "object",
					Properties: map[string]llmclient.ToolFunctionParameter{
						"message": {Type: "string"},
					},
					Required: []string{"message"},
				},
			},
			ACPToolType: acp.ToolTypeDelegateToAgent,
		}
		tools = append(tools, delegateTool)
		logger.Info("Added delegate tool for sub-agent", "name", subAgent.Name)
	}

	return tools
}

// createLLMRequestSpan creates a child span for the LLM request
func (r *TaskReconciler) createLLMRequestSpan(
	ctx context.Context, // This context should already have the root span attached via attachRootSpan
	task *acp.Task,
	numMessages int,
	numTools int,
) (context.Context, trace.Span) {
	// Now that ctx has the *root* span in it (from attachRootSpan), we can start a child:
	childCtx, childSpan := r.Tracer.Start(ctx, "LLMRequest",
		trace.WithSpanKind(trace.SpanKindClient), // Mark as client span for LLM call
	)

	childSpan.SetAttributes(
		attribute.Int("acp.task.context_window.messages", numMessages),
		attribute.Int("acp.task.tools.count", numTools),
		attribute.String("acp.task.name", task.Name), // Add task name for context
	)

	return childCtx, childSpan
}

// processLLMResponse processes the LLM response and updates the Task status
func (r *TaskReconciler) processLLMResponse(ctx context.Context, output *acp.Message, task *acp.Task, statusUpdate *acp.Task, tools []llmclient.Tool) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if output.Content != "" {
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
		r.recorder.Event(task, corev1.EventTypeNormal, "LLMFinalAnswer", "LLM response received successfully")

		// End the task trace with OK status since we have a final answer.
		// The context passed here should ideally be the one from Reconcile after attachRootSpan.
		// r.endTaskTrace(ctx, task, codes.Ok, "Task completed successfully with final answer")
		// NOTE: The plan suggests calling endTaskTrace from Reconcile when phase is FinalAnswer,
		// so we might not need to call it here. Let's follow the plan's structure.
	} else {
		// Generate a unique ID for this set of tool calls
		toolCallRequestId := uuid.New().String()[:7] // Using first 7 characters for brevity
		logger.Info("Generated toolCallRequestId for tool calls", "id", toolCallRequestId)

		// tool call branch: create ToolCall objects for each tool call returned by the LLM.
		statusUpdate.Status.Output = ""
		statusUpdate.Status.Phase = acp.TaskPhaseToolCallsPending
		statusUpdate.Status.ToolCallRequestID = toolCallRequestId
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, acp.Message{
			Role:      "assistant",
			ToolCalls: adapters.CastOpenAIToolCallsToACP(output.ToolCalls),
		})
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = acp.TaskStatusTypeReady
		statusUpdate.Status.StatusDetail = "LLM response received, tool calls pending"
		statusUpdate.Status.Error = ""
		r.recorder.Event(task, corev1.EventTypeNormal, "ToolCallsPending", "LLM response received, tool calls pending")

		// Update the parent's status before creating tool call objects.
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Unable to update Task status")
			return ctrl.Result{}, err
		}

		return r.createToolCalls(ctx, task, statusUpdate, output.ToolCalls, tools)
	}
	return ctrl.Result{}, nil
}

// createToolCalls creates ToolCall objects for each tool call
func (r *TaskReconciler) createToolCalls(ctx context.Context, task *acp.Task, statusUpdate *acp.Task, toolCalls []acp.MessageToolCall, tools []llmclient.Tool) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if statusUpdate.Status.ToolCallRequestID == "" {
		err := fmt.Errorf("no ToolCallRequestID found in statusUpdate, cannot create tool calls")
		logger.Error(err, "Missing ToolCallRequestID")
		return ctrl.Result{}, err
	}

	// Create a map of tool name to tool type for quick lookup
	toolTypeMap := make(map[string]acp.ToolType)
	for _, tool := range tools {
		toolTypeMap[tool.Function.Name] = tool.ACPToolType
	}

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
		if err := r.Client.Create(ctx, newTC); err != nil {
			logger.Error(err, "Failed to create ToolCall", "name", newName)
			return ctrl.Result{}, err
		}
		logger.Info("Created ToolCall", "name", newName, "requestId", statusUpdate.Status.ToolCallRequestID, "toolType", toolType)
		r.recorder.Event(task, corev1.EventTypeNormal, "ToolCallCreated", "Created ToolCall "+newName)
	}
	return ctrl.Result{RequeueAfter: time.Second * 5}, nil
}

// Reconcile validates the task's agent reference and sends the prompt to the LLM.
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var task acp.Task
	if err := r.Get(ctx, req.NamespacedName, &task); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", task.Name)

	// Create a copy for status update
	statusUpdate := task.DeepCopy()

	// Initialize phase and root span if not set
	if statusUpdate.Status.Phase == "" || statusUpdate.Status.SpanContext == nil {
		// If phase is empty OR span context is missing, initialize.
		logger.Info("Initializing phase and span context", "name", task.Name)
		return r.initializePhaseAndSpan(ctx, statusUpdate)
	}

	// For all subsequent reconciles, reattach the root span context
	ctx = r.attachRootSpan(ctx, &task)

	// reconcileCtx, reconcileSpan := r.createReconcileSpan(ctx, &task)
	// if reconcileSpan != nil {
	// 	defer reconcileSpan.End()
	// }

	// Skip reconciliation for terminal states, but end the trace if needed
	if statusUpdate.Status.Phase == acp.TaskPhaseFinalAnswer {
		logger.V(1).Info("Task in FinalAnswer state, ensuring trace is ended", "name", task.Name)
		// Call endTaskTrace here as per the plan
		r.endTaskTrace(ctx, statusUpdate, codes.Ok, "Task completed successfully with final answer")
		return ctrl.Result{}, nil // No further action needed
	}
	if statusUpdate.Status.Phase == acp.TaskPhaseFailed {
		logger.V(1).Info("Task in Failed state, ensuring trace is ended", "name", task.Name)
		// End trace with error status
		errMsg := "Task failed"
		if statusUpdate.Status.Error != "" {
			errMsg = statusUpdate.Status.Error
		}
		r.endTaskTrace(ctx, statusUpdate, codes.Error, errMsg)
		return ctrl.Result{}, nil // No further action needed
	}

	// Step 1: Validate Agent
	logger.V(3).Info("Validating Agent")
	agent, result, err := r.validateTaskAndAgent(ctx, &task, statusUpdate)
	if err != nil || !result.IsZero() {
		return result, err
	}

	// Step 2: Initialize Phase if necessary
	logger.V(3).Info("Preparing for LLM")
	if result, err := r.prepareForLLM(ctx, &task, statusUpdate, agent); err != nil || !result.IsZero() {
		return result, err
	}

	// Step 3: Handle tool calls phase
	logger.V(3).Info("Handling tool calls phase")
	if task.Status.Phase == acp.TaskPhaseToolCallsPending {
		return r.processToolCalls(ctx, &task)
	}

	// Step 4: Check for unexpected phase
	if task.Status.Phase != acp.TaskPhaseReadyForLLM {
		logger.Info("Task in unknown phase", "phase", task.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Step 5: Get API credentials (LLM is returned but not used)
	logger.V(3).Info("Getting API credentials")
	llm, apiKey, err := r.getLLMAndCredentials(ctx, agent, &task, statusUpdate)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Step 6: Create LLM client
	logger.V(3).Info("Creating LLM client")
	llmClient, err := r.newLLMClient(ctx, llm, apiKey)
	if err != nil {
		logger.Error(err, "Failed to create LLM client")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = acp.TaskStatusTypeError
		statusUpdate.Status.Phase = acp.TaskPhaseFailed
		statusUpdate.Status.StatusDetail = "Failed to create LLM client: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&task, corev1.EventTypeWarning, "LLMClientCreationFailed", err.Error())

		// End trace since we've failed with a terminal error
		r.endTaskTrace(ctx, statusUpdate, codes.Error, "Failed to create LLM client: "+err.Error())

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return ctrl.Result{}, updateErr
		}
		// Don't return the error itself, as status is updated and trace ended.
		return ctrl.Result{}, nil
	}

	// Step 7: Collect tools from all sources
	tools := r.collectTools(ctx, agent)

	r.recorder.Event(&task, corev1.EventTypeNormal, "SendingContextWindowToLLM", "Sending context window to LLM")

	// Create child span for LLM call
	llmCtx, llmSpan := r.createLLMRequestSpan(ctx, &task, len(task.Status.ContextWindow), len(tools))
	if llmSpan != nil {
		defer llmSpan.End()
	}

	logger.V(3).Info("Sending LLM request")
	// Step 8: Send the prompt to the LLM
	output, err := llmClient.SendRequest(llmCtx, task.Status.ContextWindow, tools)
	if err != nil {
		logger.Error(err, "LLM request failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = acp.TaskStatusTypeError
		statusUpdate.Status.StatusDetail = fmt.Sprintf("LLM request failed: %v", err)
		statusUpdate.Status.Error = err.Error()

		// Check for LLMRequestError with 4xx status code
		var llmErr *llmclient.LLMRequestError
		is4xxError := errors.As(err, &llmErr) && llmErr.StatusCode >= 400 && llmErr.StatusCode < 500

		if is4xxError {
			logger.Info("LLM request failed with 4xx status code, marking as failed",
				"statusCode", llmErr.StatusCode,
				"message", llmErr.Message)
			statusUpdate.Status.Phase = acp.TaskPhaseFailed // Set phase to Failed for 4xx
			r.recorder.Event(&task, corev1.EventTypeWarning, "LLMRequestFailed4xx",
				fmt.Sprintf("LLM request failed with status %d: %s", llmErr.StatusCode, llmErr.Message))
		} else {
			// For non-4xx errors, just record the event, phase remains ReadyForLLM (or current)
			r.recorder.Event(&task, corev1.EventTypeWarning, "LLMRequestFailed", err.Error())
		}

		// Record error in span
		if llmSpan != nil {
			llmSpan.RecordError(err)
			llmSpan.SetStatus(codes.Error, err.Error())
		}

		// Attempt to update the status
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status after LLM error")
			// If status update fails, return that error
			return ctrl.Result{}, updateErr
		}

		// If it was a 4xx error and status update succeeded, return nil error (terminal state)
		if is4xxError {
			return ctrl.Result{}, nil
		}

		// Otherwise (non-4xx error), return the original LLM error to trigger requeue/backoff
		return ctrl.Result{}, err
	}

	// Mark span as successful and add attributes
	if llmSpan != nil {
		llmSpan.SetStatus(codes.Ok, "LLM request succeeded")
		// Add attributes based on the request and response
		llmSpan.SetAttributes(
			attribute.String("llm.request.model", llm.Spec.Parameters.Model),
			attribute.Int("llm.response.tool_calls.count", len(output.ToolCalls)),
			attribute.Bool("llm.response.has_content", output.Content != ""),
		)
		llmSpan.End()
	}

	logger.V(3).Info("Processing LLM response")
	// Step 9: Process LLM response
	var llmResult ctrl.Result
	llmResult, err = r.processLLMResponse(ctx, output, &task, statusUpdate, tools)
	if err != nil {
		logger.Error(err, "Failed to process LLM response")
		statusUpdate.Status.Status = acp.TaskStatusTypeError
		statusUpdate.Status.Phase = acp.TaskPhaseFailed
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Failed to process LLM response: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&task, corev1.EventTypeWarning, "LLMResponseProcessingFailed", err.Error())

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status after LLM response processing error")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil // Don't return the error to avoid requeuing
	}

	if !llmResult.IsZero() {
		return llmResult, nil
	}

	// Step 10: Update final status
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update Task status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled task",
		"name", task.Name,
		"ready", statusUpdate.Status.Ready,
		"phase", statusUpdate.Status.Phase)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("task-controller")
	if r.newLLMClient == nil {
		r.newLLMClient = llmclient.NewLLMClient
	}

	// Initialize MCPManager if not already set
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManager()
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.Task{}).
		Complete(r)
}
