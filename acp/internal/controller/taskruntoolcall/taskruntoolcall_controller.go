package toolcall

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/google/uuid"
	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayer"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
)

const (
	DetailToolExecutedSuccess = "Tool executed successfully"
	DetailInvalidArgsJSON     = "Invalid arguments JSON"
)

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=toolcalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=toolcalls/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=tools,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// ToolCallReconciler reconciles a ToolCall object.
type ToolCallReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	server          *http.Server
	MCPManager      mcpmanager.MCPManagerInterface
	HLClientFactory humanlayer.HumanLayerClientFactory
	Tracer          trace.Tracer
}

// --- OTel Helper Functions ---

// attachTaskRootSpan reconstructs the parent Task's root span context and attaches it to the current context.
func (r *ToolCallReconciler) attachTaskRootSpan(ctx context.Context, task *acp.Task) context.Context {
	if task.Status.SpanContext == nil || task.Status.SpanContext.TraceID == "" || task.Status.SpanContext.SpanID == "" {
		return ctx // No valid parent context to attach
	}
	traceID, err := trace.TraceIDFromHex(task.Status.SpanContext.TraceID)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to parse parent Task TraceID", "traceID", task.Status.SpanContext.TraceID)
		return ctx
	}
	spanID, err := trace.SpanIDFromHex(task.Status.SpanContext.SpanID)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to parse parent Task SpanID", "spanID", task.Status.SpanContext.SpanID)
		return ctx
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled, // Assuming we always sample if the parent was sampled
		Remote:     true,
	})
	return trace.ContextWithSpanContext(ctx, sc)
}

// attachTCRootSpan reconstructs the ToolCall's own root span context and attaches it.
func (r *ToolCallReconciler) attachTCRootSpan(ctx context.Context, tc *acp.ToolCall) context.Context {
	if tc.Status.SpanContext == nil || tc.Status.SpanContext.TraceID == "" || tc.Status.SpanContext.SpanID == "" {
		return ctx // No valid context to attach
	}
	traceID, err := trace.TraceIDFromHex(tc.Status.SpanContext.TraceID)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to parse TC TraceID", "traceID", tc.Status.SpanContext.TraceID)
		return ctx
	}
	spanID, err := trace.SpanIDFromHex(tc.Status.SpanContext.SpanID)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to parse TC SpanID", "spanID", tc.Status.SpanContext.SpanID)
		return ctx
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled, // Assuming we always sample if the parent was sampled
		Remote:     true,
	})
	return trace.ContextWithSpanContext(ctx, sc)
}

// --- End OTel Helper Functions ---

func (r *ToolCallReconciler) webhookHandler(w http.ResponseWriter, req *http.Request) {
	logger := log.FromContext(context.Background())
	var webhook humanlayer.FunctionCall
	if err := json.NewDecoder(req.Body).Decode(&webhook); err != nil {
		logger.Error(err, "Failed to decode webhook payload")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logger.Info("Received webhook", "webhook", webhook)

	if webhook.Status != nil && webhook.Status.Approved != nil {
		if *webhook.Status.Approved {
			logger.Info("Email approved", "comment", webhook.Status.Comment)
		} else {
			logger.Info("Email request denied")
		}

		// Update ToolCall status
		if err := r.updateToolCall(context.Background(), webhook); err != nil {
			logger.Error(err, "Failed to update ToolCall status")
			http.Error(w, "Failed to update status", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status": "ok"}`)); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func (r *ToolCallReconciler) updateToolCall(ctx context.Context, webhook humanlayer.FunctionCall) error {
	logger := log.FromContext(ctx)
	var tc acp.ToolCall

	if err := r.Get(ctx, client.ObjectKey{Namespace: "default", Name: webhook.RunID}, &tc); err != nil {
		return fmt.Errorf("failed to get ToolCall: %w", err)
	}

	logger.Info("Webhook received",
		"runID", webhook.RunID,
		"status", webhook.Status,
		"approved", *webhook.Status.Approved,
		"comment", webhook.Status.Comment)

	if webhook.Status != nil && webhook.Status.Approved != nil {
		// Update the ToolCall status with the webhook data
		if *webhook.Status.Approved {
			tc.Status.Result = "Approved"
			tc.Status.Phase = acp.ToolCallPhaseSucceeded
			tc.Status.Status = acp.ToolCallStatusTypeSucceeded
			tc.Status.StatusDetail = DetailToolExecutedSuccess
		} else {
			tc.Status.Result = "Rejected"
			tc.Status.Phase = acp.ToolCallPhaseToolCallRejected
			tc.Status.Status = acp.ToolCallStatusTypeSucceeded
			tc.Status.StatusDetail = "Tool execution rejected"
		}

		// if webhook.Status.RespondedAt != nil {
		// 		tc.Status.RespondedAt = &metav1.Time{Time: *webhook.Status.RespondedAt}
		// }

		// if webhook.Status.Approved != nil {
		// 		tc.Status.Approved = webhook.Status.Approved
		// }

		if err := r.Status().Update(ctx, &tc); err != nil {
			return fmt.Errorf("failed to update ToolCall status: %w", err)
		}
		logger.Info("ToolCall status updated", "name", tc.Name, "phase", tc.Status.Phase)
	}

	return nil
}

// isMCPTool checks if a tool is an MCP tool and extracts the server name and actual tool name
func isMCPTool(tc *acp.ToolCall) (serverName string, actualToolName string, isMCP bool) {
	// If this isn't an MCP, no server name__tool_name to split
	if tc.Spec.ToolType != acp.ToolTypeMCP {
		return "", tc.Spec.ToolRef.Name, false
	}

	// For MCP tools, we still need to parse the name to get the server and tool parts
	parts := strings.Split(tc.Spec.ToolRef.Name, "__")
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	// This shouldn't happen if toolType is set correctly, but just in case
	return "", tc.Spec.ToolRef.Name, true
}

// executeMCPTool executes a tool call on an MCP server, wrapped in a child span.
func (r *ToolCallReconciler) executeMCPTool(ctx context.Context, tc *acp.ToolCall, serverName, toolName string, args map[string]interface{}) error {
	logger := log.FromContext(ctx)

	// Start child span for MCP execution
	execCtx, execSpan := r.Tracer.Start(ctx, "ExecuteMCPTool", trace.WithAttributes(
		attribute.String("acp.mcp.server", serverName),
		attribute.String("acp.mcp.tool", toolName),
		attribute.String("acp.toolcall.name", tc.Name),
	))
	defer execSpan.End() // Ensure the span is ended

	if r.MCPManager == nil {
		err := fmt.Errorf("MCPManager is not initialized")
		execSpan.RecordError(err)
		execSpan.SetStatus(codes.Error, "MCPManager not initialized")
		return err
	}

	// Call the MCP tool
	result, err := r.MCPManager.CallTool(execCtx, serverName, toolName, args) // Use execCtx
	if err != nil {
		logger.Error(err, "Failed to call MCP tool",
			"serverName", serverName,
			"toolName", toolName)
		execSpan.RecordError(err)
		execSpan.SetStatus(codes.Error, "MCP tool call failed")
		return err // Propagate error
	}

	// Update ToolCall status with the MCP tool result
	tc.Status.Result = result
	tc.Status.Phase = acp.ToolCallPhaseSucceeded
	tc.Status.Status = acp.ToolCallStatusTypeSucceeded
	tc.Status.StatusDetail = "MCP tool executed successfully"

	execSpan.SetStatus(codes.Ok, "MCP tool executed successfully")
	execSpan.SetAttributes(attribute.String("acp.tool.result_preview", truncateString(result, 100))) // Add result preview

	return nil // Success
}

// initializeTC initializes the ToolCall status to Pending:Pending
// Returns error if update fails
func (r *ToolCallReconciler) initializeTC(ctx context.Context, tc *acp.ToolCall) error {
	logger := log.FromContext(ctx)

	tc.Status.Phase = acp.ToolCallPhasePending
	tc.Status.Status = acp.ToolCallStatusTypePending
	tc.Status.StatusDetail = "Initializing"
	tc.Status.StartTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, tc); err != nil {
		logger.Error(err, "Failed to update initial status on ToolCall")
		return err
	}
	return nil
}

// completeSetup transitions a ToolCall from Pending:Pending to Ready:Pending
// Returns error if update fails
func (r *ToolCallReconciler) completeSetup(ctx context.Context, tc *acp.ToolCall) error {
	logger := log.FromContext(ctx)

	tc.Status.Status = acp.ToolCallStatusTypeReady
	tc.Status.StatusDetail = "Setup complete"
	if err := r.Status().Update(ctx, tc); err != nil {
		logger.Error(err, "Failed to update status to Ready on ToolCall")
		return err
	}
	return nil
}

// checkCompletedOrExisting checks if the TC is already complete or has a child TaskRun
func (r *ToolCallReconciler) checkCompletedOrExisting(ctx context.Context, tc *acp.ToolCall) (completed bool, err error, handled bool) {
	logger := log.FromContext(ctx)

	// Check if a child TaskRun already exists for this tool call
	var taskList acp.TaskList
	if err := r.List(ctx, &taskList, client.InNamespace(tc.Namespace), client.MatchingLabels{"acp.humanlayer.dev/task": tc.Name}); err != nil {
		logger.Error(err, "Failed to list child Tasks")
		return true, err, true
	}
	if len(taskList.Items) > 0 {
		logger.Info("Child Task already exists", "childTask", taskList.Items[0].Name)
		// Optionally, sync status from child to parent.
		return true, nil, true
	}

	return false, nil, false
}

// parseArguments parses the tool call arguments
func (r *ToolCallReconciler) parseArguments(ctx context.Context, tc *acp.ToolCall) (args map[string]interface{}, err error) {
	logger := log.FromContext(ctx)

	// Parse the arguments string as JSON (needed for both MCP and traditional tools)
	if err := json.Unmarshal([]byte(tc.Spec.Arguments), &args); err != nil {
		logger.Error(err, "Failed to parse arguments")
		tc.Status.Status = acp.ToolCallStatusTypeError
		tc.Status.Phase = acp.ToolCallPhaseFailed
		tc.Status.StatusDetail = DetailInvalidArgsJSON
		tc.Status.Error = err.Error()
		r.recorder.Event(tc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
		if err := r.Status().Update(ctx, tc); err != nil {
			logger.Error(err, "Failed to update status")
			return nil, err
		}
		return nil, err
	}

	return args, nil
}

// processMCPTool handles execution of an MCP tool
func (r *ToolCallReconciler) processMCPTool(ctx context.Context, tc *acp.ToolCall, serverName, mcpToolName string, args map[string]interface{}) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Executing MCP tool", "serverName", serverName, "toolName", mcpToolName)

	// Execute the MCP tool
	if err := r.executeMCPTool(ctx, tc, serverName, mcpToolName, args); err != nil {
		tc.Status.Status = acp.ToolCallStatusTypeError
		tc.Status.StatusDetail = fmt.Sprintf("MCP tool execution failed: %v", err)
		tc.Status.Error = err.Error()
		tc.Status.Phase = acp.ToolCallPhaseFailed
		r.recorder.Event(tc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())

		if updateErr := r.Status().Update(ctx, tc); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Save the result
	if err := r.Status().Update(ctx, tc); err != nil {
		logger.Error(err, "Failed to update ToolCall status after execution")
		return ctrl.Result{}, err
	}
	logger.Info("MCP tool execution completed", "result", tc.Status.Result)
	r.recorder.Event(tc, corev1.EventTypeNormal, "ExecutionSucceeded",
		fmt.Sprintf("MCP tool %q executed successfully", tc.Spec.ToolRef.Name))
	return ctrl.Result{}, nil
}

// processDelegateToAgent handles agent delegation (not yet implemented)
func (r *ToolCallReconciler) processDelegateToAgent(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	err := fmt.Errorf("delegation is not implemented yet; only direct execution is supported")
	logger.Error(err, "Delegation not implemented")
	tc.Status.Status = acp.ToolCallStatusTypeError
	tc.Status.StatusDetail = err.Error()
	tc.Status.Error = err.Error()
	r.recorder.Event(tc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
	if err := r.Status().Update(ctx, tc); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}

// handleUnsupportedToolType handles the fallback for unrecognized tool types
func (r *ToolCallReconciler) handleUnsupportedToolType(ctx context.Context, tc *acp.ToolCall) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	err := fmt.Errorf("unsupported tool configuration")
	logger.Error(err, "Unsupported tool configuration")
	tc.Status.Status = acp.ToolCallStatusTypeError
	tc.Status.StatusDetail = err.Error()
	tc.Status.Error = err.Error()
	r.recorder.Event(tc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
	if err := r.Status().Update(ctx, tc); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}

// getMCPServer gets the MCPServer for a tool and checks if it requires approval
func (r *ToolCallReconciler) getMCPServer(ctx context.Context, tc *acp.ToolCall) (*acp.MCPServer, bool, error) {
	logger := log.FromContext(ctx)

	// Check if this is an MCP tool
	serverName, _, isMCP := isMCPTool(tc)
	if !isMCP {
		return nil, false, nil
	}

	// Get the MCPServer
	var mcpServer acp.MCPServer
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: tc.Namespace,
		Name:      serverName,
	}, &mcpServer); err != nil {
		logger.Error(err, "Failed to get MCPServer", "serverName", serverName)
		return nil, false, err
	}

	return &mcpServer, mcpServer.Spec.ApprovalContactChannel != nil, nil
}

// getContactChannel fetches and validates the ContactChannel resource
func (r *ToolCallReconciler) getContactChannel(ctx context.Context, mcpServer *acp.MCPServer, tcNamespace string) (*acp.ContactChannel, error) {
	var contactChannel acp.ContactChannel
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: tcNamespace,
		Name:      mcpServer.Spec.ApprovalContactChannel.Name,
	}, &contactChannel); err != nil {

		err := fmt.Errorf("failed to get ContactChannel: %v", err)
		return nil, err
	}

	// Validate that the ContactChannel is ready
	if !contactChannel.Status.Ready {
		err := fmt.Errorf("ContactChannel %s is not ready: %s", contactChannel.Name, contactChannel.Status.StatusDetail)
		return nil, err
	}

	return &contactChannel, nil
}

func (r *ToolCallReconciler) getHumanLayerAPIKey(ctx context.Context, secretKeyRefName string, secretKeyRefKey string, tcNamespace string) (string, error) {
	var secret corev1.Secret
	err := r.Get(ctx, client.ObjectKey{
		Namespace: tcNamespace,
		Name:      secretKeyRefName,
	}, &secret)
	if err != nil {
		err := fmt.Errorf("failed to get API key secret: %v", err)
		return "", err
	}

	apiKey := string(secret.Data[secretKeyRefKey])
	return apiKey, nil
}

//nolint:unparam
func (r *ToolCallReconciler) setStatusError(ctx context.Context, tcPhase acp.ToolCallPhase, eventType string, tc *acp.ToolCall, err error) (ctrl.Result, error, bool) {
	tcDeepCopy := tc.DeepCopy()
	logger := log.FromContext(ctx)

	// Always set Status to Error when using setStatusError
	tcDeepCopy.Status.Status = acp.ToolCallStatusTypeError
	// Set Phase to the provided Phase value
	tcDeepCopy.Status.Phase = tcPhase

	// Handle nil error case
	errorMessage := "Unknown error occurred"
	if err != nil {
		errorMessage = err.Error()
	}

	tcDeepCopy.Status.StatusDetail = errorMessage
	tcDeepCopy.Status.Error = errorMessage
	r.recorder.Event(tcDeepCopy, corev1.EventTypeWarning, eventType, errorMessage)

	if err := r.Status().Update(ctx, tcDeepCopy); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err, true
	}
	return ctrl.Result{}, nil, true
}

func (r *ToolCallReconciler) updateTCStatus(ctx context.Context, tc *acp.ToolCall, tcStatusType acp.ToolCallStatusType, tcStatusPhase acp.ToolCallPhase, statusDetail string, result string) (ctrl.Result, error, bool) {
	logger := log.FromContext(ctx)

	tcDeepCopy := tc.DeepCopy()

	tcDeepCopy.Status.Status = tcStatusType
	tcDeepCopy.Status.StatusDetail = statusDetail
	tcDeepCopy.Status.Phase = tcStatusPhase

	// Store the result for tool call rejection
	if tcStatusPhase == acp.ToolCallPhaseToolCallRejected {
		toolName := tc.Spec.ToolRef.Name
		tcDeepCopy.Status.Result = fmt.Sprintf("User denied `%s` with feedback: %s", toolName, result)
	}

	if err := r.Status().Update(ctx, tcDeepCopy); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err, true
	}
	return ctrl.Result{}, nil, true
}

func (r *ToolCallReconciler) postToHumanLayer(ctx context.Context, tc *acp.ToolCall, contactChannel *acp.ContactChannel, apiKey string) (*humanlayerapi.FunctionCallOutput, int, error) {
	client := r.HLClientFactory.NewHumanLayerClient()

	switch contactChannel.Spec.Type {
	case acp.ContactChannelTypeSlack:
		client.SetSlackConfig(contactChannel.Spec.Slack)
	case acp.ContactChannelTypeEmail:
		client.SetEmailConfig(contactChannel.Spec.Email)
	default:
		return nil, 0, fmt.Errorf("unsupported channel type: %s", contactChannel.Spec.Type)
	}

	toolName := tc.Spec.ToolRef.Name
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Spec.Arguments), &args); err != nil {
		// Set default error map if JSON parsing fails
		args = map[string]interface{}{
			"error": "Error reading JSON",
		}
	}
	client.SetFunctionCallSpec(toolName, args)

	client.SetCallID("ec-" + uuid.New().String()[:7])
	client.SetRunID(tc.Name)
	client.SetAPIKey(apiKey)

	functionCall, statusCode, err := client.RequestApproval(ctx)

	if err == nil {
		r.recorder.Event(tc, corev1.EventTypeNormal, "HumanLayerRequestSent", "HumanLayer request sent")
	}

	return functionCall, statusCode, err
}

// handlePendingApproval checks if an existing human approval is completed and updates status accordingly
func (r *ToolCallReconciler) handlePendingApproval(ctx context.Context, tc *acp.ToolCall, apiKey string) (ctrl.Result, error, bool) {
	logger := log.FromContext(ctx)

	// Only process if in the awaiting human approval phase
	if tc.Status.Phase != acp.ToolCallPhaseAwaitingHumanApproval {
		return ctrl.Result{}, nil, false
	}

	// Verify we have a call ID
	if tc.Status.ExternalCallID == "" {
		logger.Info("Missing ExternalCallID in AwaitingHumanApproval phase")
		return ctrl.Result{}, nil, false
	}

	client := r.HLClientFactory.NewHumanLayerClient()
	client.SetCallID(tc.Status.ExternalCallID)
	client.SetAPIKey(apiKey)
	// Fix: Ensure correct assignment for 3 return values
	functionCall, _, err := client.GetFunctionCallStatus(ctx) // Assign *humanlayerapi.FunctionCallOutput, int, error
	if err != nil {
		// Log the error but attempt to requeue, as it might be transient
		logger.Error(err, "Failed to get function call status from HumanLayer")
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil, true // Requeue after delay
	}

	// Check if functionCall is nil before accessing GetStatus
	if functionCall == nil {
		logger.Error(fmt.Errorf("GetFunctionCallStatus returned nil functionCall"), "HumanLayer API call returned unexpected nil object")
		// Decide how to handle this - maybe requeue or set an error status
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil, true // Requeue for now
	}

	status := functionCall.GetStatus()

	approved, ok := status.GetApprovedOk()

	if !ok || approved == nil {
		// Still pending, requeue
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil, true
	}

	if *approved {
		// Approval received, update status to ReadyToExecuteApprovedTool
		return r.updateTCStatus(ctx, tc,
			acp.ToolCallStatusTypeReady,
			acp.ToolCallPhaseReadyToExecuteApprovedTool,
			"Ready to execute approved tool", "")
	} else {
		// Rejection received, update status to ToolCallRejected
		return r.updateTCStatus(ctx, tc,
			acp.ToolCallStatusTypeSucceeded, // Succeeded because the rejection was processed
			acp.ToolCallPhaseToolCallRejected,
			"Tool execution rejected", status.GetComment())
	}
}

// requestHumanApproval handles setting up a new human approval request, wrapped in a child span.
func (r *ToolCallReconciler) requestHumanApproval(ctx context.Context, tc *acp.ToolCall,
	contactChannel *acp.ContactChannel, apiKey string, mcpServer *acp.MCPServer,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Start child span for the approval request process
	approvalCtx, approvalSpan := r.Tracer.Start(ctx, "RequestHumanApproval", trace.WithAttributes(
		attribute.String("acp.contactchannel.name", contactChannel.Name),
		attribute.String("acp.contactchannel.type", string(contactChannel.Spec.Type)),
		attribute.String("acp.toolcall.name", tc.Name),
	))
	defer approvalSpan.End() // Ensure the span is ended

	// Skip if already in progress or approved
	if tc.Status.Phase == acp.ToolCallPhaseReadyToExecuteApprovedTool {
		approvalSpan.SetStatus(codes.Ok, "Already approved, skipping request")
		return ctrl.Result{}, nil
	}

	// Update to awaiting approval phase while maintaining current status
	tc.Status.Phase = acp.ToolCallPhaseAwaitingHumanApproval
	tc.Status.StatusDetail = fmt.Sprintf("Waiting for human approval via contact channel %s", mcpServer.Spec.ApprovalContactChannel.Name)
	r.recorder.Event(tc, corev1.EventTypeNormal, "AwaitingHumanApproval",
		fmt.Sprintf("Tool execution requires approval via contact channel %s", mcpServer.Spec.ApprovalContactChannel.Name))

	// Use approvalCtx for the status update
	if err := r.Status().Update(approvalCtx, tc); err != nil {
		logger.Error(err, "Failed to update ToolCall status to AwaitingHumanApproval")
		approvalSpan.RecordError(err)
		approvalSpan.SetStatus(codes.Error, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Verify HLClient is initialized
	if r.HLClientFactory == nil {
		err := fmt.Errorf("HLClient not initialized")
		approvalSpan.RecordError(err)
		approvalSpan.SetStatus(codes.Error, "HLClient not initialized")
		// Use approvalCtx for setStatusError
		// Fix: Adjust return values from setStatusError
		result, errStatus, _ := r.setStatusError(approvalCtx, acp.ToolCallPhaseErrorRequestingHumanApproval,
			"NoHumanLayerClient", tc, err)
		return result, errStatus // Return only Result and error
	}

	// Post to HumanLayer to request approval using approvalCtx
	functionCall, statusCode, err := r.postToHumanLayer(approvalCtx, tc, contactChannel, apiKey)
	if err != nil {
		errorMsg := fmt.Errorf("HumanLayer request failed with status code: %d", statusCode)
		if err != nil {
			errorMsg = fmt.Errorf("HumanLayer request failed with status code %d: %v", statusCode, err)
		}
		approvalSpan.RecordError(errorMsg)
		approvalSpan.SetStatus(codes.Error, "HumanLayer request failed")
		// Use approvalCtx for setStatusError
		// Fix: Adjust return values from setStatusError
		result, errStatus, _ := r.setStatusError(approvalCtx, acp.ToolCallPhaseErrorRequestingHumanApproval,
			"HumanLayerRequestFailed", tc, errorMsg)
		return result, errStatus // Return only Result and error
	}

	// Update with call ID and requeue using approvalCtx
	callId := functionCall.GetCallId()
	tc.Status.ExternalCallID = callId
	approvalSpan.SetAttributes(attribute.String("acp.humanlayer.call_id", callId)) // Add call ID to span
	if err := r.Status().Update(approvalCtx, tc); err != nil {
		logger.Error(err, "Failed to update ToolCall status with ExternalCallID")
		approvalSpan.RecordError(err)
		approvalSpan.SetStatus(codes.Error, "Failed to update status with ExternalCallID")
		return ctrl.Result{}, err
	}

	approvalSpan.SetStatus(codes.Ok, "HumanLayer approval request sent")
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// handleMCPApprovalFlow encapsulates the MCP approval flow logic
func (r *ToolCallReconciler) handleMCPApprovalFlow(ctx context.Context, tc *acp.ToolCall) (result ctrl.Result, err error, handled bool) {
	// We've already been through the approval flow and are ready to execute the tool
	if tc.Status.Phase == acp.ToolCallPhaseReadyToExecuteApprovedTool {
		return ctrl.Result{}, nil, false
	}

	// Check if this is an MCP tool and needs approval
	mcpServer, needsApproval, err := r.getMCPServer(ctx, tc)
	if err != nil {
		return ctrl.Result{}, err, true
	}

	// If not an MCP tool or no approval needed, continue with normal processing
	if mcpServer == nil || !needsApproval {
		return ctrl.Result{}, nil, false
	}

	// Get contact channel and API key information
	tcNamespace := tc.Namespace
	contactChannel, err := r.getContactChannel(ctx, mcpServer, tcNamespace)
	if err != nil {
		result, errStatus, _ := r.setStatusError(ctx, acp.ToolCallPhaseErrorRequestingHumanApproval,
			"NoContactChannel", tc, err)
		return result, errStatus, true
	}

	apiKey, err := r.getHumanLayerAPIKey(ctx,
		contactChannel.Spec.APIKeyFrom.SecretKeyRef.Name,
		contactChannel.Spec.APIKeyFrom.SecretKeyRef.Key,
		tcNamespace)

	if err != nil || apiKey == "" {
		result, errStatus, _ := r.setStatusError(ctx, acp.ToolCallPhaseErrorRequestingHumanApproval,
			"NoAPIKey", tc, err)
		return result, errStatus, true
	}

	// Handle pending approval check first
	if tc.Status.Phase == acp.ToolCallPhaseAwaitingHumanApproval {
		result, err, handled := r.handlePendingApproval(ctx, tc, apiKey)
		if handled {
			return result, err, true
		}
	}

	// Request human approval if not already done
	result, err = r.requestHumanApproval(ctx, tc, contactChannel, apiKey, mcpServer)
	return result, err, true
}

// dispatchToolExecution routes tool execution to the appropriate handler based on tool type
func (r *ToolCallReconciler) dispatchToolExecution(ctx context.Context, tc *acp.ToolCall,
	args map[string]interface{},
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// Check for MCP tool first
	serverName, mcpToolName, isMCP := isMCPTool(tc)
	if isMCP && r.MCPManager != nil {
		return r.processMCPTool(ctx, tc, serverName, mcpToolName, args)
	}

	// Get traditional Tool resource
	tool, toolType, err := r.getTraditionalTool(ctx, tc)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Dispatch based on tool type
	switch toolType {
	case "delegateToAgent":
		return r.processDelegateToAgent(ctx, tc)
	case "function":
		// return r.processBuiltinFunction(ctx, tc, tool, args)
		logger.V(1).Info("Builtin function not implemented", "tool", tool.Name)
		return ctrl.Result{}, nil
	case "externalAPI":
		// return r.processExternalAPI(ctx, tc, tool)
		logger.V(1).Info("External API not implemented", "tool", tool.Name)
		return ctrl.Result{}, nil
	default:
		return r.handleUnsupportedToolType(ctx, tc)
	}
}

// Reconcile processes ToolCall objects.
func (r *ToolCallReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get the ToolCall resource
	var tc acp.ToolCall
	if err := r.Get(ctx, req.NamespacedName, &tc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("Reconciling ToolCall", "name", tc.Name)

	// Handle terminal states with proper span ending
	if tc.Status.Status == acp.ToolCallStatusTypeError ||
		tc.Status.Status == acp.ToolCallStatusTypeSucceeded {
		logger.Info("ToolCall in terminal state, nothing to do", "status", tc.Status.Status, "phase", tc.Status.Phase)

		// Attach the TC root span for finalization
		ctx = r.attachTCRootSpan(ctx, &tc)

		// Create a final span to properly end the trace
		_, endSpan := r.Tracer.Start(ctx, "FinalizeToolCall")
		if tc.Status.Status == acp.ToolCallStatusTypeError {
			endSpan.SetStatus(codes.Error, "TC ended with error")
		} else {
			endSpan.SetStatus(codes.Ok, "TC completed successfully")
		}
		endSpan.End()

		return ctrl.Result{}, nil
	}

	// Create the ToolCall root span if it doesn't exist yet
	if tc.Status.SpanContext == nil {
		// 1. Fetch parent task name from label
		parentTaskName := tc.Labels["acp.humanlayer.dev/task"]
		var parentTask acp.Task
		if err := r.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: parentTaskName}, &parentTask); err == nil {
			ctx = r.attachTaskRootSpan(ctx, &parentTask)
		}

		// 2. Create TC root span as child of Task span
		toolCallCtx, span := r.Tracer.Start(ctx, "ToolCall")
		defer span.End() // span is short-lived, just to write context

		// Add attributes to make traces more readable
		span.SetAttributes(
			attribute.String("toolcall.name", tc.Name),
			attribute.String("toolcall.tool", tc.Spec.ToolRef.Name),
			attribute.String("toolcall.toolType", string(tc.Spec.ToolType)),
		)

		tc.Status.SpanContext = &acp.SpanContext{
			TraceID: span.SpanContext().TraceID().String(),
			SpanID:  span.SpanContext().SpanID().String(),
		}

		if err := r.Status().Update(toolCallCtx, &tc); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to store ToolCall spanContext: %w", err)
		}

		return ctrl.Result{Requeue: true}, nil // requeue so we re-enter with this span next time
	}

	// Attach the TC root span for all other operations
	ctx = r.attachTCRootSpan(ctx, &tc)

	// 2. Initialize Pending:Pending status if not set
	if tc.Status.Phase == "" {
		logger.Info("Initializing ToolCall to Pending:Pending")
		if err := r.initializeTC(ctx, &tc); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 3. Complete setup: transition from Pending:Pending to Ready:Pending
	if tc.Status.Status == acp.ToolCallStatusTypePending {
		logger.Info("Transitioning ToolCall from Pending:Pending to Ready:Pending")
		if err := r.completeSetup(ctx, &tc); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 4. Check if already completed or has child TaskRun
	done, err, handled := r.checkCompletedOrExisting(ctx, &tc)
	if handled {
		if err != nil {
			return ctrl.Result{}, err
		}
		if done {
			return ctrl.Result{}, nil
		}
	}

	// 5. Check that we're in Ready status before continuing
	if tc.Status.Status != acp.ToolCallStatusTypeReady {
		logger.Error(nil, "ToolCall not in Ready status before execution",
			"status", tc.Status.Status,
			"phase", tc.Status.Phase)
		result, err, _ := r.setStatusError(ctx, acp.ToolCallPhaseFailed,
			"ExecutionFailedNotReady", &tc, fmt.Errorf("ToolCall must be in Ready status before execution"))
		return result, err
	}

	// 6. Handle MCP approval flow
	result, err, handled := r.handleMCPApprovalFlow(ctx, &tc)
	if handled {
		return result, err
	}

	// 7. Parse arguments for execution
	args, err := r.parseArguments(ctx, &tc)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 8. Execute the appropriate tool type
	return r.dispatchToolExecution(ctx, &tc, args)
}

func (r *ToolCallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("toolcall-controller")
	r.server = &http.Server{Addr: ":8080"} // Choose a port
	http.HandleFunc("/webhook/inbound", r.webhookHandler)

	// Initialize MCPManager if it hasn't been initialized yet
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManagerWithClient(r.Client)
	}

	if r.HLClientFactory == nil {
		client, err := humanlayer.NewHumanLayerClientFactory("")
		if err != nil {
			return err
		}

		r.HLClientFactory = client
	}

	go func() {
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Log.Error(err, "Failed to start HTTP server")
		}
	}()

	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.ToolCall{}).
		Complete(r)
}

func (r *ToolCallReconciler) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.server.Shutdown(ctx); err != nil {
		log.Log.Error(err, "Failed to shut down HTTP server")
	}
}

// Helper function to truncate strings for attributes
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getTraditionalTool retrieves and validates the Traditional Tool resource
func (r *ToolCallReconciler) getTraditionalTool(ctx context.Context, tc *acp.ToolCall) (*acp.Tool, string, error) {
	logger := log.FromContext(ctx)

	// Get the Tool resource
	var tool acp.Tool
	if err := r.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: tc.Spec.ToolRef.Name}, &tool); err != nil {
		logger.Error(err, "Failed to get Tool", "tool", tc.Spec.ToolRef.Name)
		tc.Status.Status = acp.ToolCallStatusTypeError
		tc.Status.StatusDetail = fmt.Sprintf("Failed to get Tool: %v", err)
		tc.Status.Error = err.Error()
		r.recorder.Event(tc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if err := r.Status().Update(ctx, tc); err != nil {
			logger.Error(err, "Failed to update status")
			return nil, "", err
		}
		return nil, "", err
	}

	// Determine tool type from the Tool resource
	var toolType string
	if tool.Spec.Execute.Builtin != nil {
		toolType = "function"
	} else if tool.Spec.AgentRef != nil {
		toolType = "delegateToAgent"
	} else if tool.Spec.ToolType != "" {
		toolType = string(tool.Spec.ToolType) // Use ToolType field if present
	} else {
		err := fmt.Errorf("unknown tool type: tool doesn't have valid execution configuration")
		logger.Error(err, "Invalid tool configuration")
		tc.Status.Status = acp.ToolCallStatusTypeError
		tc.Status.StatusDetail = err.Error()
		tc.Status.Error = err.Error()
		r.recorder.Event(tc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if err := r.Status().Update(ctx, tc); err != nil {
			logger.Error(err, "Failed to update status")
			return nil, "", err
		}
		return nil, "", err
	}

	return &tool, toolType, nil
}
