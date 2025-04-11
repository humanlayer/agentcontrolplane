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

type ToolType string

const (
	ToolTypeMCP          ToolType = "MCP"
	ToolTypeHumanContact ToolType = "HumanContact"
	ToolTypeExecute      ToolType = "Execute"
)

type ToolCallSpec struct {
	ToolCallID string `json:"toolCallId"`
	TaskRef    LocalObjectReference `json:"taskRef"`
	ToolRef    LocalObjectReference `json:"toolRef"`
	ToolType   ToolType `json:"toolType,omitempty"`
	Arguments  string `json:"arguments"`
}

type ToolCallStatus struct {
	Phase         ToolCallPhase `json:"phase,omitempty"`
	Ready         bool          `json:"ready,omitempty"`
	Status        ToolCallStatusType `json:"status,omitempty"`
	StatusDetail  string        `json:"statusDetail,omitempty"`
	ExternalCallID string       `json:"externalCallID"`
	Result        string        `json:"result,omitempty"`
	Error         string        `json:"error,omitempty"`
	StartTime     *metav1.Time  `json:"startTime,omitempty"`
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	SpanContext   *SpanContext   `json:"spanContext,omitempty"`
}

type ToolCallPhase string

const (
	ToolCallPhasePending                   ToolCallPhase = "Pending"
	ToolCallPhaseRunning                   ToolCallPhase = "Running"
	ToolCallPhaseSucceeded                 ToolCallPhase = "Succeeded"
	ToolCallPhaseFailed                    ToolCallPhase = "Failed"
	ToolCallPhaseAwaitingHumanInput        ToolCallPhase = "AwaitingHumanInput"
	ToolCallPhaseAwaitingSubAgent          ToolCallPhase = "AwaitingSubAgent"
	ToolCallPhaseAwaitingHumanApproval     ToolCallPhase = "AwaitingHumanApproval"
	ToolCallPhaseReadyToExecuteApprovedTool ToolCallPhase = "ReadyToExecuteApprovedTool"
	ToolCallPhaseErrorRequestingHumanApproval ToolCallPhase = "ErrorRequestingHumanApproval"
	ToolCallPhaseToolCallRejected          ToolCallPhase = "ToolCallRejected"
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

type ToolCall struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ToolCallSpec   `json:"spec,omitempty"`
	Status            ToolCallStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ToolCallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ToolCall `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ToolCall{}, &ToolCallList{})
}
