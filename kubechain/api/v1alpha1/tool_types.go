package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ToolSpec defines the desired state of Tool
type ToolSpec struct {
	// Name is used for inline/function tools (optional if the object name is used).
	Name string `json:"name,omitempty"`

	// Description provides a description of the tool.
	Description string `json:"description,omitempty"`

	// Parameters defines the JSON schema for the tool's parameters.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	Parameters runtime.RawExtension `json:"parameters,omitempty"`

	// Arguments defines the JSON schema for the tool's arguments.
	// +kubebuilder:pruning:PreserveUnknownFields
	Arguments runtime.RawExtension `json:"arguments,omitempty"`

	// ToolType represents the type of tool; e.g. "function", "delegateToAgent", "externalAPI" etc.
	// +kubebuilder:validation:Enum=function;delegateToAgent;externalAPI
	ToolType string `json:"toolType,omitempty"`

	// Execute defines how the tool should be executed.
	Execute ToolExecute `json:"execute,omitempty"`

	// AgentRef is used for delegation-type tools.
	AgentRef *AgentReference `json:"agentRef,omitempty"`
}

type ToolExecute struct {
	// Builtin represents an inline (builtin) tool.
	Builtin *BuiltinToolSpec `json:"builtin,omitempty"`
}

// NameReference contains a name reference to another resource
type NameReference struct {
	// Name of the referent
	Name string `json:"name"`
}

// AgentReference defines a reference to an agent resource.
type AgentReference struct {
	Name string `json:"name"`
}

// BuiltinToolSpec defines the parameters for executing a builtin tool.
type BuiltinToolSpec struct {
	// Name is the identifier of the builtin function to run. Today, supports simple math operations
	// +kubebuilder:validation:Enum=add;subtract;multiply;divide
	Name string `json:"name,omitempty"`
}

type ExternalAPISpec struct {
	// URL for the API endpoint
	URL string `json:"url,omitempty"`

	// Method specifies the HTTP method to use (GET, POST, etc.)
	Method string `json:"method,omitempty"`

	// RequiresApproval indicates if this API call needs explicit approval
	RequiresApproval bool `json:"requiresApproval,omitempty"`

	// Credentials reference for API authentication
	CredentialsFrom *SecretKeyRef `json:"credentialsFrom,omitempty"`
}

type SecretKeySelector struct {
	// Name of the secret
	Name string `json:"name"`

	// Key within the secret
	Key string `json:"key"`
}

// ToolStatus defines the observed state of Tool
type ToolStatus struct {
	// Ready indicates if the tool is ready to be used
	Ready bool `json:"ready,omitempty"`

	// Status indicates the current status of the tool
	// +kubebuilder:validation:Enum=Ready;Error;Pending
	Status string `json:"status,omitempty"`

	// StatusDetail provides additional details about the current status
	StatusDetail string `json:"statusDetail,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Detail",type="string",JSONPath=".status.statusDetail",priority=1
// +kubebuilder:resource:scope=Namespaced

// Tool is the Schema for the tools API
type Tool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolSpec   `json:"spec,omitempty"`
	Status ToolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ToolList contains a list of Tool
type ToolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tool `json:"items"`
}
