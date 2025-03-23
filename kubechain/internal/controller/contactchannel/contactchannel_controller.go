/*
Copyright 2025 the Kubechain Authors.

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
	"fmt"
	"net/http"
	"net/mail"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

var (
	// Status constants
	statusReady   = "Ready"
	statusError   = "Error"
	statusPending = "Pending"

	// Channel types
	channelTypeSlack = "slack"
	channelTypeEmail = "email"

	// API endpoints - variables so they can be overridden in tests
	slackAPIEndpoint = "https://slack.com/api/auth.test"
	humanLayerAPIURL = "https://api.humanlayer.dev/humanlayer/v1/function_calls"

	// Event reasons
	eventReasonInitializing        = "Initializing"
	eventReasonValidationFailed    = "ValidationFailed"
	eventReasonValidationSucceeded = "ValidationSucceeded"
)

// ContactChannelReconciler reconciles a ContactChannel object
type ContactChannelReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=contactchannels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=contactchannels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=contactchannels/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// validateSlackToken verifies that the provided Slack token is valid by making an auth.test API call
func (r *ContactChannelReconciler) validateSlackToken(token string) error {
	req, err := http.NewRequest("GET", slackAPIEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid Slack token (status code: %d)", resp.StatusCode)
	}

	return nil
}

// validateHumanLayerAPIKey checks if the HumanLayer API key is valid
func (r *ContactChannelReconciler) validateHumanLayerAPIKey(apiKey string) error {
	req, err := http.NewRequest("GET", humanLayerAPIURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	// For HumanLayer API, a 401 would indicate invalid token
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid HumanLayer API key")
	}

	// The endpoint might return other status codes even with valid token
	// since we're doing a GET without a proper body, but as long as
	// it's not a 401, we consider the token valid
	return nil
}

// validateEmailAddress checks if the email address is valid
func (r *ContactChannelReconciler) validateEmailAddress(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email address: %w", err)
	}
	return nil
}

// validateChannelConfig validates the channel configuration based on channel type
func (r *ContactChannelReconciler) validateChannelConfig(channel *kubechainv1alpha1.ContactChannel) error {
	switch channel.Spec.ChannelType {
	case channelTypeSlack:
		if channel.Spec.SlackConfig == nil {
			return fmt.Errorf("slackConfig is required for slack channel type")
		}
		// Slack channel ID validation is handled by the CRD validation
		return nil

	case channelTypeEmail:
		if channel.Spec.EmailConfig == nil {
			return fmt.Errorf("emailConfig is required for email channel type")
		}
		return r.validateEmailAddress(channel.Spec.EmailConfig.Address)

	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Spec.ChannelType)
	}
}

// validateSecret validates the secret and the API key
func (r *ContactChannelReconciler) validateSecret(ctx context.Context, channel *kubechainv1alpha1.ContactChannel) error {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{
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

	// First validate the HumanLayer API key
	if err := r.validateHumanLayerAPIKey(apiKey); err != nil {
		return err
	}

	// Also validate channel-specific credential if needed
	switch channel.Spec.ChannelType {
	case channelTypeSlack:
		// For Slack channels, we may need to validate Slack token separately
		// if the implementation requires a separate Slack token
		// This would depend on how HumanLayer handles the integration
		return nil

	case channelTypeEmail:
		// Email validation doesn't require additional API key validation
		return nil

	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Spec.ChannelType)
	}
}

// Reconcile handles the reconciliation of ContactChannel resources
func (r *ContactChannelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the ContactChannel instance
	var channel kubechainv1alpha1.ContactChannel
	if err := r.Get(ctx, req.NamespacedName, &channel); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Starting reconciliation", "namespacedName", req.NamespacedName, "channelType", channel.Spec.ChannelType)

	// Create a copy for status update
	statusUpdate := channel.DeepCopy()

	// Initialize status if not set
	if statusUpdate.Status.Status == "" {
		statusUpdate.Status.Status = statusPending
		statusUpdate.Status.StatusDetail = "Validating configuration"
		r.recorder.Event(&channel, corev1.EventTypeNormal, eventReasonInitializing, "Starting validation")
	}

	// Validate channel configuration
	if err := r.validateChannelConfig(&channel); err != nil {
		log.Error(err, "Channel configuration validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = statusError
		statusUpdate.Status.StatusDetail = err.Error()
		r.recorder.Event(&channel, corev1.EventTypeWarning, eventReasonValidationFailed, err.Error())

		// Update status and return
		if err := r.Status().Patch(ctx, statusUpdate, client.MergeFrom(&channel)); err != nil {
			log.Error(err, "Unable to update ContactChannel status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Validate secret and API key
	if err := r.validateSecret(ctx, &channel); err != nil {
		log.Error(err, "Secret validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = statusError
		statusUpdate.Status.StatusDetail = err.Error()
		r.recorder.Event(&channel, corev1.EventTypeWarning, eventReasonValidationFailed, err.Error())
	} else {
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = statusReady
		statusUpdate.Status.StatusDetail = fmt.Sprintf("HumanLayer %s channel validated successfully", channel.Spec.ChannelType)
		r.recorder.Event(&channel, corev1.EventTypeNormal, eventReasonValidationSucceeded, statusUpdate.Status.StatusDetail)
	}

	// Update status using SubResource client
	if err := r.Status().Patch(ctx, statusUpdate, client.MergeFrom(&channel)); err != nil {
		log.Error(err, "Unable to update ContactChannel status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled ContactChannel",
		"channelType", channel.Spec.ChannelType,
		"ready", statusUpdate.Status.Ready,
		"status", statusUpdate.Status.Status,
		"statusDetail", statusUpdate.Status.StatusDetail)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ContactChannelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("contactchannel-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.ContactChannel{}).
		Named("contactchannel").
		Complete(r)
}
