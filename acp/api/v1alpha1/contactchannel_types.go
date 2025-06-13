/*
Copyright 2025 the Agent Control Plane Authors.

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

type ContactChannelType string

const (
	ContactChannelTypeSlack ContactChannelType = "slack"
	ContactChannelTypeEmail ContactChannelType = "email"
)

// SlackChannelConfig defines configuration specific to Slack channels
type SlackChannelConfig struct {
	// ChannelOrUserID is the Slack channel ID (C...) or user ID (U...)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[CU][A-Z0-9]+$
	ChannelOrUserID string `json:"channelOrUserID"`

	// ContextAboutChannelOrUser provides context for the LLM about the channel or user
	ContextAboutChannelOrUser string `json:"contextAboutChannelOrUser,omitempty"`

	// AllowedResponderIDs restricts who can respond (Slack user IDs)
	AllowedResponderIDs []string `json:"allowedResponderIDs,omitempty"`
}

// EmailChannelConfig defines configuration specific to Email channels
type EmailChannelConfig struct {
	// Address is the recipient email address
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=.+@.+\..+
	Address string `json:"address"`

	// ContextAboutUser provides context for the LLM about the recipient
	ContextAboutUser string `json:"contextAboutUser,omitempty"`

	// Subject is the custom subject line
	Subject string `json:"subject,omitempty"`
}

// ContactChannelSpec defines the desired state of ContactChannel.
type ContactChannelSpec struct {
	// Type is the type of channel (e.g. "slack", "email")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=slack;email
	Type ContactChannelType `json:"type"`

	// APIKeyFrom references the secret containing the API key or token
	// Required unless ChannelAPIKeyFrom and ChannelID are provided
	// +optional
	APIKeyFrom *APIKeySource `json:"apiKeyFrom,omitempty"`

	// ChannelAPIKeyFrom references the secret containing the channel-specific API key
	// Mutually exclusive with APIKeyFrom. Requires ChannelID when set.
	// +optional
	ChannelAPIKeyFrom *APIKeySource `json:"channelApiKeyFrom,omitempty"`

	// ChannelID specifies the channel ID when using channel-specific authentication
	// Required when ChannelAPIKeyFrom is set
	// +optional
	ChannelID string `json:"channelId,omitempty"`

	// Slack holds configuration specific to Slack channels
	// +optional
	Slack *SlackChannelConfig `json:"slack,omitempty"`

	// Email holds configuration specific to Email channels
	// +optional
	Email *EmailChannelConfig `json:"email,omitempty"`
}

// ContactChannelStatus defines the observed state of ContactChannel.
type ContactChannelStatus struct {
	// Ready indicates if the ContactChannel is ready to be used
	Ready bool `json:"ready,omitempty"`

	// Status indicates the current status of the ContactChannel
	// +kubebuilder:validation:Enum=Ready;Error;Pending
	Status string `json:"status,omitempty"`

	// StatusDetail provides additional details about the current status
	StatusDetail string `json:"statusDetail,omitempty"`

	// ProjectSlug is the project slug from HumanLayer API
	ProjectSlug string `json:"projectSlug,omitempty"`

	// OrgSlug is the organization slug from HumanLayer API
	OrgSlug string `json:"orgSlug,omitempty"`

	// VerifiedChannelID is the verified channel ID when using channel-specific auth
	VerifiedChannelID string `json:"verifiedChannelId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Detail",type="string",JSONPath=".status.statusDetail",priority=1
// +kubebuilder:resource:scope=Namespaced

// ContactChannel is the Schema for the contactchannels API.
type ContactChannel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContactChannelSpec   `json:"spec,omitempty"`
	Status ContactChannelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ContactChannelList contains a list of ContactChannel.
type ContactChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContactChannel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContactChannel{}, &ContactChannelList{})
}
