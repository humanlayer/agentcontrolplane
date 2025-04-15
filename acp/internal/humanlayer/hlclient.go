// Note, may eventually move on from the client.go in this project
// in which case I would rename this file to client.go
package humanlayer

import (
	"context"
	"fmt"
	"net/url"
	"os"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	humanlayerapi "github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi"
)

// NewHumanLayerClientFactory creates a new API client using either the provided API key
// or falling back to the HUMANLAYER_API_KEY environment variable. Similarly,
// it uses the provided API base URL or falls back to HUMANLAYER_API_BASE.
func NewHumanLayerClientFactory(optionalApiBase string) (HumanLayerClientFactory, error) {
	config := humanlayerapi.NewConfiguration()

	// Get API base from parameter or environment variable
	apiBase := os.Getenv("HUMANLAYER_API_BASE")
	if optionalApiBase != "" {
		apiBase = optionalApiBase
	}

	if apiBase == "" {
		apiBase = "https://api.humanlayer.dev"
	}

	parsedURL, err := url.Parse(apiBase)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API base URL: %v", err)
	}

	config.Host = parsedURL.Host
	config.Scheme = parsedURL.Scheme
	config.Servers = humanlayerapi.ServerConfigurations{
		{
			URL:         apiBase,
			Description: "HumanLayer API server",
		},
	}

	// Enable debug mode to log request/response
	// config.Debug = true

	// Create the API client with the configuration
	client := humanlayerapi.NewAPIClient(config)

	return &RealHumanLayerClientFactory{client: client}, nil
}

type HumanLayerClientWrapper interface {
	SetSlackConfig(slackConfig *acp.SlackChannelConfig)
	SetEmailConfig(emailConfig *acp.EmailChannelConfig)
	SetFunctionCallSpec(functionName string, args map[string]interface{})
	SetCallID(callID string)
	SetRunID(runID string)
	SetAPIKey(apiKey string)

	RequestApproval(ctx context.Context) (functionCall *humanlayerapi.FunctionCallOutput, statusCode int, err error)
	RequestHumanContact(ctx context.Context, userMsg string) (humanContact *humanlayerapi.HumanContactOutput, statusCode int, err error)
	GetFunctionCallStatus(ctx context.Context) (functionCall *humanlayerapi.FunctionCallOutput, statusCode int, err error)
	GetHumanContactStatus(ctx context.Context) (humanContact *humanlayerapi.HumanContactOutput, statusCode int, err error)
}

type HumanLayerClientFactory interface {
	NewHumanLayerClient() HumanLayerClientWrapper
}

type RealHumanLayerClientWrapper struct {
	client                *humanlayerapi.APIClient
	slackChannelInput     *humanlayerapi.SlackContactChannelInput
	emailContactChannel   *humanlayerapi.EmailContactChannel
	functionCallSpecInput *humanlayerapi.FunctionCallSpecInput
	callID                string
	runID                 string
	apiKey                string
}

type RealHumanLayerClientFactory struct {
	client *humanlayerapi.APIClient
}

func (h *RealHumanLayerClientFactory) NewHumanLayerClient() HumanLayerClientWrapper {
	return &RealHumanLayerClientWrapper{
		client: h.client,
	}
}

func (h *RealHumanLayerClientWrapper) SetSlackConfig(slackConfig *acp.SlackChannelConfig) {
	slackChannelInput := humanlayerapi.NewSlackContactChannelInput(slackConfig.ChannelOrUserID)

	if slackConfig.ContextAboutChannelOrUser != "" {
		slackChannelInput.SetContextAboutChannelOrUser(slackConfig.ContextAboutChannelOrUser)
	}

	h.slackChannelInput = slackChannelInput
}

func (h *RealHumanLayerClientWrapper) SetEmailConfig(emailConfig *acp.EmailChannelConfig) {
	emailContactChannel := humanlayerapi.NewEmailContactChannel(emailConfig.Address)

	if emailConfig.ContextAboutUser != "" {
		emailContactChannel.SetContextAboutUser(emailConfig.ContextAboutUser)
	}

	h.emailContactChannel = emailContactChannel
}

func (h *RealHumanLayerClientWrapper) SetFunctionCallSpec(functionName string, args map[string]interface{}) {
	// Create the function call input with required parameters
	functionCallSpecInput := humanlayerapi.NewFunctionCallSpecInput(functionName, args)

	h.functionCallSpecInput = functionCallSpecInput
}

func (h *RealHumanLayerClientWrapper) SetCallID(callID string) {
	h.callID = callID
}

func (h *RealHumanLayerClientWrapper) SetRunID(runID string) {
	h.runID = runID
}

func (h *RealHumanLayerClientWrapper) SetAPIKey(apiKey string) {
	h.apiKey = apiKey
}

func (h *RealHumanLayerClientWrapper) RequestApproval(ctx context.Context) (functionCall *humanlayerapi.FunctionCallOutput, statusCode int, err error) {
	channel := humanlayerapi.NewContactChannelInput()

	if h.slackChannelInput != nil {
		channel.SetSlack(*h.slackChannelInput)
	}

	if h.emailContactChannel != nil {
		channel.SetEmail(*h.emailContactChannel)
	}

	h.functionCallSpecInput.SetChannel(*channel)
	functionCallInput := humanlayerapi.NewFunctionCallInput(h.runID, h.callID, *h.functionCallSpecInput)

	functionCall, resp, err := h.client.DefaultAPI.RequestApproval(ctx).
		Authorization("Bearer " + h.apiKey).
		FunctionCallInput(*functionCallInput).
		Execute()

	return functionCall, resp.StatusCode, err
}

func (h *RealHumanLayerClientWrapper) RequestHumanContact(ctx context.Context, userMsg string) (humanContact *humanlayerapi.HumanContactOutput, statusCode int, err error) {
	channel := humanlayerapi.NewContactChannelInput()

	if h.slackChannelInput != nil {
		channel.SetSlack(*h.slackChannelInput)
	}

	if h.emailContactChannel != nil {
		channel.SetEmail(*h.emailContactChannel)
	}

	humanContactSpecInput := humanlayerapi.NewHumanContactSpecInput(userMsg)
	humanContactSpecInput.SetChannel(*channel)

	humanContactInput := humanlayerapi.NewHumanContactInput(h.runID, h.callID, *humanContactSpecInput)

	humanContact, resp, err := h.client.DefaultAPI.RequestHumanContact(ctx).
		Authorization("Bearer " + h.apiKey).
		HumanContactInput(*humanContactInput).
		Execute()

	return humanContact, resp.StatusCode, err
}

func (h *RealHumanLayerClientWrapper) GetFunctionCallStatus(ctx context.Context) (functionCall *humanlayerapi.FunctionCallOutput, statusCode int, err error) {
	functionCall, resp, err := h.client.DefaultAPI.GetFunctionCallStatus(ctx, h.callID).
		Authorization("Bearer " + h.apiKey).
		Execute()

	return functionCall, resp.StatusCode, err
}

func (h *RealHumanLayerClientWrapper) GetHumanContactStatus(ctx context.Context) (humanContact *humanlayerapi.HumanContactOutput, statusCode int, err error) {
	humanContact, resp, err := h.client.DefaultAPI.GetHumanContactStatus(ctx, h.callID).
		Authorization("Bearer " + h.apiKey).
		Execute()

	return humanContact, resp.StatusCode, err
}
