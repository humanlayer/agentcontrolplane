package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ToolCallStatusType string

const (
	ToolCallStatusTypeReady     ToolCallStatusType = "Ready"
	ToolCallStatusTypeError     ToolCallStatusType = "Error"
	ToolCallStatusTypePending   ToolCallStatusType = "Pending"
	ToolCallStatusTypeSucceeded ToolCallStatusType = "Succeeded"
)

// ToolType identifies the type of the tool (Standard, MCP, HumanContact)
type ToolType string

const (
	ToolTypeMCP          ToolType = "MCP"
	ToolTypeHumanContact ToolType = "HumanContact"
)

// ToolCallSpec defines the desired state of ToolCall
type ToolCallSpec struct {
	// ToolCallID is the unique identifier for this tool call
	ToolCallID string `json:"toolCallId"`

	// TaskRef references the parent Task
	// +kubebuilder:validation:Required
	TaskRef LocalObjectReference `json:"taskRef"`

	// ToolRef references the tool to execute
	// +kubebuilder:validation:Required
	ToolRef LocalObjectReference `json:"toolRef"`

	// ToolType identifies the type of the tool (MCP, HumanContact)
	// +optional
	ToolType ToolType `json:"toolType,omitempty"`

	// Arguments contains the arguments for the tool call
	// +kubebuilder:validation:Required
	Arguments string `json:"arguments"`
}

// ToolCallStatus defines the observed state of ToolCall
type ToolCallStatus struct {
	// Phase indicates the current phase of the tool call
	// +optional
	Phase ToolCallPhase `json:"phase,omitempty"`

	// Ready indicates if the tool call is ready to be executed
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Status indicates the current status of the tool call
	// +kubebuilder:validation:Enum=Ready;Error;Pending;Succeeded
	Status ToolCallStatusType `json:"status,omitempty"`

	// StatusDetail provides additional details about the current status
	// +optional
	StatusDetail string `json:"statusDetail,omitempty"`

	// ExternalCallID is the unique identifier for this function call in external services
	ExternalCallID string `json:"externalCallID"`

	// Result contains the result of the tool call if completed
	// +optional
	Result string `json:"result,omitempty"`

	// Error message if the tool call failed
	// +optional
	Error string `json:"error,omitempty"`

	// StartTime is when the tool call started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the tool call completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// SpanContext contains OpenTelemetry span context information
	// +optional
	SpanContext *SpanContext `json:"spanContext,omitempty"`
}

// ToolCallPhase represents the phase of a ToolCall
// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;AwaitingHumanInput;AwaitingSubAgent;AwaitingHumanApproval;ReadyToExecuteApprovedTool;ErrorRequestingHumanApproval;ToolCallRejected
type ToolCallPhase string

const (
	// ToolCallPhasePending indicates the tool call is pending execution
	ToolCallPhasePending ToolCallPhase = "Pending"
	// ToolCallPhaseRunning indicates the tool call is currently executing
	ToolCallPhaseRunning ToolCallPhase = "Running"
	// ToolCallPhaseSucceeded indicates the tool call completed successfully
	ToolCallPhaseSucceeded ToolCallPhase = "Succeeded"
	// ToolCallPhaseFailed indicates the tool call failed
	ToolCallPhaseFailed ToolCallPhase = "Failed"
	// ToolCallPhaseAwaitingHumanInput indicates the tool call is waiting for human input
	ToolCallPhaseAwaitingHumanInput ToolCallPhase = "AwaitingHumanInput"
	// ToolCallPhaseAwaitingSubAgent indicates the tool call is waiting for a sub-agent to complete
	ToolCallPhaseAwaitingSubAgent ToolCallPhase = "AwaitingSubAgent"
	// ToolCallPhaseAwaitingHumanApproval indicates the tool call is waiting for human approval
	ToolCallPhaseAwaitingHumanApproval ToolCallPhase = "AwaitingHumanApproval"
	// ToolCallPhaseReadyToExecuteApprovedTool indicates the tool call is ready to execute after receiving approval
	ToolCallPhaseReadyToExecuteApprovedTool ToolCallPhase = "ReadyToExecuteApprovedTool"
	// ToolCallPhaseErrorRequestingHumanApproval indicates there was an error requesting human approval
	ToolCallPhaseErrorRequestingHumanApproval ToolCallPhase = "ErrorRequestingHumanApproval"
	// ToolCallPhaseToolCallRejected indicates the tool call was rejected by human approval
	ToolCallPhaseToolCallRejected ToolCallPhase = "ToolCallRejected"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="TaskRun",type="string",JSONPath=".spec.taskRunRef.name"
// +kubebuilder:printcolumn:name="Tool",type="string",JSONPath=".spec.toolRef.name"
// +kubebuilder:printcolumn:name="Started",type="date",JSONPath=".status.startTime",priority=1
// +kubebuilder:printcolumn:name="Completed",type="date",JSONPath=".status.completionTime",priority=1
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error",priority=1
// +kubebuilder:resource:scope=Namespaced

// ToolCall is the Schema for the taskruntoolcalls API
type ToolCall struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolCallSpec   `json:"spec,omitempty"`
	Status ToolCallStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ToolCallList contains a list of ToolCall
type ToolCallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ToolCall `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ToolCall{}, &ToolCallList{})
}
