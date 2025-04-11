package agent

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
)

const (
	StatusReady = "Ready"
	StatusError = "Error"
)

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=agents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=agents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=llms,verbs=get;list;watch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=mcpservers,verbs=get;list;watch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=contactchannels,verbs=get;list;watch

// AgentReconciler reconciles a Agent object
type AgentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	recorder   record.EventRecorder
	MCPManager *mcpmanager.MCPServerManager
}

// validateLLM checks if the referenced LLM exists and is ready
func (r *AgentReconciler) validateLLM(ctx context.Context, agent *acp.Agent) error {
	llm := &acp.LLM{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: agent.Namespace,
		Name:      agent.Spec.LLMRef.Name,
	}, llm)
	if err != nil {
		return fmt.Errorf("failed to get LLM %q: %w", agent.Spec.LLMRef.Name, err)
	}

	if llm.Status.Status != StatusReady {
		return fmt.Errorf("LLM %q is not ready", agent.Spec.LLMRef.Name)
	}

	return nil
}

// validateMCPServers checks if all referenced MCP servers exist and are connected
func (r *AgentReconciler) validateMCPServers(ctx context.Context, agent *acp.Agent) ([]acp.ResolvedMCPServer, error) {
	if r.MCPManager == nil {
		return nil, fmt.Errorf("MCPManager is not initialized")
	}

	validMCPServers := make([]acp.ResolvedMCPServer, 0, len(agent.Spec.MCPServers))

	for _, serverRef := range agent.Spec.MCPServers {
		mcpServer := &acp.MCPServer{}
		err := r.Get(ctx, client.ObjectKey{
			Namespace: agent.Namespace,
			Name:      serverRef.Name,
		}, mcpServer)
		if err != nil {
			return validMCPServers, fmt.Errorf("failed to get MCPServer %q: %w", serverRef.Name, err)
		}

		if !mcpServer.Status.Connected {
			return validMCPServers, fmt.Errorf("MCPServer %q is not connected", serverRef.Name)
		}

		// TODO(dex) why don't we just pull the tools off the MCPServer Status - Agent shouldn't know too much about mcp impl
		tools, exists := r.MCPManager.GetTools(mcpServer.Name)
		if !exists {
			return validMCPServers, fmt.Errorf("failed to get tools for MCPServer %q", mcpServer.Name)
		}

		// Create list of tool names
		toolNames := make([]string, 0, len(tools))
		for _, tool := range tools {
			toolNames = append(toolNames, tool.Name)
		}

		validMCPServers = append(validMCPServers, acp.ResolvedMCPServer{
			Name:  serverRef.Name,
			Tools: toolNames,
		})
	}

	return validMCPServers, nil
}

// validateHumanContactChannels checks if all referenced contact channels exist and are ready
// and have the required context information for the LLM
func (r *AgentReconciler) validateHumanContactChannels(ctx context.Context, agent *acp.Agent) ([]acp.ResolvedContactChannel, error) {
	validChannels := make([]acp.ResolvedContactChannel, 0, len(agent.Spec.HumanContactChannels))

	for _, channelRef := range agent.Spec.HumanContactChannels {
		channel := &acp.ContactChannel{}
		err := r.Get(ctx, client.ObjectKey{
			Namespace: agent.Namespace,
			Name:      channelRef.Name,
		}, channel)
		if err != nil {
			return validChannels, fmt.Errorf("failed to get ContactChannel %q: %w", channelRef.Name, err)
		}

		if !channel.Status.Ready {
			return validChannels, fmt.Errorf("ContactChannel %q is not ready", channelRef.Name)
		}

		// Check that the context about the user/channel is provided based on the channel type
		switch channel.Spec.Type {
		case acp.ContactChannelTypeEmail:
			if channel.Spec.Email == nil {
				return validChannels, fmt.Errorf("ContactChannel %q is missing Email configuration", channelRef.Name)
			}
			if channel.Spec.Email.ContextAboutUser == "" {
				return validChannels, fmt.Errorf("ContactChannel %q must have ContextAboutUser set", channelRef.Name)
			}
		case acp.ContactChannelTypeSlack:
			if channel.Spec.Slack == nil {
				return validChannels, fmt.Errorf("ContactChannel %q is missing Slack configuration", channelRef.Name)
			}
			if channel.Spec.Slack.ContextAboutChannelOrUser == "" {
				return validChannels, fmt.Errorf("ContactChannel %q must have ContextAboutChannelOrUser set", channelRef.Name)
			}
		default:
			return validChannels, fmt.Errorf("ContactChannel %q has unsupported type %q", channelRef.Name, channel.Spec.Type)
		}

		validChannels = append(validChannels, acp.ResolvedContactChannel{
			Name: channelRef.Name,
			Type: string(channel.Spec.Type),
		})
	}

	return validChannels, nil
}

//nolint:unparam
func (r *AgentReconciler) setStatusError(ctx context.Context, agent *acp.Agent, err error, statusUpdate *acp.Agent, reason string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	statusUpdate.Status.Ready = false
	statusUpdate.Status.Status = StatusError
	statusUpdate.Status.StatusDetail = err.Error()
	r.recorder.Event(agent, corev1.EventTypeWarning, reason, err.Error())

	if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
		logger.Error(updateErr, "Failed to update Agent status")
		return ctrl.Result{}, fmt.Errorf("failed to update agent status: %v", err)
	}

	return ctrl.Result{}, err
}

// Reconcile validates the agent's LLM and Tool references
func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var agent acp.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", agent.Name)

	// Create a copy for status update
	statusUpdate := agent.DeepCopy()

	// Initialize status if not set
	if statusUpdate.Status.Status == "" {
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.StatusDetail = "Validating dependencies"
		r.recorder.Event(&agent, corev1.EventTypeNormal, "Initializing", "Starting validation")
	}

	// Initialize empty valid tools, servers, and human contact channels slices
	validMCPServers := make([]acp.ResolvedMCPServer, 0)
	validHumanContactChannels := make([]acp.ResolvedContactChannel, 0)

	statusUpdate.Status.ValidMCPServers = validMCPServers
	statusUpdate.Status.ValidHumanContactChannels = validHumanContactChannels

	// Validate LLM reference
	if err := r.validateLLM(ctx, &agent); err != nil {
		logger.Error(err, "LLM validation failed")
		return r.setStatusError(ctx, &agent, err, statusUpdate, "ValidationFailed")
	}

	var err error

	// Validate MCP server references, if any
	if len(agent.Spec.MCPServers) > 0 && r.MCPManager != nil {
		validMCPServers, err = r.validateMCPServers(ctx, &agent)
		if err != nil {
			logger.Error(err, "MCP server validation failed")
			return r.setStatusError(ctx, &agent, err, statusUpdate, "ValidationFailed")
		}

		statusUpdate.Status.ValidMCPServers = validMCPServers
	}

	// Validate HumanContactChannel references, if any
	if len(agent.Spec.HumanContactChannels) > 0 {
		validHumanContactChannels, err = r.validateHumanContactChannels(ctx, &agent)
		if err != nil {
			logger.Error(err, "HumanContactChannel validation failed")
			return r.setStatusError(ctx, &agent, err, statusUpdate, "ValidationFailed")
		}

		statusUpdate.Status.ValidHumanContactChannels = validHumanContactChannels
	}

	// All validations passed
	statusUpdate.Status.Ready = true
	statusUpdate.Status.Status = StatusReady
	statusUpdate.Status.StatusDetail = "All dependencies validated successfully"

	r.recorder.Event(&agent, corev1.EventTypeNormal, "ValidationSucceeded", "All dependencies validated successfully")

	// Update status
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update Agent status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled agent",
		"name", agent.Name,
		"ready", statusUpdate.Status.Ready,
		"status", statusUpdate.Status.Status,
		"validHumanContactChannels", statusUpdate.Status.ValidHumanContactChannels)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("agent-controller")

	// Initialize MCPManager if not already set
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManager()
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.Agent{}).
		Complete(r)
}
