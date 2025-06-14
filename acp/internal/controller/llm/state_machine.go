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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// StateMachine handles all LLM state transitions
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

// Process handles an LLM and returns the next action
func (sm *StateMachine) Process(ctx context.Context, llm *acp.LLM) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing LLM", "name", llm.Name, "status", llm.Status.Status)

	// Determine current state
	state := sm.getLLMState(llm)

	// Dispatch to handlers based on state
	switch state {
	case statusPending, "":
		return sm.initialize(ctx, llm)
	case statusReady:
		return sm.handleReady(ctx, llm)
	case statusError:
		return sm.handleError(ctx, llm)
	default:
		return sm.initialize(ctx, llm) // Default to pending
	}
}

// State transition methods

func (sm *StateMachine) initialize(ctx context.Context, llm *acp.LLM) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Initializing LLM", "provider", llm.Spec.Provider)

	// Create a copy for status update
	statusUpdate := llm.DeepCopy()

	// Initialize status if not set
	if statusUpdate.Status.Status == "" {
		statusUpdate.Status.Status = statusPending
		statusUpdate.Status.StatusDetail = "Validating configuration"
		if sm.recorder != nil {
			sm.recorder.Event(llm, corev1.EventTypeNormal, "Initializing", "Starting validation")
		}
	}

	// Now proceed to validate
	return sm.validateProvider(ctx, statusUpdate)
}

func (sm *StateMachine) validateProvider(ctx context.Context, llm *acp.LLM) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Validating provider", "provider", llm.Spec.Provider)

	// Validate secret and get API key
	apiKey, err := sm.validateSecret(ctx, llm)
	if err != nil {
		logger.Error(err, "Secret validation failed")
		llm.Status.Ready = false
		llm.Status.Status = statusError
		llm.Status.StatusDetail = err.Error()
		if sm.recorder != nil {
			sm.recorder.Event(llm, corev1.EventTypeWarning, "SecretValidationFailed", err.Error())
		}
		return sm.updateAndComplete(ctx, llm)
	}

	// Validate provider configuration with API key
	err = sm.validateProviderConfig(ctx, llm, apiKey)
	if err != nil {
		logger.Error(err, "Provider validation failed")
		llm.Status.Ready = false
		llm.Status.Status = statusError
		llm.Status.StatusDetail = err.Error()
		if sm.recorder != nil {
			sm.recorder.Event(llm, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		}
	} else {
		llm.Status.Ready = true
		llm.Status.Status = statusReady
		llm.Status.StatusDetail = fmt.Sprintf("%s provider validated successfully", llm.Spec.Provider)
		if sm.recorder != nil {
			sm.recorder.Event(llm, corev1.EventTypeNormal, "ValidationSucceeded", llm.Status.StatusDetail)
		}
	}

	return sm.updateAndComplete(ctx, llm)
}

func (sm *StateMachine) handleReady(ctx context.Context, llm *acp.LLM) (ctrl.Result, error) {
	// LLM is ready, no action needed
	return ctrl.Result{}, nil
}

func (sm *StateMachine) handleError(ctx context.Context, llm *acp.LLM) (ctrl.Result, error) {
	// Could implement retry logic here if needed
	return ctrl.Result{}, nil
}

// Helper methods

func (sm *StateMachine) getLLMState(llm *acp.LLM) string {
	if llm.Status.Status == "" {
		return statusPending
	}
	return llm.Status.Status
}

func (sm *StateMachine) updateStatus(ctx context.Context, llm *acp.LLM) error {
	// Fetch the latest version to avoid UID conflicts
	namespacedName := client.ObjectKey{Name: llm.Name, Namespace: llm.Namespace}
	latestLLM := &acp.LLM{}
	if err := sm.client.Get(ctx, namespacedName, latestLLM); err != nil {
		return err
	}

	// Copy status fields to latest version
	latestLLM.Status = llm.Status

	return sm.client.Status().Update(ctx, latestLLM)
}

func (sm *StateMachine) updateAndComplete(ctx context.Context, llm *acp.LLM) (ctrl.Result, error) {
	if err := sm.updateStatus(ctx, llm); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (sm *StateMachine) validateSecret(ctx context.Context, llm *acp.LLM) (string, error) {
	// All providers require API keys
	if llm.Spec.APIKeyFrom == nil {
		return "", fmt.Errorf("apiKeyFrom is required for provider %s", llm.Spec.Provider)
	}

	secret := &corev1.Secret{}
	err := sm.client.Get(ctx, types.NamespacedName{
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

// validateProviderConfig validates the LLM provider configuration against the actual API
func (sm *StateMachine) validateProviderConfig(ctx context.Context, llm *acp.LLM, apiKey string) error { //nolint:gocyclo
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
