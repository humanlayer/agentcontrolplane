package taskrun

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"

	"github.com/humanlayer/smallchain/kubechain/internal/adapters"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
	"github.com/humanlayer/smallchain/kubechain/internal/mcpmanager"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// These constants are kept for backward compatibility during refactoring
	StatusReady   = kubechainv1alpha1.TaskRunStatusStatusReady
	StatusError   = kubechainv1alpha1.TaskRunStatusStatusError
	StatusPending = kubechainv1alpha1.TaskRunStatusStatusPending
)

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=taskruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=taskruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=tasks,verbs=get;list;watch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=llms,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// TaskRunReconciler reconciles a TaskRun object
type TaskRunReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	recorder     record.EventRecorder
	newLLMClient func(apiKey string) (llmclient.OpenAIClient, error)
	MCPManager   *mcpmanager.MCPServerManager
	Tracer       trace.Tracer
}

// getTask fetches the parent Task for this TaskRun
func (r *TaskRunReconciler) getTask(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun) (*kubechainv1alpha1.Task, error) {
	// If we have agentRef and userMessage, we don't need a task
	if taskRun.Spec.AgentRef != nil && taskRun.Spec.UserMessage != "" {
		return nil, nil
	}

	// If TaskRef is nil, we can't fetch the task
	if taskRun.Spec.TaskRef == nil {
		return nil, fmt.Errorf("taskRef is required when agentRef and userMessage are not provided")
	}

	task := &kubechainv1alpha1.Task{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: taskRun.Namespace,
		Name:      taskRun.Spec.TaskRef.Name,
	}, task)
	if err != nil {
		return nil, fmt.Errorf("failed to get Task %q: %w", taskRun.Spec.TaskRef.Name, err)
	}

	if !task.Status.Ready {
		return task, nil // Return task but indicate it's not ready
	}

	return task, nil
}

// validateTaskAndAgent checks if the task and agent exist and are ready
func (r *TaskRunReconciler) validateTaskAndAgent(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun) (*kubechainv1alpha1.Task, *kubechainv1alpha1.Agent, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Validate that we have either taskRef or agentRef+userMessage
	if taskRun.Spec.TaskRef == nil && (taskRun.Spec.AgentRef == nil || taskRun.Spec.UserMessage == "") {
		err := fmt.Errorf("must specify either taskRef or both agentRef and userMessage")
		logger.Error(err, "Validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.StatusDetail = err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "ValidationFailed", err.Error())

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, fmt.Errorf("failed to update taskrun status: %v", err)
		}
		return nil, nil, ctrl.Result{}, err
	}

	// Get parent Task if needed
	task, err := r.getTask(ctx, taskRun)
	if err != nil {
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "TaskValidationFailed", err.Error())
		if apierrors.IsNotFound(err) {
			logger.Info("Task not found", "task", taskRun.Spec.TaskRef.Name)
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
			statusUpdate.Status.Error = fmt.Sprintf("Task %q not found", taskRun.Spec.TaskRef.Name)
			statusUpdate.Status.StatusDetail = fmt.Sprintf("Task %q not found", taskRun.Spec.TaskRef.Name)
			statusUpdate.Status.Status = StatusError

			// End span since we've failed with a terminal error
			r.endTaskRunSpan(ctx, taskRun, codes.Error, fmt.Sprintf("Task %q not found", taskRun.Spec.TaskRef.Name))
		} else {
			logger.Error(err, "Task validation failed")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = StatusError
			statusUpdate.Status.StatusDetail = fmt.Sprintf("Task validation failed: %v", err)
			statusUpdate.Status.Error = err.Error()

			// End span since we've failed with a terminal error
			r.endTaskRunSpan(ctx, taskRun, codes.Error, fmt.Sprintf("Task validation failed: %v", err))
		}

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, fmt.Errorf("failed to update taskrun status: %v", err)
		}
		return nil, nil, ctrl.Result{}, err
	}

	// Check if task exists but is not ready
	if task != nil && !task.Status.Ready {
		logger.Info("Task exists but is not ready", "task", task.Name)
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusPending
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Waiting for task %q to become ready", task.Name)
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "TaskNotReady", fmt.Sprintf("Waiting for task %q to become ready", task.Name))
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, err
		}
		return nil, nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Get the Agent - use TaskRun's AgentRef if provided, otherwise use Task's AgentRef
	var agentRef kubechainv1alpha1.LocalObjectReference
	if taskRun.Spec.AgentRef != nil {
		agentRef = *taskRun.Spec.AgentRef
	} else if task != nil {
		agentRef = task.Spec.AgentRef
	} else {
		err := fmt.Errorf("no agent reference found")
		logger.Error(err, "Missing agent reference")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.StatusDetail = err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, updateErr
		}
		return nil, nil, ctrl.Result{}, err
	}

	var agent kubechainv1alpha1.Agent
	if err := r.Get(ctx, client.ObjectKey{Namespace: taskRun.Namespace, Name: agentRef.Name}, &agent); err != nil {
		logger.Error(err, "Failed to get Agent")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusPending
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.StatusDetail = "Waiting for Agent to exist"
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "Waiting", "Waiting for Agent to exist")
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, updateErr
		}
		return nil, nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Check if agent is ready
	if !agent.Status.Ready {
		logger.Info("Agent exists but is not ready", "agent", agent.Name)
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusPending
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Waiting for agent %q to become ready", agent.Name)
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "Waiting", fmt.Sprintf("Waiting for agent %q to become ready", agent.Name))
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, err
		}
		return nil, nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	return task, &agent, ctrl.Result{}, nil
}

// prepareForLLM sets up the initial state of a TaskRun with the correct context and phase
func (r *TaskRunReconciler) prepareForLLM(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun, task *kubechainv1alpha1.Task, agent *kubechainv1alpha1.Agent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseInitializing || statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhasePending {
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
		statusUpdate.Status.Ready = true

		// Use userMessage from TaskRun if provided, otherwise use task's message
		message := taskRun.Spec.UserMessage
		if message == "" && task != nil {
			message = task.Spec.Message
		}

		statusUpdate.Status.ContextWindow = []kubechainv1alpha1.Message{
			{
				Role:    "system",
				Content: agent.Spec.System,
			},
			{
				Role:    "user",
				Content: message,
			},
		}
		statusUpdate.Status.Status = StatusReady
		statusUpdate.Status.StatusDetail = "Ready to send to LLM"
		statusUpdate.Status.Error = "" // Clear any previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "ValidationSucceeded", "Task validated successfully")
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// processToolCalls handles the ToolCallsPending phase by checking tool call completion
func (r *TaskRunReconciler) processToolCalls(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// List all tool calls for this ToolCallRequestID
	toolCallList := &kubechainv1alpha1.TaskRunToolCallList{}
	logger.Info("Listing tool calls", "taskrun", taskRun.Name, "requestId", taskRun.Status.ToolCallRequestID)

	if err := r.List(ctx, toolCallList, client.InNamespace(taskRun.Namespace),
		client.MatchingLabels{"kubechain.humanlayer.dev/toolcallrequest": taskRun.Status.ToolCallRequestID}); err != nil {
		logger.Error(err, "Failed to list tool calls")
		return ctrl.Result{}, err
	}

	logger.Info("Found tool calls", "count", len(toolCallList.Items))
	if len(toolCallList.Items) == 0 {
		logger.Info("No tool calls found, something is very wrong")
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "ToolCallsPendingWithNoToolCalls", "TaskRun is in ToolCallsPending phase but no tool calls were found")
		return ctrl.Result{}, fmt.Errorf("TaskRun %q is in ToolCallsPending phase but no tool calls were found", taskRun.Name)
	}

	// Check if all tool calls are complete
	allComplete := true
	toolResults := make([]kubechainv1alpha1.Message, 0, len(toolCallList.Items))

	for _, tc := range toolCallList.Items {
		logger.Info("Checking tool call", "name", tc.Name, "phase", tc.Status.Phase)
		if tc.Status.Phase != kubechainv1alpha1.TaskRunToolCallPhaseSucceeded {
			allComplete = false
			logger.Info("Found incomplete tool call", "name", tc.Name)
			break
		}
		toolResults = append(toolResults, kubechainv1alpha1.Message{
			ToolCallId: tc.Spec.ToolCallId,
			Role:       "tool",
			Content:    tc.Status.Result,
		})
	}

	if allComplete {
		logger.Info("All tool calls complete, transitioning to ReadyForLLM")
		// All tool calls are complete, update context window and move back to ReadyForLLM phase
		// so that the LLM can process the tool results and provide a final answer
		statusUpdate := taskRun.DeepCopy()
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, toolResults...)
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = StatusReady
		statusUpdate.Status.StatusDetail = "All tool calls completed, ready to send tool results to LLM"
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "AllToolCallsCompleted", "All tool calls completed, ready to send tool results to LLM")

		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Not all tool calls are complete, requeue while staying in ToolCallsPending phase
	return ctrl.Result{RequeueAfter: time.Second * 5}, nil
}

// getLLMAndCredentials fetches the LLM and its API key from the referenced secret
func (r *TaskRunReconciler) getLLMAndCredentials(ctx context.Context, agent *kubechainv1alpha1.Agent, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun) (kubechainv1alpha1.LLM, string, error) {
	logger := log.FromContext(ctx)

	// Get the LLM referenced by the Agent
	var llm kubechainv1alpha1.LLM
	if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: agent.Spec.LLMRef.Name}, &llm); err != nil {
		logger.Error(err, "Failed to get LLM")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = "Failed to get LLM: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "LLMFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	// Get the API key from the referenced secret
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: llm.Namespace,
		Name:      llm.Spec.APIKeyFrom.SecretKeyRef.Name,
	}, &secret); err != nil {
		logger.Error(err, "Failed to get API key secret")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = "Failed to get API key secret: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "SecretFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	// Validate API key
	apiKey := string(secret.Data[llm.Spec.APIKeyFrom.SecretKeyRef.Key])
	if apiKey == "" {
		err := fmt.Errorf("API key is empty in secret %s", secret.Name)
		logger.Error(err, "Empty API key")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = "API key is empty"
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "EmptyAPIKey", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	return llm, apiKey, nil
}

// collectTools gathers tools from all sources (Tool CRDs and MCP servers)
func (r *TaskRunReconciler) collectTools(ctx context.Context, agent *kubechainv1alpha1.Agent) []llmclient.Tool {
	logger := log.FromContext(ctx)
	var tools []llmclient.Tool

	// First, add tools from traditional Tool CRDs
	if len(agent.Status.ValidTools) > 0 {
		logger.Info("Adding traditional tools to LLM request", "toolCount", len(agent.Status.ValidTools))

		for _, validTool := range agent.Status.ValidTools {
			if validTool.Kind != "Tool" {
				continue
			}

			// Get the Tool resource
			tool := &kubechainv1alpha1.Tool{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: validTool.Name}, tool); err != nil {
				logger.Error(err, "Failed to get Tool", "name", validTool.Name)
				continue
			}

			// Convert to LLM client format
			if clientTool := llmclient.FromKubechainTool(*tool); clientTool != nil {
				tools = append(tools, *clientTool)
				logger.Info("Added traditional tool", "name", tool.Name)
			}
		}
	}

	// Then, add tools from MCP servers if available
	if r.MCPManager != nil && len(agent.Status.ValidMCPServers) > 0 {
		logger.Info("Adding MCP tools to LLM request", "mcpServerCount", len(agent.Status.ValidMCPServers))

		for _, mcpServer := range agent.Status.ValidMCPServers {
			// Get tools for this server
			mcpTools, exists := r.MCPManager.GetTools(mcpServer.Name)
			if !exists {
				logger.Error(fmt.Errorf("MCP server tools not found"), "Failed to get tools for MCP server", "server", mcpServer.Name)
				continue
			}

			// Convert MCP tools to LLM client format
			mcpClientTools := adapters.ConvertMCPToolsToLLMClientTools(mcpTools, mcpServer.Name)
			tools = append(tools, mcpClientTools...)

			logger.Info("Added MCP tools", "server", mcpServer.Name, "toolCount", len(mcpTools))
		}
	}

	return tools
}

// endTaskRunSpan ends the parent span for a TaskRun if it exists
func (r *TaskRunReconciler) endTaskRunSpan(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, status codes.Code, description string) {
	// Only try to end span if we have SpanContext info
	if taskRun.Status.SpanContext == nil || taskRun.Status.SpanContext.TraceID == "" {
		return
	}

	// Get tracer
	tracer := r.Tracer
	if tracer == nil {
		tracer = otel.GetTracerProvider().Tracer("taskrun")
	}

	// Parse the trace and span IDs from the stored context
	var traceID trace.TraceID
	var spanID trace.SpanID

	// Convert hex strings to byte arrays
	traceIDBytes, err := trace.TraceIDFromHex(taskRun.Status.SpanContext.TraceID)
	if err != nil {
		return
	}
	traceID = traceIDBytes

	spanIDBytes, err := trace.SpanIDFromHex(taskRun.Status.SpanContext.SpanID)
	if err != nil {
		return
	}
	spanID = spanIDBytes

	// Create a span context with the stored IDs
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     false,
	})

	// Create context with the span context
	ctxWithSpan := trace.ContextWithSpanContext(ctx, spanCtx)

	// Create a final completion span that's a child of the original span
	_, span := tracer.Start(ctxWithSpan, fmt.Sprintf("TaskRun/%s/Completion", taskRun.Name),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("taskrun.name", taskRun.Name),
			attribute.String("taskrun.namespace", taskRun.Namespace),
			attribute.String("taskrun.phase", string(taskRun.Status.Phase)),
			attribute.String("taskrun.status", string(taskRun.Status.Status)),
		),
	)
	span.SetStatus(status, description)
	span.End()
}

// processLLMResponse handles the LLM's output and updates status accordingly
func (r *TaskRunReconciler) processLLMResponse(ctx context.Context, output *kubechainv1alpha1.Message, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if output.Content != "" {
		// final answer branch
		statusUpdate.Status.Output = output.Content
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFinalAnswer
		statusUpdate.Status.Ready = true
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, kubechainv1alpha1.Message{
			Role:    "assistant",
			Content: output.Content,
		})
		statusUpdate.Status.Status = StatusReady
		statusUpdate.Status.StatusDetail = "LLM final response received"
		statusUpdate.Status.Error = ""
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "LLMFinalAnswer", "LLM response received successfully")

		// End the parent span since we've reached a terminal state
		r.endTaskRunSpan(ctx, taskRun, codes.Ok, "TaskRun completed successfully with final answer")
	} else {
		// Generate a unique ID for this set of tool calls
		toolCallRequestId := uuid.New().String()[:7] // Using first 7 characters for brevity
		logger.Info("Generated toolCallRequestId for tool calls", "id", toolCallRequestId)

		// tool call branch: create TaskRunToolCall objects for each tool call returned by the LLM.
		statusUpdate.Status.Output = ""
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseToolCallsPending
		statusUpdate.Status.ToolCallRequestID = toolCallRequestId
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, kubechainv1alpha1.Message{
			Role:      "assistant",
			ToolCalls: adapters.CastOpenAIToolCallsToKubechain(output.ToolCalls),
		})
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = StatusReady
		statusUpdate.Status.StatusDetail = "LLM response received, tool calls pending"
		statusUpdate.Status.Error = ""
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "ToolCallsPending", "LLM response received, tool calls pending")

		// Update the parent's status before creating tool call objects.
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Unable to update TaskRun status")
			return ctrl.Result{}, err
		}

		return r.createToolCalls(ctx, taskRun, statusUpdate, output.ToolCalls)
	}
	return ctrl.Result{}, nil
}

// createToolCalls creates TaskRunToolCall objects for each tool call
func (r *TaskRunReconciler) createToolCalls(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun, toolCalls []kubechainv1alpha1.ToolCall) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if statusUpdate.Status.ToolCallRequestID == "" {
		err := fmt.Errorf("no ToolCallRequestID found in statusUpdate, cannot create tool calls")
		logger.Error(err, "Missing ToolCallRequestID")
		return ctrl.Result{}, err
	}

	// For each tool call, create a new TaskRunToolCall with a unique name using the ToolCallRequestID
	for i, tc := range toolCalls {
		newName := fmt.Sprintf("%s-%s-tc-%02d", statusUpdate.Name, statusUpdate.Status.ToolCallRequestID, i+1)
		newTRTC := &kubechainv1alpha1.TaskRunToolCall{
			ObjectMeta: metav1.ObjectMeta{
				Name:      newName,
				Namespace: statusUpdate.Namespace,
				Labels: map[string]string{
					"kubechain.humanlayer.dev/taskruntoolcall": statusUpdate.Name,
					"kubechain.humanlayer.dev/toolcallrequest": statusUpdate.Status.ToolCallRequestID,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "kubechain.humanlayer.dev/v1alpha1",
						Kind:       "TaskRun",
						Name:       statusUpdate.Name,
						UID:        statusUpdate.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Spec: kubechainv1alpha1.TaskRunToolCallSpec{
				ToolCallId: tc.ID,
				TaskRunRef: kubechainv1alpha1.LocalObjectReference{
					Name: statusUpdate.Name,
				},
				ToolRef: kubechainv1alpha1.LocalObjectReference{
					Name: tc.Function.Name,
				},
				Arguments: tc.Function.Arguments,
			},
		}
		if err := r.Client.Create(ctx, newTRTC); err != nil {
			logger.Error(err, "Failed to create TaskRunToolCall", "name", newName)
			return ctrl.Result{}, err
		}
		logger.Info("Created TaskRunToolCall", "name", newName, "requestId", statusUpdate.Status.ToolCallRequestID)
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "ToolCallCreated", "Created TaskRunToolCall "+newName)
	}
	return ctrl.Result{RequeueAfter: time.Second * 5}, nil
}

// initializePhaseAndSpan initializes the TaskRun phase and starts tracing
func (r *TaskRunReconciler) initializePhaseAndSpan(ctx context.Context, statusUpdate *kubechainv1alpha1.TaskRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Start tracing the TaskRun
	tracer := r.Tracer
	if tracer == nil {
		tracer = otel.GetTracerProvider().Tracer("taskrun")
	}

	// Make sure we provide a meaningful span name that includes the TaskRun name
	spanName := fmt.Sprintf("TaskRun/%s", statusUpdate.Name)
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))

	// We need to explicitly end the span so the root span is properly recorded
	// This is not what we want long-term, but it ensures spans show up correctly
	// while we resolve the issue with maintaining spans across reconciliation calls
	defer span.End()

	// Store span context in status
	spanCtx := span.SpanContext()
	statusUpdate.Status.SpanContext = &kubechainv1alpha1.SpanContext{
		TraceID: spanCtx.TraceID().String(),
		SpanID:  spanCtx.SpanID().String(),
	}

	// Set useful attributes on the span
	span.SetAttributes(
		attribute.String("taskrun.name", statusUpdate.Name),
		attribute.String("taskrun.namespace", statusUpdate.Namespace),
		attribute.String("taskrun.uid", string(statusUpdate.UID)),
	)

	statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseInitializing
	statusUpdate.Status.Ready = false
	statusUpdate.Status.Status = kubechainv1alpha1.TaskRunStatusStatusPending
	statusUpdate.Status.StatusDetail = "Initializing"
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Failed to update TaskRun status")
		span.SetStatus(codes.Error, "Failed to update TaskRun status")
		span.RecordError(err)
		return ctrl.Result{}, err
	}

	// By ending the span now, we ensure it's properly recorded
	// This approach creates a separate span for each reconciliation rather than
	// a single span that covers the entire TaskRun lifecycle
	span.SetStatus(codes.Ok, "TaskRun initialized")
	return ctrl.Result{Requeue: true}, nil
}

// createLLMRequestSpan creates a child span for an LLM request that is properly linked to the parent span
func (r *TaskRunReconciler) createLLMRequestSpan(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, contextWindowSize, toolsCount int) (context.Context, trace.Span) {
	// Use controller's tracer if available, otherwise get the global tracer
	tracer := r.Tracer
	if tracer == nil {
		tracer = otel.GetTracerProvider().Tracer("taskrun")
	}

	// If we have a parent span context, create a child span linked to the parent
	if taskRun.Status.SpanContext != nil && taskRun.Status.SpanContext.TraceID != "" {
		// Parse the trace and span IDs from the stored context
		traceIDBytes, err := trace.TraceIDFromHex(taskRun.Status.SpanContext.TraceID)
		if err == nil {
			spanIDBytes, err := trace.SpanIDFromHex(taskRun.Status.SpanContext.SpanID)
			if err == nil {
				// Create a span context with the stored IDs
				spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceIDBytes,
					SpanID:     spanIDBytes,
					TraceFlags: trace.FlagsSampled,
					Remote:     false,
				})

				// Create context with the span context
				ctx = trace.ContextWithSpanContext(ctx, spanCtx)
			}
		}

		// Create a child span that will properly link to the parent
		childCtx, childSpan := tracer.Start(ctx, fmt.Sprintf("TaskRun/%s/LLMRequest", taskRun.Name),
			trace.WithSpanKind(trace.SpanKindClient))

		// Set attributes for the LLM request
		childSpan.SetAttributes(
			attribute.Int("context_window_size", contextWindowSize),
			attribute.Int("tools_count", toolsCount),
			attribute.String("taskrun.name", taskRun.Name),
			attribute.String("taskrun.namespace", taskRun.Namespace),
		)

		return childCtx, childSpan
	}

	// No parent span, just use the current context
	return ctx, nil
}

// Reconcile validates the taskrun's task reference and sends the prompt to the LLM.
func (r *TaskRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var taskRun kubechainv1alpha1.TaskRun
	if err := r.Get(ctx, req.NamespacedName, &taskRun); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", taskRun.Name)

	// Create a copy for status update
	statusUpdate := taskRun.DeepCopy()

	// Initialize phase if not set
	if statusUpdate.Status.Phase == "" {
		return r.initializePhaseAndSpan(ctx, statusUpdate)
	}

	// Skip reconciliation for terminal states
	if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseFinalAnswer || statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseFailed {
		logger.V(1).Info("TaskRun in terminal state, skipping reconciliation", "phase", statusUpdate.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Step 1: Validate Task and Agent
	task, agent, result, err := r.validateTaskAndAgent(ctx, &taskRun, statusUpdate)
	if err != nil || !result.IsZero() {
		return result, err
	}

	// Step 2: Initialize Phase if necessary
	if result, err := r.prepareForLLM(ctx, &taskRun, statusUpdate, task, agent); err != nil || !result.IsZero() {
		return result, err
	}

	// Step 3: Handle tool calls phase
	if taskRun.Status.Phase == kubechainv1alpha1.TaskRunPhaseToolCallsPending {
		return r.processToolCalls(ctx, &taskRun)
	}

	// Step 4: Check for unexpected phase
	if taskRun.Status.Phase != kubechainv1alpha1.TaskRunPhaseReadyForLLM {
		logger.Info("TaskRun in unknown phase", "phase", taskRun.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Step 5: Get API credentials (LLM is returned but not used)
	_, apiKey, err := r.getLLMAndCredentials(ctx, agent, &taskRun, statusUpdate)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Step 6: Create LLM client
	llmClient, err := r.newLLMClient(apiKey)
	if err != nil {
		logger.Error(err, "Failed to create OpenAI client")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.StatusDetail = "Failed to create OpenAI client: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "OpenAIClientCreationFailed", err.Error())

		// End span since we've failed with a terminal error
		r.endTaskRunSpan(ctx, &taskRun, codes.Error, "Failed to create OpenAI client: "+err.Error())

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Step 7: Collect tools from all sources
	tools := r.collectTools(ctx, agent)

	r.recorder.Event(&taskRun, corev1.EventTypeNormal, "SendingContextWindowToLLM", "Sending context window to LLM")

	// Create child span for LLM call
	childCtx, childSpan := r.createLLMRequestSpan(ctx, &taskRun, len(taskRun.Status.ContextWindow), len(tools))
	if childSpan != nil {
		defer childSpan.End()
	}

	// Step 8: Send the prompt to the LLM
	output, err := llmClient.SendRequest(childCtx, taskRun.Status.ContextWindow, tools)
	if err != nil {
		logger.Error(err, "LLM request failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = fmt.Sprintf("LLM request failed: %v", err)
		statusUpdate.Status.Error = err.Error()

		// Check for LLMRequestError with 4xx status code
		var llmErr *llmclient.LLMRequestError
		if errors.As(err, &llmErr) && llmErr.StatusCode >= 400 && llmErr.StatusCode < 500 {
			logger.Info("LLM request failed with 4xx status code, marking as failed",
				"statusCode", llmErr.StatusCode,
				"message", llmErr.Message)
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
			r.recorder.Event(&taskRun, corev1.EventTypeWarning, "LLMRequestFailed4xx",
				fmt.Sprintf("LLM request failed with status %d: %s", llmErr.StatusCode, llmErr.Message))
		} else {
			r.recorder.Event(&taskRun, corev1.EventTypeWarning, "LLMRequestFailed", err.Error())
		}

		// Record error in span
		if childSpan != nil {
			childSpan.RecordError(err)
			childSpan.SetStatus(codes.Error, err.Error())
		}

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status after LLM error")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Mark span as successful if we reach here
	if childSpan != nil {
		childSpan.SetStatus(codes.Ok, "LLM request succeeded")
	}

	// Step 9: Process LLM response
	var llmResult ctrl.Result
	llmResult, err = r.processLLMResponse(ctx, output, &taskRun, statusUpdate)
	if err != nil {
		logger.Error(err, "Failed to process LLM response")
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Failed to process LLM response: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "LLMResponseProcessingFailed", err.Error())

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status after LLM response processing error")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil // Don't return the error to avoid requeuing
	}

	if !llmResult.IsZero() {
		return llmResult, nil
	}

	// Step 10: Update final status
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update TaskRun status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled taskrun",
		"name", taskRun.Name,
		"ready", statusUpdate.Status.Ready,
		"phase", statusUpdate.Status.Phase)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("taskrun-controller")
	if r.newLLMClient == nil {
		r.newLLMClient = llmclient.NewRawOpenAIClient
	}

	// Initialize MCPManager if not already set
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManager()
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.TaskRun{}).
		Complete(r)
}
