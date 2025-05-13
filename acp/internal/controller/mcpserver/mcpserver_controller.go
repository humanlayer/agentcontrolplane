package mcpserver

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
)

const (
	StatusError   = "Error"
	StatusPending = "Pending"
	StatusReady   = "Ready"
)

// MCPServerManagerInterface defines the interface for MCP server management
type MCPServerManagerInterface interface {
	ConnectServer(ctx context.Context, mcpServer *acp.MCPServer) error
	GetTools(serverName string) ([]acp.MCPTool, bool)
	GetConnection(serverName string) (*mcpmanager.MCPConnection, bool)
	DisconnectServer(serverName string)
	GetToolsForAgent(agent *acp.Agent) []acp.MCPTool
	CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (string, error)
	FindServerForTool(fullToolName string) (serverName string, toolName string, found bool)
	Close()
}

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=mcpservers/status,verbs=get;update;patch

// MCPServerReconciler reconciles a MCPServer object
type MCPServerReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	recorder   record.EventRecorder
	MCPManager MCPServerManagerInterface
}

// updateStatus updates the status of the MCPServer resource with the latest version
// This method handles conflicts by retrying the status update up to 3 times
func (r *MCPServerReconciler) updateStatus(ctx context.Context, req ctrl.Request, statusUpdate *acp.MCPServer) error {
	logger := log.FromContext(ctx)
	const maxRetries = 3

	var updateErr error
	for i := 0; i < maxRetries; i++ {
		// Get the latest version of the MCPServer
		var latestMCPServer acp.MCPServer
		if err := r.Get(ctx, req.NamespacedName, &latestMCPServer); err != nil {
			logger.Error(err, "Failed to get latest MCPServer before status update")
			return err
		}

		// Apply status updates to the latest version
		latestMCPServer.Status.Connected = statusUpdate.Status.Connected
		latestMCPServer.Status.Status = statusUpdate.Status.Status
		latestMCPServer.Status.StatusDetail = statusUpdate.Status.StatusDetail
		latestMCPServer.Status.Tools = statusUpdate.Status.Tools

		// Update the status
		updateErr = r.Status().Update(ctx, &latestMCPServer)
		if updateErr == nil {
			// Success - no need for more retries
			return nil
		}

		// If conflict, wait briefly and retry
		logger.Info("Status update conflict, retrying", "attempt", i+1, "error", updateErr)
		time.Sleep(time.Millisecond * 100)
	}

	// If we got here, we failed all retries
	logger.Error(updateErr, "Failed to update MCPServer status after retries")
	return updateErr
}

// Reconcile processes the MCPServer resource and establishes a connection to the MCP server
func (r *MCPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the MCPServer instance
	var mcpServer acp.MCPServer
	if err := r.Get(ctx, req.NamespacedName, &mcpServer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", mcpServer.Name)

	// Create a status update copy
	statusUpdate := mcpServer.DeepCopy()

	if statusUpdate.Spec.ApprovalContactChannel != nil {
		// validate the approval contact channel
		approvalContactChannel := &acp.ContactChannel{}
		err := r.Get(ctx, types.NamespacedName{Name: statusUpdate.Spec.ApprovalContactChannel.Name, Namespace: statusUpdate.Namespace}, approvalContactChannel)
		if err != nil {
			statusUpdate.Status.Connected = false
			statusUpdate.Status.Status = StatusError
			// todo handle other types of error, not just "not found"
			statusUpdate.Status.StatusDetail = fmt.Sprintf("ContactChannel %q not found", statusUpdate.Spec.ApprovalContactChannel.Name)
			r.recorder.Event(&mcpServer, corev1.EventTypeWarning, "ContactChannelNotFound", fmt.Sprintf("ContactChannel %q not found", statusUpdate.Spec.ApprovalContactChannel.Name))
			if err := r.updateStatus(ctx, req, statusUpdate); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		if !approvalContactChannel.Status.Ready {
			statusUpdate.Status.Connected = false
			statusUpdate.Status.Status = StatusPending
			statusUpdate.Status.StatusDetail = fmt.Sprintf("ContactChannel %q is not ready", statusUpdate.Spec.ApprovalContactChannel.Name)
			r.recorder.Event(&mcpServer, corev1.EventTypeWarning, "ContactChannelNotReady", fmt.Sprintf("ContactChannel %q is not ready", statusUpdate.Spec.ApprovalContactChannel.Name))
			if err := r.updateStatus(ctx, req, statusUpdate); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}
	}

	// Basic validation
	if err := r.validateMCPServer(&mcpServer); err != nil {
		statusUpdate.Status.Connected = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Validation failed: %v", err)
		r.recorder.Event(&mcpServer, corev1.EventTypeWarning, "ValidationFailed", err.Error())

		if updateErr := r.updateStatus(ctx, req, statusUpdate); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Try to connect to the MCP server
	err := r.MCPManager.ConnectServer(ctx, &mcpServer)
	if err != nil {
		statusUpdate.Status.Connected = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Connection failed: %v", err)
		r.recorder.Event(&mcpServer, corev1.EventTypeWarning, "ConnectionFailed", err.Error())

		if updateErr := r.updateStatus(ctx, req, statusUpdate); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil // Retry after 30 seconds
	}

	// Get tools from the manager
	tools, exists := r.MCPManager.GetTools(mcpServer.Name)
	if !exists {
		statusUpdate.Status.Connected = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = "Failed to get tools from manager"
		r.recorder.Event(&mcpServer, corev1.EventTypeWarning, "GetToolsFailed", "Failed to get tools from manager")

		if updateErr := r.updateStatus(ctx, req, statusUpdate); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil // Retry after 30 seconds
	}

	// Update status with tools
	statusUpdate.Status.Connected = true
	statusUpdate.Status.Status = StatusReady
	statusUpdate.Status.StatusDetail = fmt.Sprintf("Connected successfully with %d tools", len(tools))
	statusUpdate.Status.Tools = tools
	r.recorder.Event(&mcpServer, corev1.EventTypeNormal, "Connected", "MCP server connected successfully")

	// Update status
	if updateErr := r.updateStatus(ctx, req, statusUpdate); updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	logger.Info("Successfully reconciled MCPServer",
		"name", mcpServer.Name,
		"connected", statusUpdate.Status.Connected,
		"toolCount", len(statusUpdate.Status.Tools))

	// Schedule periodic reconciliation to refresh tool list
	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

// validateMCPServer performs basic validation on the MCPServer spec
func (r *MCPServerReconciler) validateMCPServer(mcpServer *acp.MCPServer) error {
	// Check server transport type
	if mcpServer.Spec.Transport != "stdio" && mcpServer.Spec.Transport != "http" {
		return fmt.Errorf("invalid server transport: %s", mcpServer.Spec.Transport)
	}

	// Validate stdio transport
	if mcpServer.Spec.Transport == "stdio" {
		if mcpServer.Spec.Command == "" {
			return fmt.Errorf("command is required for stdio servers")
		}
		// Other validations as needed
	}

	// Validate http transport
	if mcpServer.Spec.Transport == "http" {
		if mcpServer.Spec.URL == "" {
			return fmt.Errorf("url is required for http servers")
		}
		// Other validations as needed
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("mcpserver-controller")

	// Initialize the MCP manager if not already set
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManagerWithClient(r.Client)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.MCPServer{}).
		Complete(r)
}
