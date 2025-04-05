/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretKeyRef contains the reference to a secret key
type SecretKeyRef struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key is the key in the secret
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// APIKeySource defines how to get the API key
type APIKeySource struct {
	// SecretKeyRef references a key in a secret
	SecretKeyRef SecretKeyRef `json:"secretKeyRef"`
}

// BaseConfig holds common configuration options across providers
type BaseConfig struct {
	// Model name to use
	Model string `json:"model,omitempty"`

	// BaseURL for API endpoints (used by many providers)
	BaseURL string `json:"baseUrl,omitempty"`

	// Temperature adjusts the LLM response randomness (0.0 to 1.0)
	// +kubebuilder:validation:Pattern=^0(\.[0-9]+)?|1(\.0+)?$
	Temperature string `json:"temperature,omitempty"`

	// MaxTokens defines the maximum number of tokens for the LLM
	// +kubebuilder:validation:Minimum=1
	MaxTokens *int `json:"maxTokens,omitempty"`

	// TopP controls diversity via nucleus sampling (0.0 to 1.0)
	// +kubebuilder:validation:Pattern=^(0(\.[0-9]+)?|1(\.0+)?)$
	TopP string `json:"topP,omitempty"`

	// TopK controls diversity by limiting the top K tokens to sample from
	// +kubebuilder:validation:Minimum=1
	TopK *int `json:"topK,omitempty"`

	// FrequencyPenalty reduces repetition by penalizing frequent tokens
	// +kubebuilder:validation:Pattern=^-?[0-2](\.[0-9]+)?$
	FrequencyPenalty string `json:"frequencyPenalty,omitempty"`

	// PresencePenalty reduces repetition by penalizing tokens that appear at all
	// +kubebuilder:validation:Pattern=^-?[0-2](\.[0-9]+)?$
	PresencePenalty string `json:"presencePenalty,omitempty"`
}

// OpenAIConfig for OpenAI-specific options
type OpenAIConfig struct {
	// Organization is the OpenAI organization ID
	Organization string `json:"organization,omitempty"`

	// APIType specifies which OpenAI API type to use
	// +kubebuilder:validation:Enum=OPEN_AI;AZURE;AZURE_AD
	// +kubebuilder:default=OPEN_AI
	APIType string `json:"apiType,omitempty"`

	// APIVersion is required when using Azure API types
	// Example: "2023-05-15"
	APIVersion string `json:"apiVersion,omitempty"`
}

// AnthropicConfig for Anthropic-specific options
type AnthropicConfig struct {
	// AnthropicBetaHeader adds the Anthropic Beta header to support extended options
	// Common values include "max-tokens-3-5-sonnet-2024-07-15" for extended token limits
	// +kubebuilder:validation:Optional
	AnthropicBetaHeader string `json:"anthropicBetaHeader,omitempty"`
}

// VertexConfig for Vertex-specific options
type VertexConfig struct {
	// CloudProject is the Google Cloud project ID
	// +kubebuilder:validation:Required
	CloudProject string `json:"cloudProject"`

	// CloudLocation is the Google Cloud region
	// +kubebuilder:validation:Required
	CloudLocation string `json:"cloudLocation"`
}

// MistralConfig for Mistral-specific options
type MistralConfig struct {
	// MaxRetries sets the maximum number of retries for API calls
	// +kubebuilder:validation:Minimum=0
	MaxRetries *int `json:"maxRetries,omitempty"`

	// Timeout specifies the timeout duration for API calls (in seconds)
	// +kubebuilder:validation:Minimum=1
	Timeout *int `json:"timeout,omitempty"`

	// RandomSeed provides a seed for deterministic sampling
	// +kubebuilder:validation:Optional
	RandomSeed *int `json:"randomSeed,omitempty"`
}

// GoogleConfig for Google AI-specific options
type GoogleConfig struct {
	// CloudProject is the Google Cloud project ID
	CloudProject string `json:"cloudProject,omitempty"`

	// CloudLocation is the Google Cloud region
	CloudLocation string `json:"cloudLocation,omitempty"`
}

// ProviderConfig holds provider-specific configurations
type ProviderConfig struct {
	OpenAIConfig    *OpenAIConfig    `json:"openaiConfig,omitempty"`
	AnthropicConfig *AnthropicConfig `json:"anthropicConfig,omitempty"`
	VertexConfig    *VertexConfig    `json:"vertexConfig,omitempty"`
	MistralConfig   *MistralConfig   `json:"mistralConfig,omitempty"`
	GoogleConfig    *GoogleConfig    `json:"googleConfig,omitempty"`
}

// LLMSpec defines the desired state of LLM
type LLMSpec struct {
	// Provider is the LLM provider name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=openai;anthropic;mistral;google;vertex;
	Provider string `json:"provider"`

	// APIKeyFrom references the secret containing the API key or credentials
	APIKeyFrom *APIKeySource `json:"apiKeyFrom,omitempty"`

	// Parameters holds common configuration options across providers
	// +optional
	Parameters BaseConfig `json:"parameters,omitempty"`

	// OpenAI provider-specific configuration
	// +optional
	OpenAI *OpenAIConfig `json:"openai,omitempty"`

	// Anthropic provider-specific configuration
	// +optional
	Anthropic *AnthropicConfig `json:"anthropic,omitempty"`

	// Vertex provider-specific configuration
	// +optional
	Vertex *VertexConfig `json:"vertex,omitempty"`

	// Mistral provider-specific configuration
	// +optional
	Mistral *MistralConfig `json:"mistral,omitempty"`

	// Google provider-specific configuration
	// +optional
	Google *GoogleConfig `json:"google,omitempty"`
}

// LLMStatus defines the observed state of LLM
type LLMStatus struct {
	// Ready indicates if the LLM is ready to be used
	Ready bool `json:"ready,omitempty"`

	// Status indicates the current status of the LLM
	// +kubebuilder:validation:Enum=Ready;Error;Pending
	Status string `json:"status,omitempty"`

	// StatusDetail provides additional details about the current status
	StatusDetail string `json:"statusDetail,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.provider"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Detail",type="string",JSONPath=".status.statusDetail",priority=1
// +kubebuilder:resource:scope=Namespaced

// LLM is the Schema for the llms API
type LLM struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMSpec   `json:"spec,omitempty"`
	Status LLMStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LLMList contains a list of LLM
type LLMList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LLM `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LLM{}, &LLMList{})
}
