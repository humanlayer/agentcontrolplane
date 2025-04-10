package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MCPServerSpec defines the desired state of MCPServer
type MCPServerSpec struct {
	// Transport specifies the transport type for the MCP server
	// +kubebuilder:validation:Enum=stdio;http
	// +kubebuilder:validation:Required
	Transport string `json:"transport"`

	// Command is the command to run for stdio MCP servers
	// +optional
	Command string `json:"command,omitempty"`

	// Args are the arguments to pass to the command for stdio MCP servers
	// +optional
	Args []string `json:"args,omitempty"`

	// Env are environment variables to set for stdio MCP servers
	// +optional
	Env []EnvVar `json:"env,omitempty"`

	// URL is the endpoint for HTTP MCP servers
	// +optional
	URL string `json:"url,omitempty"`

	// ResourceRequirements defines CPU/Memory resources requests/limits
	// +optional
	Resources ResourceRequirements `json:"resources,omitempty"`

	// ApprovalContactChannel is the contact channel for approval
	// +optional
	ApprovalContactChannel *LocalObjectReference `json:"approvalContactChannel,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	// Name of the environment variable
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Value of the environment variable (direct literal value)
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom represents a source for the value of an environment variable
	// +optional
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

// EnvVarSource represents a source for the value of an environment variable
type EnvVarSource struct {
	// SecretKeyRef selects a key of a secret in the pod's namespace
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// We're using the SecretKeySelector from tool_types.go

// ResourceRequirements describes the compute resource requirements
type ResourceRequirements struct {
	// Limits describes the maximum amount of compute resources allowed
	// +optional
	Limits ResourceList `json:"limits,omitempty"`

	// Requests describes the minimum amount of compute resources required
	// +optional
	Requests ResourceList `json:"requests,omitempty"`
}

// ResourceList is a set of (resource name, quantity) pairs
type ResourceList map[ResourceName]resource.Quantity

// ResourceName is the name identifying various resources
type ResourceName string

const (
	// ResourceCPU CPU resource
	ResourceCPU ResourceName = "cpu"
	// ResourceMemory memory resource
	ResourceMemory ResourceName = "memory"
)

// MCPTool represents a tool provided by an MCP server
type MCPTool struct {
	// Name of the tool
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the tool
	// +optional
	Description string `json:"description,omitempty"`

	// InputSchema is the JSON schema for the tool's input parameters
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	InputSchema runtime.RawExtension `json:"inputSchema,omitempty"`
}

// MCPServerStatus defines the observed state of MCPServer
type MCPServerStatus struct {
	// Connected indicates if the MCP server is currently connected and operational
	Connected bool `json:"connected,omitempty"`

	// Status indicates the current status of the MCP server
	// +kubebuilder:validation:Enum=Ready;Error;Pending
	Status string `json:"status,omitempty"`

	// StatusDetail provides additional details about the current status
	StatusDetail string `json:"statusDetail,omitempty"`

	// Tools is the list of tools provided by this MCP server
	// +optional
	Tools []MCPTool `json:"tools,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Connected",type="boolean",JSONPath=".status.connected"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Detail",type="string",JSONPath=".status.statusDetail",priority=1
// +kubebuilder:resource:scope=Namespaced

// MCPServer is the Schema for the mcpservers API
type MCPServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MCPServerSpec   `json:"spec,omitempty"`
	Status MCPServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MCPServerList contains a list of MCPServer
type MCPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPServer `json:"items"`
}
