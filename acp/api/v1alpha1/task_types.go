package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Message roles
const (
	MessageRoleSystem    = "system"
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
	MessageRoleTool      = "tool"
)

// ValidMessageRoles provides a map of valid message roles for validation
var ValidMessageRoles = map[string]bool{
	MessageRoleSystem:    true,
	MessageRoleUser:      true,
	MessageRoleAssistant: true,
	MessageRoleTool:      true,
}

// TaskSpec defines the desired state of Task
type TaskSpec struct {
	// AgentRef references the agent that will execute this Task.
	// +kubebuilder:validation:Required
	AgentRef LocalObjectReference `json:"agentRef"`

	// UserMessage is the message to send to the agent.
	// If provided, userMessage will be used and contextWindow must be empty.
	// +optional
	UserMessage string `json:"userMessage,omitempty"`

	// ContextWindow provides the initial conversation context when creating a Task.
	// If provided, contextWindow will be used and userMessage must be empty.
	// This will be copied to status.ContextWindow, which is the source of truth
	// for the ongoing conversation.
	// +optional
	ContextWindow []Message `json:"contextWindow,omitempty"`

	// BaseURL specifies the base URL for the human contact channel.
	// +optional
	BaseURL string `json:"baseURL,omitempty"`

	// ChannelTokenFrom references a secret containing the token for the human contact channel.
	// +optional
	ChannelTokenFrom *SecretKeyRef `json:"channelTokenFrom,omitempty"`
}

// Message represents a single message in the conversation
type Message struct {
	// Role is the role of the message sender (system, user, assistant, tool)
	// +kubebuilder:validation:Enum=system;user;assistant;tool
	Role string `json:"role"`

	// Content is the message content
	Content string `json:"content"`

	// ToolCalls contains any tool calls requested by this message
	// +optional
	ToolCalls []MessageToolCall `json:"toolCalls,omitempty"`

	// ToolCallID is the unique identifier for this tool call
	// +optional
	ToolCallID string `json:"toolCallId,omitempty"`

	// Name is the name of the tool that was called
	// +optional
	Name string `json:"name,omitempty"`
}

// ToolCall represents a request to call a tool
type MessageToolCall struct {
	// ID is the unique identifier for this tool call
	ID string `json:"id"`

	// Function contains the details of the function to call
	Function ToolCallFunction `json:"function"`

	// Type indicates the type of tool call. Currently only "function" is supported.
	Type string `json:"type"`
}

// ToolCallFunction contains the function details for a tool call
type ToolCallFunction struct {
	// Name is the name of the function to call
	Name string `json:"name"`

	// Arguments contains the arguments to pass to the function in JSON format
	Arguments string `json:"arguments"`
}

// SpanContext contains OpenTelemetry span context information
type SpanContext struct {
	// TraceID is the trace ID for the span
	TraceID string `json:"traceID,omitempty"`

	// SpanID is the span ID
	SpanID string `json:"spanID,omitempty"`
}

// TaskStatus defines the observed state of Task
type TaskStatus struct {
	// Phase indicates the current phase of the Task
	// +optional
	Phase TaskPhase `json:"phase,omitempty"`

	// Ready indicates if the Task is ready to be executed
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Status indicates the current status of the task
	// +kubebuilder:validation:Enum=Ready;Error;Pending
	Status TaskStatusType `json:"status,omitempty"`

	// StatusDetail provides additional details about the current status
	StatusDetail string `json:"statusDetail,omitempty"`

	// StartTime is when the Task started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the Task completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Output contains the result of the task execution
	// +optional
	Output string `json:"output,omitempty"`

	// ContextWindow maintains the conversation history as a sequence of messages
	// +optional
	ContextWindow []Message `json:"contextWindow,omitempty"`

	// MessageCount contains the number of messages in the context window
	// +optional
	MessageCount int `json:"messageCount,omitempty"`

	// UserMsgPreview stores the first 50 characters of the user's message
	// +optional
	UserMsgPreview string `json:"userMsgPreview,omitempty"`

	// Error message if the task failed
	// +optional
	Error string `json:"error,omitempty"`

	// SpanContext contains OpenTelemetry span context information
	// +optional
	SpanContext *SpanContext `json:"spanContext,omitempty"`

	// ToolCallRequestID uniquely identifies a set of tool calls from a single LLM response
	// +optional
	ToolCallRequestID string `json:"toolCallRequestId,omitempty"`
}

type TaskStatusType string

const (
	TaskStatusTypeReady   TaskStatusType = "Ready"
	TaskStatusTypeError   TaskStatusType = "Error"
	TaskStatusTypePending TaskStatusType = "Pending"
)

// TaskPhase represents the phase of a Task
// +kubebuilder:validation:Enum=Initializing;Pending;ReadyForLLM;SendContextWindowToLLM;ToolCallsPending;CheckingToolCalls;FinalAnswer;ErrorBackoff;Failed
type TaskPhase string

const (
	// TaskPhaseInitializing indicates the Task is initializing with span contexts
	TaskPhaseInitializing TaskPhase = "Initializing"
	// TaskPhasePending indicates the Task is pending execution
	TaskPhasePending TaskPhase = "Pending"
	// TaskPhaseReadyForLLM indicates the Task is ready for context to be sent to LLM
	TaskPhaseReadyForLLM TaskPhase = "ReadyForLLM"
	// TaskPhaseSendContextWindowToLLM indicates the Task is sending context to LLM
	TaskPhaseSendContextWindowToLLM TaskPhase = "SendContextWindowToLLM"
	// TaskPhaseToolCallsPending indicates the Task has pending tool calls
	TaskPhaseToolCallsPending TaskPhase = "ToolCallsPending"
	// TaskPhaseCheckingToolCalls indicates the Task is checking if tool calls are complete
	TaskPhaseCheckingToolCalls TaskPhase = "CheckingToolCalls"
	// TaskPhaseFinalAnswer indicates the Task has received final answer
	TaskPhaseFinalAnswer TaskPhase = "FinalAnswer"
	// TaskPhaseErrorBackoff indicates the Task has failed and is in error backoff
	TaskPhaseErrorBackoff TaskPhase = "ErrorBackoff"
	// TaskPhaseFailed indicates the Task has failed
	TaskPhaseFailed TaskPhase = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Agent",type="string",JSONPath=".spec.agentRef.name"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Detail",type="string",JSONPath=".status.statusDetail",priority=1
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Preview",type="string",JSONPath=".status.userMsgPreview"
// +kubebuilder:printcolumn:name="Output",type="string",JSONPath=".status.output"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error",priority=1
// +kubebuilder:printcolumn:name="Started",type="date",JSONPath=".status.startTime",priority=1
// +kubebuilder:printcolumn:name="Completed",type="date",JSONPath=".status.completionTime",priority=1
// +kubebuilder:resource:scope=Namespaced

// Task is the Schema for the tasks API
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TaskList contains a list of Task
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
