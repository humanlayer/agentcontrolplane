package mcpserver

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// StateMachine handles all MCPServer state transitions in one place
type StateMachine struct {
	client        client.Client
	recorder      record.EventRecorder
	mcpManager    MCPServerManagerInterface
	clientFactory MCPClientFactory
	envProcessor  EnvVarProcessor
}

// NewStateMachine creates a new state machine
func NewStateMachine(client client.Client, recorder record.EventRecorder, mcpManager MCPServerManagerInterface, clientFactory MCPClientFactory, envProcessor EnvVarProcessor) *StateMachine {
	return &StateMachine{
		client:        client,
		recorder:      recorder,
		mcpManager:    mcpManager,
		clientFactory: clientFactory,
		envProcessor:  envProcessor,
	}
}

// Process handles a MCPServer and returns the next action
func (sm *StateMachine) Process(ctx context.Context, mcpServer *acp.MCPServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Processing MCPServer", "name", mcpServer.Name, "status", mcpServer.Status.Status)

	// Determine current state
	state := mcpServer.Status.Status

	// Dispatch to handlers based on state
	switch state {
	case "":
		return sm.initialize(ctx, mcpServer)
	case StatusPending:
		return sm.validateAndConnect(ctx, mcpServer)
	case StatusError:
		return sm.handleError(ctx, mcpServer)
	case StatusReady:
		return sm.maintainConnection(ctx, mcpServer)
	default:
		// Unknown state - reset to initialization
		return sm.initialize(ctx, mcpServer)
	}
}

// State transition methods

func (sm *StateMachine) initialize(ctx context.Context, mcpServer *acp.MCPServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Initializing MCPServer", "name", mcpServer.Name)

	update := MCPServerStatusUpdate{
		Connected:    false,
		Status:       StatusPending,
		StatusDetail: "Initializing",
		EventType:    corev1.EventTypeNormal,
		EventReason:  "Initializing",
		EventMessage: "Starting MCPServer initialization",
	}

	if err := sm.updateMCPServerStatus(ctx, mcpServer, update); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

func (sm *StateMachine) validateAndConnect(ctx context.Context, mcpServer *acp.MCPServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Validate spec
	if err := validateMCPServerSpec(mcpServer.Spec); err != nil {
		return sm.handleValidationError(ctx, mcpServer, err)
	}

	// Validate contact channel if specified
	if err := validateContactChannelReference(ctx, sm.client, mcpServer); err != nil {
		// Check if it's a "not ready" error vs "not found" error
		if mcpServer.Spec.ApprovalContactChannel != nil {
			var contactChannel acp.ContactChannel
			if getErr := sm.client.Get(ctx, types.NamespacedName{
				Name:      mcpServer.Spec.ApprovalContactChannel.Name,
				Namespace: mcpServer.Namespace,
			}, &contactChannel); getErr == nil && !contactChannel.Status.Ready {
				// Contact channel exists but not ready - wait
				update := MCPServerStatusUpdate{
					Connected:    false,
					Status:       StatusPending,
					StatusDetail: fmt.Sprintf("ContactChannel %q is not ready", mcpServer.Spec.ApprovalContactChannel.Name),
					EventType:    corev1.EventTypeWarning,
					EventReason:  "ContactChannelNotReady",
					EventMessage: fmt.Sprintf("ContactChannel %q is not ready", mcpServer.Spec.ApprovalContactChannel.Name),
				}
				if updateErr := sm.updateMCPServerStatus(ctx, mcpServer, update); updateErr != nil {
					return ctrl.Result{}, updateErr
				}
				return ctrl.Result{RequeueAfter: time.Second * 5}, nil
			} else {
				// Contact channel not found - specific error
				logger := log.FromContext(ctx)
				update := MCPServerStatusUpdate{
					Connected:    false,
					Status:       StatusError,
					StatusDetail: fmt.Sprintf("Validation failed: %v", err),
					Error:        err.Error(),
					EventType:    corev1.EventTypeWarning,
					EventReason:  "ContactChannelNotFound",
					EventMessage: err.Error(),
				}
				if updateErr := sm.updateMCPServerStatus(ctx, mcpServer, update); updateErr != nil {
					logger.Error(updateErr, "Failed to update status")
					return ctrl.Result{}, updateErr
				}
				return ctrl.Result{}, err
			}
		}
		return sm.handleValidationError(ctx, mcpServer, err)
	}

	// All validation passed, try to connect
	err := sm.mcpManager.ConnectServer(ctx, mcpServer)
	if err != nil {
		return sm.handleConnectionError(ctx, mcpServer, err)
	}

	// Get tools from the manager
	tools, exists := sm.mcpManager.GetTools(mcpServer.Name)
	if !exists {
		err := fmt.Errorf("failed to get tools from manager")
		return sm.handleConnectionError(ctx, mcpServer, err)
	}

	// Success - update to ready state
	update := MCPServerStatusUpdate{
		Connected:    true,
		Status:       StatusReady,
		StatusDetail: fmt.Sprintf("Connected successfully with %d tools", len(tools)),
		Tools:        tools,
		EventType:    corev1.EventTypeNormal,
		EventReason:  "Connected",
		EventMessage: "MCP server connected successfully",
	}

	if err := sm.updateMCPServerStatus(ctx, mcpServer, update); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully connected MCPServer",
		"name", mcpServer.Name,
		"toolCount", len(tools))

	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

func (sm *StateMachine) maintainConnection(ctx context.Context, mcpServer *acp.MCPServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Simple health check - verify connection still exists
	connection, exists := sm.mcpManager.GetConnection(mcpServer.Name)
	if !exists || connection == nil {
		logger.Info("Connection lost, reconnecting")
		return sm.reconnectServer(ctx, mcpServer)
	}

	// Refresh tools and check for changes
	tools, toolsExist := sm.mcpManager.GetTools(mcpServer.Name)
	if !toolsExist {
		logger.Info("Tools unavailable, reconnecting")
		return sm.reconnectServer(ctx, mcpServer)
	}

	// Only update if tools actually changed
	if toolsChanged(mcpServer.Status.Tools, tools) {
		update := MCPServerStatusUpdate{
			Connected:    true,
			Status:       StatusReady,
			StatusDetail: fmt.Sprintf("Updated with %d tools", len(tools)),
			Tools:        tools,
			EventType:    corev1.EventTypeNormal,
			EventReason:  "ToolsUpdated",
			EventMessage: fmt.Sprintf("Tool list updated: %d tools", len(tools)),
		}

		if err := sm.updateMCPServerStatus(ctx, mcpServer, update); err != nil {
			logger.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}

		logger.Info("Updated MCPServer tools", "name", mcpServer.Name, "toolCount", len(tools))
	}

	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

// reconnectServer handles moving a server back to validation phase for reconnection
func (sm *StateMachine) reconnectServer(ctx context.Context, mcpServer *acp.MCPServer) (ctrl.Result, error) {
	update := MCPServerStatusUpdate{
		Connected:    false,
		Status:       StatusPending,
		StatusDetail: "Reconnecting",
		EventType:    corev1.EventTypeWarning,
		EventReason:  "ConnectionLost",
		EventMessage: "MCP server connection lost, reconnecting",
	}
	if err := sm.updateMCPServerStatus(ctx, mcpServer, update); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (sm *StateMachine) handleError(ctx context.Context, mcpServer *acp.MCPServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Attempting error recovery", "name", mcpServer.Name)

	// Move back to validation phase to retry
	update := MCPServerStatusUpdate{
		Connected:    false,
		Status:       StatusPending,
		StatusDetail: "Retrying after error",
		EventType:    corev1.EventTypeNormal,
		EventReason:  "Retrying",
		EventMessage: "Attempting to recover from error state",
	}

	if err := sm.updateMCPServerStatus(ctx, mcpServer, update); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// Helper methods

func (sm *StateMachine) updateMCPServerStatus(ctx context.Context, mcpServer *acp.MCPServer, update MCPServerStatusUpdate) error {
	// Fetch the latest version to avoid UID conflicts
	namespacedName := types.NamespacedName{Name: mcpServer.Name, Namespace: mcpServer.Namespace}
	latestMCPServer := &acp.MCPServer{}
	if err := sm.client.Get(ctx, namespacedName, latestMCPServer); err != nil {
		return err
	}

	latestMCPServer.Status.Connected = update.Connected
	latestMCPServer.Status.Status = update.Status
	latestMCPServer.Status.StatusDetail = update.StatusDetail
	latestMCPServer.Status.Tools = update.Tools

	if update.EventType != "" && update.EventReason != "" {
		sm.recorder.Event(latestMCPServer, update.EventType, update.EventReason, update.EventMessage)
	}

	return sm.client.Status().Update(ctx, latestMCPServer)
}

func (sm *StateMachine) handleValidationError(ctx context.Context, mcpServer *acp.MCPServer, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "Validation failed")

	update := MCPServerStatusUpdate{
		Connected:    false,
		Status:       StatusError,
		StatusDetail: fmt.Sprintf("Validation failed: %v", err),
		Error:        err.Error(),
		EventType:    corev1.EventTypeWarning,
		EventReason:  "ValidationFailed",
		EventMessage: err.Error(),
	}

	if updateErr := sm.updateMCPServerStatus(ctx, mcpServer, update); updateErr != nil {
		logger.Error(updateErr, "Failed to update status")
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{}, err // Don't retry validation errors
}

func (sm *StateMachine) handleConnectionError(ctx context.Context, mcpServer *acp.MCPServer, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "Connection failed")

	update := MCPServerStatusUpdate{
		Connected:    false,
		Status:       StatusError,
		StatusDetail: fmt.Sprintf("Connection failed: %v", err),
		Error:        err.Error(),
		EventType:    corev1.EventTypeWarning,
		EventReason:  "ConnectionFailed",
		EventMessage: err.Error(),
	}

	if updateErr := sm.updateMCPServerStatus(ctx, mcpServer, update); updateErr != nil {
		logger.Error(updateErr, "Failed to update status")
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil // Retry connection errors
}
