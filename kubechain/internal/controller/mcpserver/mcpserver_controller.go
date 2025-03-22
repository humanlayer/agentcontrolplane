package mcpserver

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/mcpmanager"
)

const (
	StatusError = "Error"
)

// MCPServerManagerInterface defines the interface for MCP server management
type MCPServerManagerInterface interface {
	ConnectServer(ctx context.Context, mcpServer *kubechainv1alpha1.MCPServer) error
	GetTools(serverName string) ([]kubechainv1alpha1.MCPTool, bool)
	GetConnection(serverName string) (*mcpmanager.MCPConnection, bool)
	DisconnectServer(serverName string)
	GetToolsForAgent(agent *kubechainv1alpha1.Agent) []kubechainv1alpha1.MCPTool
	CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (string, error)
	FindServerForTool(fullToolName string) (serverName string, toolName string, found bool)
	Close()
}

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=mcpservers/status,verbs=get;update;patch

// MCPServerReconciler reconciles a MCPServer object
type MCPServerReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	recorder   record.EventRecorder
	MCPManager MCPServerManagerInterface
}

// updateStatus updates the status of the MCPServer resource with the latest version
func (r *MCPServerReconciler) updateStatus(ctx context.Context, req ctrl.Request, statusUpdate *kubechainv1alpha1.MCPServer) error {
	logger := log.FromContext(ctx)

	// Get the latest version of the MCPServer
	var latestMCPServer kubechainv1alpha1.MCPServer
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
	if err := r.Status().Update(ctx, &latestMCPServer); err != nil {
		logger.Error(err, "Failed to update MCPServer status")
		return err
	}

	return nil
}

// Reconcile processes the MCPServer resource and establishes a connection to the MCP server
func (r *MCPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the MCPServer instance
	var mcpServer kubechainv1alpha1.MCPServer
	if err := r.Get(ctx, req.NamespacedName, &mcpServer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", mcpServer.Name)

	// Create a status update copy
	statusUpdate := mcpServer.DeepCopy()

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
	statusUpdate.Status.Status = "Ready"
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
func (r *MCPServerReconciler) validateMCPServer(mcpServer *kubechainv1alpha1.MCPServer) error {
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
		For(&kubechainv1alpha1.MCPServer{}).
		Complete(r)
}
