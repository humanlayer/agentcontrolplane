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

package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/googleai/vertex"
	"github.com/tmc/langchaingo/llms/mistral"
	"github.com/tmc/langchaingo/llms/openai"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=llms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=llms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// LLMReconciler reconciles a LLM object
type LLMReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

//
// llms.withTools can be used for passing in tools
// This is in options.go in langchaingo/llms/
// WithTools will add an option to set the tools to use.
// func WithTools(tools []Tool) CallOption {
// 	return func(o *CallOptions) {
// 		o.Tools = tools
// 	}
// }

// Some providers can have a base url. Here is an example of a base url	for OpenAI.
// This is in openaillm_option.go in langchaingo/llms/openai/
// WithBaseURL passes the OpenAI base url to the client. If not set, the base url
// is read from the OPENAI_BASE_URL environment variable. If still not set in ENV
// VAR OPENAI_BASE_URL, then the default value is https://api.openai.com/v1 is used.
//
//	func WithBaseURL(baseURL string) Option {
//		return func(opts *options) {
//			opts.baseURL = baseURL
//		}
//	}

// validateProviderConfig validates the LLM provider configuration against the actual API
// TODO: Refactor this function to reduce cyclomatic complexity (currently at 59)
func (r *LLMReconciler) validateProviderConfig(ctx context.Context, llm *kubechainv1alpha1.LLM, apiKey string) error { //nolint:gocyclo
	var err error
	var model llms.Model

	// Common options from Parameters
	commonOpts := []llms.CallOption{}

	// Get parameter configuration
	params := llm.Spec.Parameters

	if params.Model != "" {
		commonOpts = append(commonOpts, llms.WithModel(params.Model))
	}
	if params.MaxTokens != nil {
		commonOpts = append(commonOpts, llms.WithMaxTokens(*params.MaxTokens))
	}
	if params.Temperature != "" {
		// Parse temperature string to float64
		var temp float64
		_, err := fmt.Sscanf(params.Temperature, "%f", &temp)
		if err == nil && temp >= 0 && temp <= 1 {
			commonOpts = append(commonOpts, llms.WithTemperature(temp))
		}
	}
	// Add TopP if configured
	if params.TopP != "" {
		// Parse TopP string to float64
		var topP float64
		_, err := fmt.Sscanf(params.TopP, "%f", &topP)
		if err == nil && topP >= 0 && topP <= 1 {
			commonOpts = append(commonOpts, llms.WithTopP(topP))
		}
	}
	// Add TopK if configured
	if params.TopK != nil {
		commonOpts = append(commonOpts, llms.WithTopK(*params.TopK))
	}
	// Add FrequencyPenalty if configured
	if params.FrequencyPenalty != "" {
		// Parse FrequencyPenalty string to float64
		var freqPenalty float64
		_, err := fmt.Sscanf(params.FrequencyPenalty, "%f", &freqPenalty)
		if err == nil && freqPenalty >= -2 && freqPenalty <= 2 {
			commonOpts = append(commonOpts, llms.WithFrequencyPenalty(freqPenalty))
		}
	}
	// Add PresencePenalty if configured
	if params.PresencePenalty != "" {
		// Parse PresencePenalty string to float64
		var presPenalty float64
		_, err := fmt.Sscanf(params.PresencePenalty, "%f", &presPenalty)
		if err == nil && presPenalty >= -2 && presPenalty <= 2 {
			commonOpts = append(commonOpts, llms.WithPresencePenalty(presPenalty))
		}
	}

	switch llm.Spec.Provider {
	case "openai":
		if llm.Spec.APIKeyFrom == nil {
			return fmt.Errorf("apiKeyFrom is required for openai")
		}
		providerOpts := []openai.Option{openai.WithToken(apiKey)}

		// Configure BaseURL if provided
		if llm.Spec.Parameters.BaseURL != "" {
			providerOpts = append(providerOpts, openai.WithBaseURL(llm.Spec.Parameters.BaseURL))
		}

		// Configure OpenAI specific options if provided
		if llm.Spec.OpenAI != nil {
			config := llm.Spec.OpenAI

			// Set organization if provided
			if config.Organization != "" {
				providerOpts = append(providerOpts, openai.WithOrganization(config.Organization))
			}

			// Configure API type if provided
			if config.APIType != "" {
				var apiType openai.APIType
				switch config.APIType {
				case "AZURE":
					apiType = openai.APITypeAzure
				case "AZURE_AD":
					apiType = openai.APITypeAzureAD
				default:
					apiType = openai.APITypeOpenAI
				}
				providerOpts = append(providerOpts, openai.WithAPIType(apiType))

				// When using Azure APIs, configure API Version
				if (config.APIType == "AZURE" || config.APIType == "AZURE_AD") && config.APIVersion != "" {
					providerOpts = append(providerOpts, openai.WithAPIVersion(config.APIVersion))
				}
			}
		}

		model, err = openai.New(providerOpts...)

	case "anthropic":
		if llm.Spec.APIKeyFrom == nil {
			return fmt.Errorf("apiKeyFrom is required for anthropic")
		}
		providerOpts := []anthropic.Option{anthropic.WithToken(apiKey)}
		if llm.Spec.Parameters.BaseURL != "" {
			providerOpts = append(providerOpts, anthropic.WithBaseURL(llm.Spec.Parameters.BaseURL))
		}
		if llm.Spec.Anthropic != nil && llm.Spec.Anthropic.AnthropicBetaHeader != "" {
			providerOpts = append(providerOpts, anthropic.WithAnthropicBetaHeader(llm.Spec.Anthropic.AnthropicBetaHeader))
		}
		model, err = anthropic.New(providerOpts...)

	case "mistral":
		if llm.Spec.APIKeyFrom == nil {
			return fmt.Errorf("apiKeyFrom is required for mistral")
		}
		providerOpts := []mistral.Option{mistral.WithAPIKey(apiKey)}

		// Configure BaseURL as endpoint
		if llm.Spec.Parameters.BaseURL != "" {
			providerOpts = append(providerOpts, mistral.WithEndpoint(llm.Spec.Parameters.BaseURL))
		}

		// Configure model
		if llm.Spec.Parameters.Model != "" {
			providerOpts = append(providerOpts, mistral.WithModel(llm.Spec.Parameters.Model))
		}

		// Configure Mistral-specific options if provided
		if llm.Spec.Mistral != nil {
			config := llm.Spec.Mistral

			// Set MaxRetries if provided
			if config.MaxRetries != nil {
				providerOpts = append(providerOpts, mistral.WithMaxRetries(*config.MaxRetries))
			}

			// Set Timeout if provided (converting seconds to time.Duration)
			if config.Timeout != nil {
				timeoutDuration := time.Duration(*config.Timeout) * time.Second
				providerOpts = append(providerOpts, mistral.WithTimeout(timeoutDuration))
			}

			// Set RandomSeed if provided
			if config.RandomSeed != nil {
				commonOpts = append(commonOpts, llms.WithSeed(*config.RandomSeed))
			}
		}

		// Create the Mistral model with the provider options
		model, err = mistral.New(providerOpts...)

		// TODO: Elipsis had feedback that should be looked at later maybe:
		// In the Mistral case, the branch calls GenerateFromSinglePrompt inside the switch then returns nil early. This deviates from the pattern of test-validation call that happens afterwards. Ensure the intended logic is maintained.
		// https://github.com/humanlayer/kubechain/pull/35#discussion_r2013064446
		// Pass any common options to the model during generation test
		if len(commonOpts) > 0 {
			commonOpts = append(commonOpts, llms.WithMaxTokens(1), llms.WithTemperature(0))
			_, err = llms.GenerateFromSinglePrompt(ctx, model, "test", commonOpts...)
			if err != nil {
				return fmt.Errorf("mistral validation failed with options: %w", err)
			}
			return nil
		}

	case "google":
		if llm.Spec.APIKeyFrom == nil {
			return fmt.Errorf("apiKeyFrom is required for google")
		}
		providerOpts := []googleai.Option{googleai.WithAPIKey(apiKey)}
		if llm.Spec.Google != nil {
			if llm.Spec.Google.CloudProject != "" {
				providerOpts = append(providerOpts, googleai.WithCloudProject(llm.Spec.Google.CloudProject))
			}
			if llm.Spec.Google.CloudLocation != "" {
				providerOpts = append(providerOpts, googleai.WithCloudLocation(llm.Spec.Google.CloudLocation))
			}
		}
		if llm.Spec.Parameters.Model != "" {
			providerOpts = append(providerOpts, googleai.WithDefaultModel(llm.Spec.Parameters.Model))
		}
		model, err = googleai.New(ctx, providerOpts...)

	case "vertex":
		if llm.Spec.Vertex == nil {
			return fmt.Errorf("vertex configuration is required for vertex provider")
		}
		config := llm.Spec.Vertex
		providerOpts := []googleai.Option{
			googleai.WithCloudProject(config.CloudProject),
			googleai.WithCloudLocation(config.CloudLocation),
		}
		if llm.Spec.APIKeyFrom != nil && apiKey != "" {
			providerOpts = append(providerOpts, googleai.WithCredentialsJSON([]byte(apiKey)))
		}
		if llm.Spec.Parameters.Model != "" {
			providerOpts = append(providerOpts, googleai.WithDefaultModel(llm.Spec.Parameters.Model))
		}
		model, err = vertex.New(ctx, providerOpts...)

	default:
		return fmt.Errorf("unsupported provider: %s. Supported providers are: openai, anthropic, mistral, google, vertex", llm.Spec.Provider)
	}

	if err != nil {
		return fmt.Errorf("failed to initialize %s client: %w", llm.Spec.Provider, err)
	}

	// Validate with a test call
	validateOptions := []llms.CallOption{llms.WithTemperature(0), llms.WithMaxTokens(1)}

	// Add model option to ensure we validate with the correct model
	if llm.Spec.Parameters.Model != "" {
		validateOptions = append(validateOptions, llms.WithModel(llm.Spec.Parameters.Model))
	}

	_, err = llms.GenerateFromSinglePrompt(ctx, model, "test", validateOptions...)
	if err != nil {
		return fmt.Errorf("%s API validation failed: %w", llm.Spec.Provider, err)
	}

	return nil
}

func (r *LLMReconciler) validateSecret(ctx context.Context, llm *kubechainv1alpha1.LLM) (string, error) {
	// All providers require API keys
	if llm.Spec.APIKeyFrom == nil {
		return "", fmt.Errorf("apiKeyFrom is required for provider %s", llm.Spec.Provider)
	}

	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      llm.Spec.APIKeyFrom.SecretKeyRef.Name,
		Namespace: llm.Namespace,
	}, secret)
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %w", err)
	}

	key := llm.Spec.APIKeyFrom.SecretKeyRef.Key
	apiKey, exists := secret.Data[key]
	if !exists {
		return "", fmt.Errorf("key %q not found in secret", key)
	}

	return string(apiKey), nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LLMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LLM instance
	var llm kubechainv1alpha1.LLM
	if err := r.Get(ctx, req.NamespacedName, &llm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Starting reconciliation", "namespacedName", req.NamespacedName, "provider", llm.Spec.Provider)

	// Create a copy for status update
	statusUpdate := llm.DeepCopy()

	// Initialize status if not set
	if statusUpdate.Status.Status == "" {
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.StatusDetail = "Validating configuration"
		r.recorder.Event(&llm, corev1.EventTypeNormal, "Initializing", "Starting validation")
	}

	// Validate secret and get API key (if applicable)
	// TODO: Will this work with amazon bedrock? Probably not?? If so we should look at adding tests for this specifically.
	apiKey, err := r.validateSecret(ctx, &llm)
	if err != nil {
		log.Error(err, "Secret validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = err.Error()
		r.recorder.Event(&llm, corev1.EventTypeWarning, "SecretValidationFailed", err.Error())
	} else {
		// Validate provider with API key
		err := r.validateProviderConfig(ctx, &llm, apiKey)
		if err != nil {
			log.Error(err, "Provider validation failed")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = "Error"
			statusUpdate.Status.StatusDetail = err.Error()
			r.recorder.Event(&llm, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		} else {
			statusUpdate.Status.Ready = true
			statusUpdate.Status.Status = "Ready"
			statusUpdate.Status.StatusDetail = fmt.Sprintf("%s provider validated successfully", llm.Spec.Provider)
			r.recorder.Event(&llm, corev1.EventTypeNormal, "ValidationSucceeded", statusUpdate.Status.StatusDetail)
		}
	}

	// Update status using SubResource client
	if err := r.Status().Patch(ctx, statusUpdate, client.MergeFrom(&llm)); err != nil {
		log.Error(err, "Unable to update LLM status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LLM",
		"provider", llm.Spec.Provider,
		"ready", statusUpdate.Status.Ready,
		"status", statusUpdate.Status.Status,
		"statusDetail", statusUpdate.Status.StatusDetail)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("llm-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.LLM{}).
		Named("llm").
		Complete(r)
}
