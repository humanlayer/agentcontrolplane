package toolcall

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayer"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ToolExecutor handles all tool execution logic in one place
type ToolExecutor struct {
	client     client.Client
	mcpManager mcpmanager.MCPManagerInterface
	hlFactory  humanlayer.HumanLayerClientFactory
}

// NewToolExecutor creates a unified tool executor
func NewToolExecutor(client client.Client, mcpManager mcpmanager.MCPManagerInterface, hlFactory humanlayer.HumanLayerClientFactory) *ToolExecutor {
	return &ToolExecutor{
		client:     client,
		mcpManager: mcpManager,
		hlFactory:  hlFactory,
	}
}

// Execute handles all tool execution types in one method
func (e *ToolExecutor) Execute(ctx context.Context, tc *acp.ToolCall) (string, error) {
	// Parse arguments once
	args, err := e.parseArguments(tc.Spec.Arguments)
	if err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Route to appropriate executor based on tool type
	switch tc.Spec.ToolType {
	case acp.ToolTypeMCP:
		return e.executeMCPTool(ctx, tc, args)
	case acp.ToolTypeDelegateToAgent:
		return e.executeDelegateToAgent(ctx, tc, args)
	case acp.ToolTypeHumanContact:
		return e.executeHumanContact(ctx, tc, args)
	default:
		return "", fmt.Errorf("unsupported tool type: %s", tc.Spec.ToolType)
	}
}

// CheckApprovalRequired determines if a tool needs human approval
func (e *ToolExecutor) CheckApprovalRequired(ctx context.Context, tc *acp.ToolCall) (bool, *acp.ContactChannel, error) {
	if tc.Spec.ToolType != acp.ToolTypeMCP {
		return false, nil, nil
	}

	serverName := e.extractServerName(tc.Spec.ToolRef.Name)
	var mcpServer acp.MCPServer
	if err := e.client.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: serverName}, &mcpServer); err != nil {
		return false, nil, err
	}

	if mcpServer.Spec.ApprovalContactChannel == nil {
		return false, nil, nil
	}

	// Get contact channel
	var contactChannel acp.ContactChannel
	if err := e.client.Get(ctx, client.ObjectKey{
		Namespace: tc.Namespace,
		Name:      mcpServer.Spec.ApprovalContactChannel.Name,
	}, &contactChannel); err != nil {
		return false, nil, err
	}

	return true, &contactChannel, nil
}

// RequestApproval sends approval request via HumanLayer
func (e *ToolExecutor) RequestApproval(ctx context.Context, tc *acp.ToolCall, contactChannel *acp.ContactChannel) (string, error) {
	apiKey, err := e.getAPIKey(ctx, contactChannel, tc.Namespace)
	if err != nil {
		return "", err
	}

	client := e.hlFactory.NewHumanLayerClient()
	e.configureContactChannel(client, contactChannel)

	args, _ := e.parseArguments(tc.Spec.Arguments)
	client.SetFunctionCallSpec(tc.Spec.ToolRef.Name, args)
	client.SetRunID(tc.Name)
	client.SetAPIKey(apiKey)

	functionCall, _, err := client.RequestApproval(ctx)
	if err != nil {
		return "", err
	}

	return functionCall.GetCallId(), nil
}

// CheckApprovalStatus checks if approval is complete
func (e *ToolExecutor) CheckApprovalStatus(ctx context.Context, callID string, contactChannel *acp.ContactChannel, namespace string) (*humanlayerapi.FunctionCallOutput, error) {
	apiKey, err := e.getAPIKey(ctx, contactChannel, namespace)
	if err != nil {
		return nil, err
	}

	client := e.hlFactory.NewHumanLayerClient()
	client.SetCallID(callID)
	client.SetAPIKey(apiKey)

	functionCall, _, err := client.GetFunctionCallStatus(ctx)
	return functionCall, err
}

// Internal helper methods

func (e *ToolExecutor) parseArguments(argsJSON string) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, err
	}
	return args, nil
}

func (e *ToolExecutor) extractServerName(toolRefName string) string {
	parts := strings.Split(toolRefName, "__")
	if len(parts) >= 2 {
		return parts[0]
	}
	return toolRefName
}

func (e *ToolExecutor) extractToolName(toolRefName string) string {
	parts := strings.Split(toolRefName, "__")
	if len(parts) >= 2 {
		return parts[1]
	}
	return toolRefName
}

func (e *ToolExecutor) executeMCPTool(ctx context.Context, tc *acp.ToolCall, args map[string]interface{}) (string, error) {
	serverName := e.extractServerName(tc.Spec.ToolRef.Name)
	toolName := e.extractToolName(tc.Spec.ToolRef.Name)

	result, err := e.mcpManager.CallTool(ctx, serverName, toolName, args)
	if err != nil {
		return "", fmt.Errorf("MCP tool execution failed: %w", err)
	}

	return result, nil
}

func (e *ToolExecutor) executeDelegateToAgent(ctx context.Context, tc *acp.ToolCall, args map[string]interface{}) (string, error) {
	message, ok := args["message"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'message' argument")
	}

	agentName := e.extractToolName(tc.Spec.ToolRef.Name) // Extract agent name from "delegate_to_agent__agentName"

	// Create child task with idempotent creation
	childTaskName := fmt.Sprintf("delegate-%s-%s", tc.Name, agentName)
	if len(childTaskName) > 63 {
		childTaskName = childTaskName[:55] + "-" + childTaskName[len(childTaskName)-7:]
	}

	// First, check if a task with this name already exists
	existingTask := &acp.Task{}
	if err := e.client.Get(ctx, client.ObjectKey{
		Name:      childTaskName,
		Namespace: tc.Namespace,
	}, existingTask); err == nil {
		// Task exists, check if it's our child task
		if parentToolCall, exists := existingTask.Labels["acp.humanlayer.dev/parent-toolcall"]; exists && parentToolCall == tc.Name {
			log.FromContext(ctx).Info("Found existing child task for sub-agent", "childTaskName", existingTask.Name, "agentName", agentName)
			return fmt.Sprintf("Delegated to agent %s via task %s", agentName, existingTask.Name), nil
		}
		// Task exists but not our child - this shouldn't happen in normal operation
		return "", fmt.Errorf("task %s already exists but is not a child of this toolcall", childTaskName)
	}

	// Task doesn't exist, create it
	childTask := &acp.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      childTaskName,
			Namespace: tc.Namespace,
			Labels: map[string]string{
				"acp.humanlayer.dev/parent-toolcall": tc.Name,
			},
		},
		Spec: acp.TaskSpec{
			AgentRef: acp.LocalObjectReference{
				Name: agentName,
			},
			UserMessage: message,
		},
	}

	if err := e.client.Create(ctx, childTask); err != nil {
		// Handle race condition - task might have been created between our check and create
		if strings.Contains(err.Error(), "already exists") {
			// Try to get the task that was created concurrently
			if getErr := e.client.Get(ctx, client.ObjectKey{
				Name:      childTaskName,
				Namespace: tc.Namespace,
			}, existingTask); getErr == nil {
				// Verify it's our child task
				if parentToolCall, exists := existingTask.Labels["acp.humanlayer.dev/parent-toolcall"]; exists && parentToolCall == tc.Name {
					log.FromContext(ctx).Info("Concurrent creation resolved - using existing child task", "childTaskName", existingTask.Name, "agentName", agentName)
					return fmt.Sprintf("Delegated to agent %s via task %s", agentName, existingTask.Name), nil
				}
			}
		}
		return "", fmt.Errorf("failed to create child task: %w", err)
	}

	log.FromContext(ctx).Info("Created child task for sub-agent", "childTaskName", childTask.Name, "agentName", agentName)
	return fmt.Sprintf("Delegated to agent %s via task %s", agentName, childTask.Name), nil
}

func (e *ToolExecutor) executeHumanContact(ctx context.Context, tc *acp.ToolCall, args map[string]interface{}) (string, error) {
	channelName := e.extractServerName(tc.Spec.ToolRef.Name) // Extract channel from "CHANNEL__toolname"

	var contactChannel acp.ContactChannel
	if err := e.client.Get(ctx, client.ObjectKey{
		Namespace: tc.Namespace,
		Name:      channelName,
	}, &contactChannel); err != nil {
		return "", fmt.Errorf("failed to get contact channel: %w", err)
	}

	// Extract message from arguments
	message, ok := args["message"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'message' argument")
	}

	apiKey, err := e.getAPIKey(ctx, &contactChannel, tc.Namespace)
	if err != nil {
		return "", err
	}

	client := e.hlFactory.NewHumanLayerClient()
	e.configureContactChannel(client, &contactChannel)
	client.SetRunID(tc.Name)
	client.SetCallID(tc.Spec.ToolCallID)
	client.SetAPIKey(apiKey)

	humanContact, _, err := client.RequestHumanContact(ctx, message)
	if err != nil {
		return "", fmt.Errorf("human contact request failed: %w", err)
	}

	return fmt.Sprintf("Human contact requested, call ID: %s", humanContact.GetCallId()), nil
}

func (e *ToolExecutor) getAPIKey(ctx context.Context, contactChannel *acp.ContactChannel, namespace string) (string, error) {
	// Determine which authentication method to use
	var apiKeySource *acp.APIKeySource
	if contactChannel.Spec.ChannelAPIKeyFrom != nil {
		apiKeySource = contactChannel.Spec.ChannelAPIKeyFrom
	} else if contactChannel.Spec.APIKeyFrom != nil {
		apiKeySource = contactChannel.Spec.APIKeyFrom
	} else {
		return "", fmt.Errorf("no API key source configured")
	}

	var secret corev1.Secret
	if err := e.client.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      apiKeySource.SecretKeyRef.Name,
	}, &secret); err != nil {
		return "", fmt.Errorf("failed to get API key secret: %w", err)
	}

	apiKey, exists := secret.Data[apiKeySource.SecretKeyRef.Key]
	if !exists {
		return "", fmt.Errorf("API key not found in secret")
	}

	return string(apiKey), nil
}

func (e *ToolExecutor) configureContactChannel(client humanlayer.HumanLayerClientWrapper, contactChannel *acp.ContactChannel) {
	// Set channel ID if using channel-specific authentication
	if contactChannel.Spec.ChannelID != "" {
		client.SetChannelID(contactChannel.Spec.ChannelID)
	}

	// Set channel configuration for traditional authentication or as fallback
	switch contactChannel.Spec.Type {
	case acp.ContactChannelTypeSlack:
		if contactChannel.Spec.Slack != nil {
			client.SetSlackConfig(contactChannel.Spec.Slack)
		}
	case acp.ContactChannelTypeEmail:
		if contactChannel.Spec.Email != nil {
			client.SetEmailConfig(contactChannel.Spec.Email)
		}
	}
}
