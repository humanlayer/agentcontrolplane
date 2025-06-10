package types

import (
	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// TaskStatusUpdate defines the parameters for updating task status
type TaskStatusUpdate struct {
	Ready        bool
	Status       acp.TaskStatusType
	Phase        acp.TaskPhase
	StatusDetail string
	Error        string
	EventType    string
	EventReason  string
	EventMessage string
}

// PhaseTransition represents a phase change with context
type PhaseTransition struct {
	From    acp.TaskPhase
	To      acp.TaskPhase
	Reason  string
	IsError bool
	Requeue bool
	Delay   *int // seconds, nil for immediate
}

// ErrorClassification categorizes errors for handling
type ErrorClassification struct {
	IsTerminal   bool
	ShouldRetry  bool
	ErrorType    string
	EventReason  string
	StatusUpdate TaskStatusUpdate
}

// ContextWindowBuildResult contains the result of context window building
type ContextWindowBuildResult struct {
	ContextWindow   []acp.Message
	UserMsgPreview  string
	ValidationError error
}

// ToolCallValidationResult contains the result of tool call validation
type ToolCallValidationResult struct {
	AllCompleted   bool
	ToolMessages   []acp.Message
	PendingCount   int
	CompletedCount int
	ErrorCount     int
}

// LLMRequestContext contains all context needed for an LLM request
type LLMRequestContext struct {
	ContextWindow []acp.Message
	Tools         []interface{} // Using interface{} to avoid import cycle
	Task          *acp.Task
	Agent         *acp.Agent
	LLM           acp.LLM
	APIKey        string
}
