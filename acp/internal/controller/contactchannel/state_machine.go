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

package contactchannel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// StateMachine handles all ContactChannel state transitions in one place
type StateMachine struct {
	client   client.Client
	recorder record.EventRecorder
}

// NewStateMachine creates a new state machine
func NewStateMachine(client client.Client, recorder record.EventRecorder) *StateMachine {
	return &StateMachine{
		client:   client,
		recorder: recorder,
	}
}

// Process handles a ContactChannel and returns the next action
func (sm *StateMachine) Process(ctx context.Context, channel *acp.ContactChannel) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing ContactChannel", "name", channel.Name, "status", channel.Status.Status)

	// Process based on current state
	switch channel.Status.Status {
	case "":
		return sm.initialize(ctx, channel)
	case statusPending:
		return sm.validateConfiguration(ctx, channel)
	case statusReady:
		return sm.handleReady(ctx, channel)
	case statusError:
		return sm.handleError(ctx, channel)
	default:
		return sm.initialize(ctx, channel) // Default to initialize
	}
}

// State transition methods

func (sm *StateMachine) initialize(ctx context.Context, channel *acp.ContactChannel) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Initializing ContactChannel", "type", channel.Spec.Type)

	// Initialize status to pending
	channel.Status.Status = statusPending
	channel.Status.StatusDetail = "Validating configuration"

	// Emit event for initialization
	if sm.recorder != nil {
		sm.recorder.Event(channel, corev1.EventTypeNormal, eventReasonInitializing, "Starting validation")
	}

	// Update status first
	if err := sm.updateStatus(ctx, channel); err != nil {
		return ctrl.Result{}, err
	}

	// Immediately proceed to validation (like the original controller)
	return sm.validateConfiguration(ctx, channel)
}

func (sm *StateMachine) validateConfiguration(ctx context.Context, channel *acp.ContactChannel) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Validating ContactChannel configuration", "type", channel.Spec.Type)

	// Validate channel configuration
	if err := sm.validateChannelConfig(channel); err != nil {
		logger.Error(err, "Channel configuration validation failed")
		channel.Status.Ready = false
		channel.Status.Status = statusError
		channel.Status.StatusDetail = err.Error()
		if sm.recorder != nil {
			sm.recorder.Event(channel, corev1.EventTypeWarning, eventReasonValidationFailed, err.Error())
		}
		return sm.updateAndComplete(ctx, channel)
	}

	// Validate secret and API key
	if err := sm.validateSecret(ctx, channel); err != nil {
		logger.Error(err, "Secret validation failed")
		channel.Status.Ready = false
		channel.Status.Status = statusError
		channel.Status.StatusDetail = err.Error()
		if sm.recorder != nil {
			sm.recorder.Event(channel, corev1.EventTypeWarning, eventReasonValidationFailed, err.Error())
		}
	} else {
		channel.Status.Ready = true
		channel.Status.Status = statusReady
		channel.Status.StatusDetail = fmt.Sprintf("HumanLayer %s channel validated successfully", channel.Spec.Type)
		if sm.recorder != nil {
			sm.recorder.Event(channel, corev1.EventTypeNormal, eventReasonValidationSucceeded, channel.Status.StatusDetail)
		}
	}

	return sm.updateAndComplete(ctx, channel)
}

func (sm *StateMachine) handleReady(ctx context.Context, channel *acp.ContactChannel) (ctrl.Result, error) {
	// Channel is ready, no action needed
	return ctrl.Result{}, nil
}

func (sm *StateMachine) handleError(ctx context.Context, channel *acp.ContactChannel) (ctrl.Result, error) {
	// Could implement retry logic here if needed
	return ctrl.Result{}, nil
}

// Helper methods

func (sm *StateMachine) updateAndComplete(ctx context.Context, channel *acp.ContactChannel) (ctrl.Result, error) {
	if err := sm.updateStatus(ctx, channel); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (sm *StateMachine) updateStatus(ctx context.Context, channel *acp.ContactChannel) error {
	// Fetch the latest version to avoid UID conflicts
	namespacedName := client.ObjectKey{Name: channel.Name, Namespace: channel.Namespace}
	latestChannel := &acp.ContactChannel{}
	if err := sm.client.Get(ctx, namespacedName, latestChannel); err != nil {
		return err
	}

	// Copy status fields to latest version
	latestChannel.Status = channel.Status

	return sm.client.Status().Update(ctx, latestChannel)
}

// Helper validation methods

// ProjectInfo holds project and organization information from HumanLayer API
type ProjectInfo struct {
	ProjectSlug string
	OrgSlug     string
}

// validateHumanLayerAPIKey checks if the HumanLayer API key is valid and gets project info
func (sm *StateMachine) validateHumanLayerAPIKey(apiKey string) (*ProjectInfo, error) {
	req, err := http.NewRequest("GET", humanLayerAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	// For HumanLayer API, a 401 would indicate invalid token
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid HumanLayer API key")
	}

	// Read the project details response
	var responseMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return nil, fmt.Errorf("failed to decode project response: %w", err)
	}

	// Extract project and org slugs from response
	projectInfo := &ProjectInfo{}
	if projectSlug, ok := responseMap["project_slug"]; ok {
		if slug, ok := projectSlug.(string); ok {
			projectInfo.ProjectSlug = slug
		}
	}
	if orgSlug, ok := responseMap["org_slug"]; ok {
		if slug, ok := orgSlug.(string); ok {
			projectInfo.OrgSlug = slug
		}
	}

	return projectInfo, nil
}

// validateEmailAddress checks if the email address is valid
func (sm *StateMachine) validateEmailAddress(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email address: %w", err)
	}
	return nil
}

// validateChannelConfig validates the channel configuration based on channel type
func (sm *StateMachine) validateChannelConfig(channel *acp.ContactChannel) error {
	switch channel.Spec.Type {
	case acp.ContactChannelTypeSlack:
		if channel.Spec.Slack == nil {
			return fmt.Errorf("slackConfig is required for slack channel type")
		}
		// Slack channel ID validation is handled by the CRD validation
		return nil

	case acp.ContactChannelTypeEmail:
		if channel.Spec.Email == nil {
			return fmt.Errorf("emailConfig is required for email channel type")
		}
		return sm.validateEmailAddress(channel.Spec.Email.Address)

	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Spec.Type)
	}
}

// validateSecret validates the secret and the API key
func (sm *StateMachine) validateSecret(ctx context.Context, channel *acp.ContactChannel) error {
	secret := &corev1.Secret{}
	err := sm.client.Get(ctx, types.NamespacedName{
		Name:      channel.Spec.APIKeyFrom.SecretKeyRef.Name,
		Namespace: channel.Namespace,
	}, secret)
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	key := channel.Spec.APIKeyFrom.SecretKeyRef.Key
	apiKeyBytes, exists := secret.Data[key]
	if !exists {
		return fmt.Errorf("key %q not found in secret", key)
	}

	apiKey := string(apiKeyBytes)
	if apiKey == "" {
		return fmt.Errorf("empty API key provided")
	}

	// First validate the HumanLayer API key and get project info
	projectInfo, err := sm.validateHumanLayerAPIKey(apiKey)
	if err != nil {
		return err
	}

	// Store the project and org slugs for status update
	channel.Status.ProjectSlug = projectInfo.ProjectSlug
	channel.Status.OrgSlug = projectInfo.OrgSlug

	// Also validate channel-specific credential if needed
	switch channel.Spec.Type {
	case acp.ContactChannelTypeSlack:
		// For Slack channels, we may need to validate Slack token separately
		// if the implementation requires a separate Slack token
		// This would depend on how HumanLayer handles the integration
		return nil

	case acp.ContactChannelTypeEmail:
		// Email validation doesn't require additional API key validation
		return nil

	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Spec.Type)
	}
}
