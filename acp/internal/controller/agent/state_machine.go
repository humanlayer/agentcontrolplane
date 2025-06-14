package agent

import (
	"context"
	"fmt"
	"time"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// StateMachine handles all Agent state transitions in one place
type StateMachine struct {
	client   client.Client
	recorder record.EventRecorder
}

// NewStateMachine creates a new state machine
func NewStateMachine(client client.Client, recorder record.EventRecorder) *StateMachine {
	return &StateMachine{
		client:   client,
		recorder: recorder,
	}
}

// Process handles an Agent and returns the next action
func (sm *StateMachine) Process(ctx context.Context, agent *acp.Agent) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("agent-state-machine")
	logger.Info("Processing Agent", "name", agent.Name, "status", agent.Status.Status)

	// Determine current state
	state := sm.getAgentState(agent)

	// Dispatch to handlers based on state
	switch state {
	case "":
		return sm.initialize(ctx, agent)
	case string(acp.AgentStatusPending):
		return sm.handlePending(ctx, agent)
	case string(acp.AgentStatusReady):
		return sm.handleReady(ctx, agent)
	case string(acp.AgentStatusError):
		return sm.handleError(ctx, agent)
	default:
		return sm.initialize(ctx, agent) // Default to initialization
	}
}

// getAgentState determines the current state of the agent
func (sm *StateMachine) getAgentState(agent *acp.Agent) string {
	if agent.Status.Status == "" {
		return "" // Return empty for empty status
	}
	return string(agent.Status.Status)
}

// State transition methods

// initialize handles empty status -> "Pending" and continues with validation
func (sm *StateMachine) initialize(ctx context.Context, agent *acp.Agent) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("agent-state-machine")
	logger.Info("Initializing agent", "agent", agent.Name)

	// Initialize status if needed
	if agent.Status.Status == "" {
		sm.recorder.Event(agent, "Normal", "Initializing", "Starting validation")
		if err := sm.updateStatus(ctx, agent, acp.AgentStatusPending, "Validating dependencies", nil); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Continue with validation to match original controller behavior
	return sm.validateDependencies(ctx, agent)
}

// handlePending validates dependencies for "Pending" -> "Ready"/"Error"
func (sm *StateMachine) handlePending(ctx context.Context, agent *acp.Agent) (ctrl.Result, error) {
	// This will contain the existing validateDependencies logic
	return sm.validateDependencies(ctx, agent)
}

// validateDependencies contains the main dependency validation logic
func (sm *StateMachine) validateDependencies(ctx context.Context, agent *acp.Agent) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("agent-state-machine")
	logger.Info("Starting agent validation", "agent", agent.Name)

	// Step 1: Validate LLM
	if err := sm.validateLLM(ctx, agent); err != nil {
		return sm.handleValidationFailed(ctx, agent, err, "LLM validation failed")
	}

	// Step 2: Validate sub-agents (if any)
	var validSubAgents []acp.ResolvedSubAgent
	if len(agent.Spec.SubAgents) > 0 {
		ready, message, subAgents := sm.validateSubAgents(ctx, agent)
		if !ready {
			sm.recorder.Event(agent, "Normal", "SubAgentsPending", message)
			if err := sm.updateStatus(ctx, agent, acp.AgentStatusPending, message, nil); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		validSubAgents = subAgents
	}

	// Step 3: Validate MCP servers (if any)
	var validMCPServers []acp.ResolvedMCPServer
	if len(agent.Spec.MCPServers) > 0 {
		servers, err := sm.validateMCPServers(ctx, agent)
		if err != nil {
			return sm.handleValidationFailed(ctx, agent, err, "MCP server validation failed")
		}
		validMCPServers = servers
	}

	// Step 4: Validate contact channels (if any)
	var validContactChannels []acp.ResolvedContactChannel
	if len(agent.Spec.HumanContactChannels) > 0 {
		channels, err := sm.validateContactChannels(ctx, agent)
		if err != nil {
			return sm.handleValidationFailed(ctx, agent, err, "Contact channel validation failed")
		}
		validContactChannels = channels
	}

	// All validations passed - set to Ready
	resources := map[string]interface{}{
		"mcpServers":      validMCPServers,
		"contactChannels": validContactChannels,
		"subAgents":       validSubAgents,
	}

	sm.recorder.Event(agent, "Normal", "ValidationSucceeded", "All dependencies validated successfully")
	if err := sm.updateStatus(ctx, agent, acp.AgentStatusReady, "All dependencies validated successfully", resources); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Agent validation completed", "agent", agent.Name, "status", "Ready")
	return ctrl.Result{}, nil
}

// handleReady processes agents in ready state
func (sm *StateMachine) handleReady(ctx context.Context, agent *acp.Agent) (ctrl.Result, error) {
	// Agent is ready, no action needed
	return ctrl.Result{}, nil
}

// handleError processes agents in error state for recovery
func (sm *StateMachine) handleError(ctx context.Context, agent *acp.Agent) (ctrl.Result, error) {
	// Could implement retry logic here if needed
	return ctrl.Result{}, nil
}

// Helper methods

// updateStatus updates agent status with retry logic for conflicts
func (sm *StateMachine) updateStatus(ctx context.Context, agent *acp.Agent, status acp.AgentStatusType, message string, resources map[string]interface{}) error {
	for i := 0; i < 3; i++ {
		// Get latest version
		var current acp.Agent
		if err := sm.client.Get(ctx, types.NamespacedName{Namespace: agent.Namespace, Name: agent.Name}, &current); err != nil {
			return err
		}

		// Update status
		current.Status.Status = status
		current.Status.StatusDetail = message
		current.Status.Ready = (status == acp.AgentStatusReady)

		// Set resolved resources if provided
		if resources != nil {
			if mcpServers, ok := resources["mcpServers"].([]acp.ResolvedMCPServer); ok {
				current.Status.ValidMCPServers = mcpServers
			}
			if contactChannels, ok := resources["contactChannels"].([]acp.ResolvedContactChannel); ok {
				current.Status.ValidHumanContactChannels = contactChannels
			}
			if subAgents, ok := resources["subAgents"].([]acp.ResolvedSubAgent); ok {
				current.Status.ValidSubAgents = subAgents
			}
		} else {
			// Clear resolved resources for non-ready states
			current.Status.ValidMCPServers = nil
			current.Status.ValidHumanContactChannels = nil
			current.Status.ValidSubAgents = nil
		}

		// Try to update
		if err := sm.client.Status().Update(ctx, &current); err != nil {
			if apierrors.IsConflict(err) && i < 2 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}
		return nil
	}
	return apierrors.NewConflict(acp.GroupVersion.WithResource("agents").GroupResource(), agent.Name, nil)
}

// validateLLM checks if the referenced LLM exists and is ready
func (sm *StateMachine) validateLLM(ctx context.Context, agent *acp.Agent) error {
	var llm acp.LLM
	if err := sm.client.Get(ctx, types.NamespacedName{Namespace: agent.Namespace, Name: agent.Spec.LLMRef.Name}, &llm); err != nil {
		return fmt.Errorf("failed to get LLM %q: %w", agent.Spec.LLMRef.Name, err)
	}
	if llm.Status.Status != "Ready" {
		return fmt.Errorf("LLM %q is not ready (status: %q)", agent.Spec.LLMRef.Name, llm.Status.Status)
	}
	return nil
}

// validateSubAgents validates all sub-agent references
func (sm *StateMachine) validateSubAgents(ctx context.Context, agent *acp.Agent) (bool, string, []acp.ResolvedSubAgent) {
	validSubAgents := make([]acp.ResolvedSubAgent, 0, len(agent.Spec.SubAgents))
	for _, ref := range agent.Spec.SubAgents {
		var subAgent acp.Agent
		if err := sm.client.Get(ctx, types.NamespacedName{Namespace: agent.Namespace, Name: ref.Name}, &subAgent); err != nil {
			return false, fmt.Sprintf("waiting for sub-agent %q (not found)", ref.Name), validSubAgents
		}
		if !subAgent.Status.Ready {
			return false, fmt.Sprintf("waiting for sub-agent %q (not ready)", ref.Name), validSubAgents
		}
		validSubAgents = append(validSubAgents, acp.ResolvedSubAgent(ref))
	}
	return true, "", validSubAgents
}

// validateMCPServers validates all MCP server references
func (sm *StateMachine) validateMCPServers(ctx context.Context, agent *acp.Agent) ([]acp.ResolvedMCPServer, error) {
	validServers := make([]acp.ResolvedMCPServer, 0, len(agent.Spec.MCPServers))
	for _, ref := range agent.Spec.MCPServers {
		var server acp.MCPServer
		if err := sm.client.Get(ctx, types.NamespacedName{Namespace: agent.Namespace, Name: ref.Name}, &server); err != nil {
			return nil, fmt.Errorf("failed to get MCPServer %q: %w", ref.Name, err)
		}
		if !server.Status.Connected {
			return nil, fmt.Errorf("MCPServer %q is not connected", ref.Name)
		}

		toolNames := make([]string, len(server.Status.Tools))
		for i, tool := range server.Status.Tools {
			toolNames[i] = tool.Name
		}

		validServers = append(validServers, acp.ResolvedMCPServer{
			Name:  ref.Name,
			Tools: toolNames,
		})
	}
	return validServers, nil
}

// validateContactChannels validates all contact channel references
func (sm *StateMachine) validateContactChannels(ctx context.Context, agent *acp.Agent) ([]acp.ResolvedContactChannel, error) {
	validChannels := make([]acp.ResolvedContactChannel, 0, len(agent.Spec.HumanContactChannels))
	for _, ref := range agent.Spec.HumanContactChannels {
		var channel acp.ContactChannel
		if err := sm.client.Get(ctx, types.NamespacedName{Namespace: agent.Namespace, Name: ref.Name}, &channel); err != nil {
			return nil, fmt.Errorf("failed to get ContactChannel %q: %w", ref.Name, err)
		}
		if !channel.Status.Ready {
			return nil, fmt.Errorf("ContactChannel %q is not ready", ref.Name)
		}

		validChannels = append(validChannels, acp.ResolvedContactChannel{
			Name: ref.Name,
			Type: string(channel.Spec.Type),
		})
	}
	return validChannels, nil
}

// handleValidationFailed handles validation errors with appropriate retry logic
func (sm *StateMachine) handleValidationFailed(ctx context.Context, agent *acp.Agent, err error, reason string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, reason)

	sm.recorder.Event(agent, "Warning", "ValidationFailed", err.Error())

	// Determine if this is a retryable error
	isRetryable := sm.isRetryableError(err)

	if isRetryable {
		// Set status to Pending for retryable errors
		if updateErr := sm.updateStatus(ctx, agent, acp.AgentStatusPending, err.Error(), nil); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Set status to Error for non-retryable errors
	if updateErr := sm.updateStatus(ctx, agent, acp.AgentStatusError, err.Error(), nil); updateErr != nil {
		logger.Error(updateErr, "Failed to update status")
	}
	return ctrl.Result{}, err
}

// isRetryableError determines if an error should trigger a retry
func (sm *StateMachine) isRetryableError(err error) bool {
	return !apierrors.IsNotFound(err)
}
