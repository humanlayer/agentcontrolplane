package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentSpec defines the desired state of Agent
type AgentSpec struct {
	// LLMRef references the LLM to use for this agent
	// +kubebuilder:validation:Required
	LLMRef LocalObjectReference `json:"llmRef"`

	// Tools is a list of tools this agent can use
	// +optional
	Tools []LocalObjectReference `json:"tools,omitempty"`

	// MCPServers is a list of MCP servers this agent can use
	// +optional
	MCPServers []LocalObjectReference `json:"mcpServers,omitempty"`

	// HumanContactChannels is a list of ContactChannel resources that can be used for human interactions
	// +optional
	HumanContactChannels []LocalObjectReference `json:"humanContactChannels,omitempty"`

	// System is the system prompt for the agent
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	System string `json:"system"`
}

// LocalObjectReference contains enough information to locate the referenced resource in the same namespace
type LocalObjectReference struct {
	// Name of the referent
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// AgentStatus defines the observed state of Agent
type AgentStatus struct {
	// Ready indicates if the agent's dependencies (LLM and Tools) are valid and ready
	Ready bool `json:"ready,omitempty"`

	// Status indicates the current status of the agent
	// +kubebuilder:validation:Enum=Ready;Error;Pending
	Status string `json:"status,omitempty"`

	// StatusDetail provides additional details about the current status
	StatusDetail string `json:"statusDetail,omitempty"`

	// ValidTools is the list of tools that were successfully validated
	// +optional
	ValidTools []ResolvedTool `json:"validTools,omitempty"`

	// ValidMCPServers is the list of MCP servers that were successfully validated
	// +optional
	ValidMCPServers []ResolvedMCPServer `json:"validMCPServers,omitempty"`

	// ValidHumanContactChannels is the list of human contact channels that were successfully validated
	// +optional
	ValidHumanContactChannels []ResolvedContactChannel `json:"validHumanContactChannels,omitempty"`
}

type ResolvedTool struct {
	// +kubebuilder:validation:Enum=Tool;ToolSet
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Name of the tool
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

type ResolvedMCPServer struct {
	// Name of the MCP server
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Tools available from this MCP server
	// +optional
	Tools []string `json:"tools,omitempty"`
}

type ResolvedContactChannel struct {
	// Name of the contact channel
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of the contact channel (e.g., "slack", "email")
	// +kubebuilder:validation:Required
	Type string `json:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Detail",type="string",JSONPath=".status.statusDetail",priority=1
// +kubebuilder:resource:scope=Namespaced

// Agent is the Schema for the agents API
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentList contains a list of Agent
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}
