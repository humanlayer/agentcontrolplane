package taskruntoolcall

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

	"github.com/google/uuid"
	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/humanlayer"
	"github.com/humanlayer/smallchain/kubechain/internal/humanlayerapi"
	"github.com/humanlayer/smallchain/kubechain/internal/mcpmanager"
)

const (
	DetailToolExecutedSuccess = "Tool executed successfully"
	DetailInvalidArgsJSON     = "Invalid arguments JSON"
)

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=taskruntoolcalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=taskruntoolcalls/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=tools,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// TaskRunToolCallReconciler reconciles a TaskRunToolCall object.
type TaskRunToolCallReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	server          *http.Server
	MCPManager      mcpmanager.MCPManagerInterface
	HLClientFactory humanlayer.HumanLayerClientFactory
}

func (r *TaskRunToolCallReconciler) webhookHandler(w http.ResponseWriter, req *http.Request) {
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

		// Update TaskRunToolCall status
		if err := r.updateTaskRunToolCall(context.Background(), webhook); err != nil {
			logger.Error(err, "Failed to update TaskRunToolCall status")
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

func (r *TaskRunToolCallReconciler) updateTaskRunToolCall(ctx context.Context, webhook humanlayer.FunctionCall) error {
	logger := log.FromContext(ctx)
	var trtc kubechainv1alpha1.TaskRunToolCall

	if err := r.Get(ctx, client.ObjectKey{Namespace: "default", Name: webhook.RunID}, &trtc); err != nil {
		return fmt.Errorf("failed to get TaskRunToolCall: %w", err)
	}

	logger.Info("Webhook received",
		"runID", webhook.RunID,
		"status", webhook.Status,
		"approved", *webhook.Status.Approved,
		"comment", webhook.Status.Comment)

	if webhook.Status != nil && webhook.Status.Approved != nil {
		// Update the TaskRunToolCall status with the webhook data
		if *webhook.Status.Approved {
			trtc.Status.Result = "Approved"
			trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseSucceeded
			trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeSucceeded
			trtc.Status.StatusDetail = DetailToolExecutedSuccess
		} else {
			trtc.Status.Result = "Rejected"
			trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseToolCallRejected
			trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeSucceeded
			trtc.Status.StatusDetail = "Tool execution rejected"
		}

		// if webhook.Status.RespondedAt != nil {
		// 		trtc.Status.RespondedAt = &metav1.Time{Time: *webhook.Status.RespondedAt}
		// }

		// if webhook.Status.Approved != nil {
		// 		trtc.Status.Approved = webhook.Status.Approved
		// }

		if err := r.Status().Update(ctx, &trtc); err != nil {
			return fmt.Errorf("failed to update TaskRunToolCall status: %w", err)
		}
		logger.Info("TaskRunToolCall status updated", "name", trtc.Name, "phase", trtc.Status.Phase)
	}

	return nil
}

// isMCPTool checks if a tool is an MCP tool and extracts the server name and actual tool name
func isMCPTool(trtc *kubechainv1alpha1.TaskRunToolCall) (serverName string, actualToolName string, isMCP bool) {
	// If this isn't an MCP, no server name__tool_name to split
	if trtc.Spec.ToolType != kubechainv1alpha1.ToolTypeMCP {
		return "", trtc.Spec.ToolRef.Name, false
	}

	// For MCP tools, we still need to parse the name to get the server and tool parts
	parts := strings.Split(trtc.Spec.ToolRef.Name, "__")
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	// This shouldn't happen if toolType is set correctly, but just in case
	return "", trtc.Spec.ToolRef.Name, true
}

// executeMCPTool executes a tool call on an MCP server
func (r *TaskRunToolCallReconciler) executeMCPTool(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall, serverName, toolName string, args map[string]interface{}) error {
	logger := log.FromContext(ctx)

	if r.MCPManager == nil {
		return fmt.Errorf("MCPManager is not initialized")
	}

	// Call the MCP tool
	result, err := r.MCPManager.CallTool(ctx, serverName, toolName, args)
	if err != nil {
		logger.Error(err, "Failed to call MCP tool",
			"serverName", serverName,
			"toolName", toolName)
		return err
	}

	// Update TaskRunToolCall status with the MCP tool result
	trtc.Status.Result = result
	trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseSucceeded
	trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeSucceeded
	trtc.Status.StatusDetail = "MCP tool executed successfully"

	return nil
}

// initializeTRTC initializes the TaskRunToolCall status to Pending:Pending
// Returns error if update fails
func (r *TaskRunToolCallReconciler) initializeTRTC(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall) error {
	logger := log.FromContext(ctx)

	trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhasePending
	trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypePending
	trtc.Status.StatusDetail = "Initializing"
	trtc.Status.StartTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, trtc); err != nil {
		logger.Error(err, "Failed to update initial status on TaskRunToolCall")
		return err
	}
	return nil
}

// completeSetup transitions a TaskRunToolCall from Pending:Pending to Ready:Pending
// Returns error if update fails
func (r *TaskRunToolCallReconciler) completeSetup(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall) error {
	logger := log.FromContext(ctx)

	trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeReady
	trtc.Status.StatusDetail = "Setup complete"
	if err := r.Status().Update(ctx, trtc); err != nil {
		logger.Error(err, "Failed to update status to Ready on TaskRunToolCall")
		return err
	}
	return nil
}

// checkCompletedOrExisting checks if the TRTC is already complete or has a child TaskRun
func (r *TaskRunToolCallReconciler) checkCompletedOrExisting(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall) (completed bool, err error, handled bool) {
	logger := log.FromContext(ctx)

	// Check if a child TaskRun already exists for this tool call
	var taskList kubechainv1alpha1.TaskList
	if err := r.List(ctx, &taskList, client.InNamespace(trtc.Namespace), client.MatchingLabels{"kubechain.humanlayer.dev/task": trtc.Name}); err != nil {
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
func (r *TaskRunToolCallReconciler) parseArguments(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall) (args map[string]interface{}, err error) {
	logger := log.FromContext(ctx)

	// Parse the arguments string as JSON (needed for both MCP and traditional tools)
	if err := json.Unmarshal([]byte(trtc.Spec.Arguments), &args); err != nil {
		logger.Error(err, "Failed to parse arguments")
		trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeError
		trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseFailed
		trtc.Status.StatusDetail = DetailInvalidArgsJSON
		trtc.Status.Error = err.Error()
		r.recorder.Event(trtc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
		if err := r.Status().Update(ctx, trtc); err != nil {
			logger.Error(err, "Failed to update status")
			return nil, err
		}
		return nil, err
	}

	return args, nil
}

// processMCPTool handles execution of an MCP tool
func (r *TaskRunToolCallReconciler) processMCPTool(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall, serverName, mcpToolName string, args map[string]interface{}) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Executing MCP tool", "serverName", serverName, "toolName", mcpToolName)

	// Execute the MCP tool
	if err := r.executeMCPTool(ctx, trtc, serverName, mcpToolName, args); err != nil {
		trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeError
		trtc.Status.StatusDetail = fmt.Sprintf("MCP tool execution failed: %v", err)
		trtc.Status.Error = err.Error()
		trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseFailed
		r.recorder.Event(trtc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())

		if updateErr := r.Status().Update(ctx, trtc); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Save the result
	if err := r.Status().Update(ctx, trtc); err != nil {
		logger.Error(err, "Failed to update TaskRunToolCall status after execution")
		return ctrl.Result{}, err
	}
	logger.Info("MCP tool execution completed", "result", trtc.Status.Result)
	r.recorder.Event(trtc, corev1.EventTypeNormal, "ExecutionSucceeded",
		fmt.Sprintf("MCP tool %q executed successfully", trtc.Spec.ToolRef.Name))
	return ctrl.Result{}, nil
}

// getTraditionalTool retrieves and validates the Traditional Tool resource
func (r *TaskRunToolCallReconciler) getTraditionalTool(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall) (*kubechainv1alpha1.Tool, string, error) {
	logger := log.FromContext(ctx)

	// Get the Tool resource
	var tool kubechainv1alpha1.Tool
	if err := r.Get(ctx, client.ObjectKey{Namespace: trtc.Namespace, Name: trtc.Spec.ToolRef.Name}, &tool); err != nil {
		logger.Error(err, "Failed to get Tool", "tool", trtc.Spec.ToolRef.Name)
		trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeError
		trtc.Status.StatusDetail = fmt.Sprintf("Failed to get Tool: %v", err)
		trtc.Status.Error = err.Error()
		r.recorder.Event(trtc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if err := r.Status().Update(ctx, trtc); err != nil {
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
		toolType = tool.Spec.ToolType
	} else {
		err := fmt.Errorf("unknown tool type: tool doesn't have valid execution configuration")
		logger.Error(err, "Invalid tool configuration")
		trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeError
		trtc.Status.StatusDetail = err.Error()
		trtc.Status.Error = err.Error()
		r.recorder.Event(trtc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if err := r.Status().Update(ctx, trtc); err != nil {
			logger.Error(err, "Failed to update status")
			return nil, "", err
		}
		return nil, "", err
	}

	return &tool, toolType, nil
}

// processDelegateToAgent handles agent delegation (not yet implemented)
func (r *TaskRunToolCallReconciler) processDelegateToAgent(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	err := fmt.Errorf("delegation is not implemented yet; only direct execution is supported")
	logger.Error(err, "Delegation not implemented")
	trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeError
	trtc.Status.StatusDetail = err.Error()
	trtc.Status.Error = err.Error()
	r.recorder.Event(trtc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
	if err := r.Status().Update(ctx, trtc); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}

// handleUnsupportedToolType handles the fallback for unrecognized tool types
func (r *TaskRunToolCallReconciler) handleUnsupportedToolType(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	err := fmt.Errorf("unsupported tool configuration")
	logger.Error(err, "Unsupported tool configuration")
	trtc.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeError
	trtc.Status.StatusDetail = err.Error()
	trtc.Status.Error = err.Error()
	r.recorder.Event(trtc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
	if err := r.Status().Update(ctx, trtc); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}

// getMCPServer gets the MCPServer for a tool and checks if it requires approval
func (r *TaskRunToolCallReconciler) getMCPServer(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall) (*kubechainv1alpha1.MCPServer, bool, error) {
	logger := log.FromContext(ctx)

	// Check if this is an MCP tool
	serverName, _, isMCP := isMCPTool(trtc)
	if !isMCP {
		return nil, false, nil
	}

	// Get the MCPServer
	var mcpServer kubechainv1alpha1.MCPServer
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: trtc.Namespace,
		Name:      serverName,
	}, &mcpServer); err != nil {
		logger.Error(err, "Failed to get MCPServer", "serverName", serverName)
		return nil, false, err
	}

	return &mcpServer, mcpServer.Spec.ApprovalContactChannel != nil, nil
}

// getContactChannel fetches and validates the ContactChannel resource
func (r *TaskRunToolCallReconciler) getContactChannel(ctx context.Context, channelName string, trtcNamespace string) (*kubechainv1alpha1.ContactChannel, error) {
	var contactChannel kubechainv1alpha1.ContactChannel
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: trtcNamespace,
		Name:      channelName,
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

func (r *TaskRunToolCallReconciler) getHumanLayerAPIKey(ctx context.Context, secretKeyRefName string, secretKeyRefKey string, trtcNamespace string) (string, error) {
	var secret corev1.Secret
	err := r.Get(ctx, client.ObjectKey{
		Namespace: trtcNamespace,
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
func (r *TaskRunToolCallReconciler) setStatusError(ctx context.Context, trtcPhase kubechainv1alpha1.TaskRunToolCallPhase, eventType string, trtc *kubechainv1alpha1.TaskRunToolCall, err error) (ctrl.Result, error, bool) {
	trtcDeepCopy := trtc.DeepCopy()
	logger := log.FromContext(ctx)

	// Always set Status to Error when using setStatusError
	trtcDeepCopy.Status.Status = kubechainv1alpha1.TaskRunToolCallStatusTypeError
	// Set Phase to the provided Phase value
	trtcDeepCopy.Status.Phase = trtcPhase

	// Handle nil error case
	errorMessage := "Unknown error occurred"
	if err != nil {
		errorMessage = err.Error()
	}

	trtcDeepCopy.Status.StatusDetail = errorMessage
	trtcDeepCopy.Status.Error = errorMessage
	r.recorder.Event(trtcDeepCopy, corev1.EventTypeWarning, eventType, errorMessage)

	if err := r.Status().Update(ctx, trtcDeepCopy); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err, true
	}
	return ctrl.Result{}, nil, true
}

func (r *TaskRunToolCallReconciler) updateTRTCStatus(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall, trtcStatusType kubechainv1alpha1.TaskRunToolCallStatusType, trtcStatusPhase kubechainv1alpha1.TaskRunToolCallPhase, statusDetail string, result string) (ctrl.Result, error, bool) {
	logger := log.FromContext(ctx)

	trtcDeepCopy := trtc.DeepCopy()

	trtcDeepCopy.Status.Status = trtcStatusType
	trtcDeepCopy.Status.StatusDetail = statusDetail
	trtcDeepCopy.Status.Phase = trtcStatusPhase

	// Store the result for tool call rejection
	if trtcStatusPhase == kubechainv1alpha1.TaskRunToolCallPhaseToolCallRejected {
		trtcDeepCopy.Status.Result = result
	}

	if err := r.Status().Update(ctx, trtcDeepCopy); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err, true
	}
	return ctrl.Result{}, nil, true
}

func (r *TaskRunToolCallReconciler) postToHumanLayer(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall, contactChannel *kubechainv1alpha1.ContactChannel, apiKey string) (*humanlayerapi.FunctionCallOutput, int, error) {
	client := r.HLClientFactory.NewHumanLayerClient()

	switch contactChannel.Spec.Type {
	case kubechainv1alpha1.ContactChannelTypeSlack:
		client.SetSlackConfig(contactChannel.Spec.Slack)
	case kubechainv1alpha1.ContactChannelTypeEmail:
		client.SetEmailConfig(contactChannel.Spec.Email)
	default:
		return nil, 0, fmt.Errorf("unsupported channel type: %s", contactChannel.Spec.Type)
	}

	toolName := trtc.Spec.ToolRef.Name
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(trtc.Spec.Arguments), &args); err != nil {
		// Set default error map if JSON parsing fails
		args = map[string]interface{}{
			"error": "Error reading JSON",
		}
	}
	client.SetFunctionCallSpec(toolName, args)

	client.SetCallID("ec-" + uuid.New().String()[:7])
	client.SetRunID(trtc.Name)
	client.SetAPIKey(apiKey)

	functionCall, statusCode, err := client.RequestApproval(ctx)

	if err == nil {
		r.recorder.Event(trtc, corev1.EventTypeNormal, "HumanLayerRequestSent", "HumanLayer request sent")
	}

	return functionCall, statusCode, err
}

// handlePendingApproval checks if an existing human approval is completed and updates status accordingly
func (r *TaskRunToolCallReconciler) handlePendingApproval(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall, apiKey string) (ctrl.Result, error, bool) {
	logger := log.FromContext(ctx)

	// Only process if in the awaiting human approval phase
	if trtc.Status.Phase != kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanApproval {
		return ctrl.Result{}, nil, false
	}

	// Verify we have a call ID
	if trtc.Status.ExternalCallID == "" {
		logger.Info("Missing ExternalCallID in AwaitingHumanApproval phase")
		return ctrl.Result{}, nil, false
	}

	client := r.HLClientFactory.NewHumanLayerClient()
	client.SetCallID(trtc.Status.ExternalCallID)
	client.SetAPIKey(apiKey)
	functionCall, _, err := client.GetFunctionCallStatus(ctx)
	if err != nil {
		return ctrl.Result{}, err, true
	}

	status := functionCall.GetStatus()

	approved, ok := status.GetApprovedOk()

	if !ok || approved == nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil, true
	}

	if *approved {
		return r.updateTRTCStatus(ctx, trtc,
			kubechainv1alpha1.TaskRunToolCallStatusTypeReady,
			kubechainv1alpha1.TaskRunToolCallPhaseReadyToExecuteApprovedTool,
			"Ready to execute approved tool", "")
	} else {
		return r.updateTRTCStatus(ctx, trtc,
			kubechainv1alpha1.TaskRunToolCallStatusTypeSucceeded,
			kubechainv1alpha1.TaskRunToolCallPhaseToolCallRejected,
			"Tool execution rejected", status.GetComment())
	}
}

// requestHumanApproval handles setting up a new human approval request
func (r *TaskRunToolCallReconciler) requestHumanApproval(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall,
	contactChannel *kubechainv1alpha1.ContactChannel, apiKey string, mcpServer *kubechainv1alpha1.MCPServer,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Skip if already in progress or approved
	if trtc.Status.Phase == kubechainv1alpha1.TaskRunToolCallPhaseReadyToExecuteApprovedTool {
		return ctrl.Result{}, nil
	}

	// Update to awaiting approval phase while maintaining current status
	trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanApproval
	trtc.Status.StatusDetail = fmt.Sprintf("Waiting for human approval via contact channel %s", mcpServer.Spec.ApprovalContactChannel.Name)
	r.recorder.Event(trtc, corev1.EventTypeNormal, "AwaitingHumanApproval",
		fmt.Sprintf("Tool execution requires approval via contact channel %s", mcpServer.Spec.ApprovalContactChannel.Name))

	if err := r.Status().Update(ctx, trtc); err != nil {
		logger.Error(err, "Failed to update TaskRunToolCall status")
		return ctrl.Result{}, err
	}

	// Verify HLClient is initialized
	if r.HLClientFactory == nil {
		err := fmt.Errorf("HLClient not initialized")
		result, errStatus, _ := r.setStatusError(ctx, kubechainv1alpha1.TaskRunToolCallPhaseErrorRequestingHumanApproval,
			"NoHumanLayerClient", trtc, err)
		return result, errStatus
	}

	// Post to HumanLayer to request approval
	functionCall, statusCode, err := r.postToHumanLayer(ctx, trtc, contactChannel, apiKey)
	if err != nil {
		errorMsg := fmt.Errorf("HumanLayer request failed with status code: %d", statusCode)
		if err != nil {
			errorMsg = fmt.Errorf("HumanLayer request failed with status code %d: %v", statusCode, err)
		}
		result, errStatus, _ := r.setStatusError(ctx, kubechainv1alpha1.TaskRunToolCallPhaseErrorRequestingHumanApproval,
			"HumanLayerRequestFailed", trtc, errorMsg)
		return result, errStatus
	}

	// Update with call ID and requeue
	callId := functionCall.GetCallId()
	trtc.Status.ExternalCallID = callId
	if err := r.Status().Update(ctx, trtc); err != nil {
		logger.Error(err, "Failed to update TaskRunToolCall status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *TaskRunToolCallReconciler) requestHumanContact(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall, contactChannel *kubechainv1alpha1.ContactChannel, apiKey string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanInput
	trtc.Status.StatusDetail = fmt.Sprintf("Waiting for human input via contact channel %s", contactChannel.Name)

	r.recorder.Event(trtc, corev1.EventTypeNormal, "AwaitingHumanContact",
		fmt.Sprintf("Tool response requires human input via contact channel %s", contactChannel.Name))

	if err := r.Status().Update(ctx, trtc); err != nil {
		logger.Error(err, "Failed to update TaskRunToolCall status")
		return ctrl.Result{}, err
	}

	// TODO: Update ExternalCallID

	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// handleMCPApprovalFlow encapsulates the MCP approval flow logic
func (r *TaskRunToolCallReconciler) handleMCPApprovalFlow(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall) (result ctrl.Result, err error, handled bool) {
	// We've already been through the approval flow and are ready to execute the tool
	if trtc.Status.Phase == kubechainv1alpha1.TaskRunToolCallPhaseReadyToExecuteApprovedTool {
		return ctrl.Result{}, nil, false
	}

	// Check if this is an MCP tool and needs approval
	mcpServer, needsApproval, err := r.getMCPServer(ctx, trtc)
	if err != nil {
		return ctrl.Result{}, err, true
	}

	// If not an MCP tool or no approval needed, continue with normal processing
	if mcpServer == nil || !needsApproval {
		return ctrl.Result{}, nil, false
	}

	// Get contact channel and API key information
	trtcNamespace := trtc.Namespace
	contactChannel, err := r.getContactChannel(ctx, mcpServer.Spec.ApprovalContactChannel.Name, trtcNamespace)
	if err != nil {
		result, errStatus, _ := r.setStatusError(ctx, kubechainv1alpha1.TaskRunToolCallPhaseErrorRequestingHumanApproval,
			"NoContactChannel", trtc, err)
		return result, errStatus, true
	}

	apiKey, err := r.getHumanLayerAPIKey(ctx,
		contactChannel.Spec.APIKeyFrom.SecretKeyRef.Name,
		contactChannel.Spec.APIKeyFrom.SecretKeyRef.Key,
		trtcNamespace)

	if err != nil || apiKey == "" {
		result, errStatus, _ := r.setStatusError(ctx, kubechainv1alpha1.TaskRunToolCallPhaseErrorRequestingHumanApproval,
			"NoAPIKey", trtc, err)
		return result, errStatus, true
	}

	// Handle pending approval check first
	if trtc.Status.Phase == kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanApproval {
		result, err, handled := r.handlePendingApproval(ctx, trtc, apiKey)
		if handled {
			return result, err, true
		}
	}

	// Request human approval if not already done
	result, err = r.requestHumanApproval(ctx, trtc, contactChannel, apiKey, mcpServer)
	return result, err, true
}

func (r *TaskRunToolCallReconciler) handleHumanContactFlow(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall,
) (result ctrl.Result, err error, handled bool) {
	if trtc.Spec.ToolType != kubechainv1alpha1.ToolTypeHumanContact {
		return ctrl.Result{}, nil, false
	}

	trtcNamespace := trtc.Namespace
	contactChannel, err := r.getContactChannel(ctx, trtc.Spec.ToolRef.Name, trtcNamespace)
	if err != nil {
		result, errStatus, _ := r.setStatusError(ctx, kubechainv1alpha1.TaskRunToolCallPhaseErrorRequestingHumanApproval,
			"NoContactChannel", trtc, err)
		return result, errStatus, true
	}

	apiKey, err := r.getHumanLayerAPIKey(ctx,
		contactChannel.Spec.APIKeyFrom.SecretKeyRef.Name,
		contactChannel.Spec.APIKeyFrom.SecretKeyRef.Key,
		trtcNamespace)

	if err != nil || apiKey == "" {
		result, errStatus, _ := r.setStatusError(ctx, kubechainv1alpha1.TaskRunToolCallPhaseErrorRequestingHumanApproval,
			"NoAPIKey", trtc, err)
		return result, errStatus, true
	}

	if trtc.Status.Phase == kubechainv1alpha1.TaskRunToolCallPhaseAwaitingHumanInput {
		// TODO: Handle pending human input
	}

	result, err = r.requestHumanContact(ctx, trtc, contactChannel, apiKey)
	return result, err, true
}

// dispatchToolExecution routes tool execution to the appropriate handler based on tool type
func (r *TaskRunToolCallReconciler) dispatchToolExecution(ctx context.Context, trtc *kubechainv1alpha1.TaskRunToolCall,
	args map[string]interface{},
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// Check for MCP tool first
	serverName, mcpToolName, isMCP := isMCPTool(trtc)
	if isMCP && r.MCPManager != nil {
		return r.processMCPTool(ctx, trtc, serverName, mcpToolName, args)
	}

	// Get traditional Tool resource
	tool, toolType, err := r.getTraditionalTool(ctx, trtc)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Dispatch based on tool type
	switch toolType {
	case "delegateToAgent":
		return r.processDelegateToAgent(ctx, trtc)
	case "function":
		// return r.processBuiltinFunction(ctx, trtc, tool, args)
		logger.V(1).Info("Builtin function not implemented", "tool", tool.Name)
		return ctrl.Result{}, nil
	case "externalAPI":
		// return r.processExternalAPI(ctx, trtc, tool)
		logger.V(1).Info("External API not implemented", "tool", tool.Name)
		return ctrl.Result{}, nil
	default:
		return r.handleUnsupportedToolType(ctx, trtc)
	}
}

// Reconcile processes TaskRunToolCall objects.
func (r *TaskRunToolCallReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get the TaskRunToolCall resource
	var trtc kubechainv1alpha1.TaskRunToolCall
	if err := r.Get(ctx, req.NamespacedName, &trtc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("Reconciling TaskRunToolCall", "name", trtc.Name)

	// 1. Check for terminal states - early return
	if trtc.Status.Status == kubechainv1alpha1.TaskRunToolCallStatusTypeError ||
		trtc.Status.Status == kubechainv1alpha1.TaskRunToolCallStatusTypeSucceeded {
		logger.Info("TaskRunToolCall in terminal state, nothing to do", "status", trtc.Status.Status, "phase", trtc.Status.Phase)
		return ctrl.Result{}, nil
	}

	// 2. Initialize Pending:Pending status if not set
	if trtc.Status.Phase == "" {
		logger.Info("Initializing TaskRunToolCall to Pending:Pending")
		if err := r.initializeTRTC(ctx, &trtc); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 3. Complete setup: transition from Pending:Pending to Ready:Pending
	if trtc.Status.Status == kubechainv1alpha1.TaskRunToolCallStatusTypePending {
		logger.Info("Transitioning TaskRunToolCall from Pending:Pending to Ready:Pending")
		if err := r.completeSetup(ctx, &trtc); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 4. Check if already completed or has child TaskRun
	done, err, handled := r.checkCompletedOrExisting(ctx, &trtc)
	if handled {
		if err != nil {
			return ctrl.Result{}, err
		}
		if done {
			return ctrl.Result{}, nil
		}
	}

	// 5. Check that we're in Ready status before continuing
	if trtc.Status.Status != kubechainv1alpha1.TaskRunToolCallStatusTypeReady {
		logger.Error(nil, "TaskRunToolCall not in Ready status before execution",
			"status", trtc.Status.Status,
			"phase", trtc.Status.Phase)
		result, err, _ := r.setStatusError(ctx, kubechainv1alpha1.TaskRunToolCallPhaseFailed,
			"ExecutionFailedNotReady", &trtc, fmt.Errorf("TaskRunToolCall must be in Ready status before execution"))
		return result, err
	}

	// 6. Handle MCP approval flow
	result, err, handled := r.handleMCPApprovalFlow(ctx, &trtc)
	if handled {
		return result, err
	}

	// 7. Check if we need to request human approval
	result, err, handled = r.handleHumanContactFlow(ctx, &trtc)
	if handled {
		return result, err
	}

	// 8. Parse arguments for execution
	args, err := r.parseArguments(ctx, &trtc)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 9. Execute the appropriate tool type
	return r.dispatchToolExecution(ctx, &trtc, args)
}

func (r *TaskRunToolCallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("taskruntoolcall-controller")
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
		For(&kubechainv1alpha1.TaskRunToolCall{}).
		Complete(r)
}

func (r *TaskRunToolCallReconciler) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.server.Shutdown(ctx); err != nil {
		log.Log.Error(err, "Failed to shut down HTTP server")
	}
}
