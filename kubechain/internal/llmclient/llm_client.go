package llmclient

import (
	"context"
	"fmt"

	"github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// LLMClient defines the interface for interacting with LLM providers
type LLMClient interface {
	// SendRequest sends a request to the LLM and returns the response
	SendRequest(ctx context.Context, messages []kubechainv1alpha1.Message, tools []Tool) (*kubechainv1alpha1.Message, error)
}

// LLMRequestError represents an error that occurred during an LLM request
// and includes HTTP status code information
type LLMRequestError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *LLMRequestError) Error() string {
	return fmt.Sprintf("LLM request failed with status %d: %s", e.StatusCode, e.Message)
}

func (e *LLMRequestError) Unwrap() error {
	return e.Err
}

// Tool represents a function that can be called by the LLM
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
	// KubechainToolType represents the Kubechain-specific type of tool (MCP, HumanContact)
	// This field is not sent to the LLM API but is used internally for tool identification
	KubechainToolType v1alpha1.ToolType `json:"-"`
}

// ToolFunction contains the function details
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  ToolFunctionParameters `json:"parameters"`
}

// ToolFunctionParameter defines a parameter type
type ToolFunctionParameter struct {
	Type string `json:"type"`
}

// ToolFunctionParameters defines the schema for the function parameters
type ToolFunctionParameters struct {
	Type       string                           `json:"type"`
	Properties map[string]ToolFunctionParameter `json:"properties"`
	Required   []string                         `json:"required,omitempty"`
}

// FromContactChannel creates a Tool from a ContactChannel resource
func ToolFromContactChannel(channel v1alpha1.ContactChannel) *Tool {
	// Create base parameters structure for human contact tools
	params := ToolFunctionParameters{
		Type: "object",
		Properties: map[string]ToolFunctionParameter{
			"message": {Type: "string"},
		},
		Required: []string{"message"},
	}

	var description string

	// Create the Tool
	return &Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        channel.Name,
			Description: description,
			Parameters:  params,
		},
		KubechainToolType: v1alpha1.ToolTypeHumanContact, // Set as HumanContact type
	}
}
