package humanlayer

import (
	"context"
	"time"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	humanlayerapi "github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi"
)

// MockHumanLayerClientFactory implements HumanLayerClientFactory for testing
type MockHumanLayerClientFactory struct {
	ShouldFail            bool
	StatusCode            int
	ReturnError           error
	ShouldReturnApproval  bool
	ShouldReturnRejection bool
	LastAPIKey            string
	LastCallID            string
	LastRunID             string
	LastFunction          string
	LastArguments         map[string]interface{}
	StatusComment         string
}

// MockHumanLayerClientWrapper implements HumanLayerClientWrapper for testing
type MockHumanLayerClientWrapper struct {
	parent       *MockHumanLayerClientFactory
	slackConfig  *acp.SlackChannelConfig
	emailConfig  *acp.EmailChannelConfig
	functionName string
	functionArgs map[string]interface{}
	callID       string
	runID        string
	apiKey       string
}

// NewHumanLayerClient creates a new mock client
func NewMockHumanLayerClient(shouldFail bool, statusCode int, returnError error) *MockHumanLayerClientFactory {
	return &MockHumanLayerClientFactory{
		ShouldFail:  shouldFail,
		StatusCode:  statusCode,
		ReturnError: returnError,
	}
}

// NewHumanLayerClient implements HumanLayerClientFactory
func (m *MockHumanLayerClientFactory) NewHumanLayerClient() HumanLayerClientWrapper {
	return &MockHumanLayerClientWrapper{
		parent: m,
	}
}

// SetSlackConfig implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) SetSlackConfig(slackConfig *acp.SlackChannelConfig) {
	m.slackConfig = slackConfig
}

// SetEmailConfig implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) SetEmailConfig(emailConfig *acp.EmailChannelConfig) {
	m.emailConfig = emailConfig
}

// SetFunctionCallSpec implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) SetFunctionCallSpec(functionName string, args map[string]interface{}) {
	m.functionName = functionName
	m.functionArgs = args
}

// SetCallID implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) SetCallID(callID string) {
	m.callID = callID
}

// SetRunID implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) SetRunID(runID string) {
	m.runID = runID
}

// SetAPIKey implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) SetAPIKey(apiKey string) {
	m.apiKey = apiKey
}

// SetThreadID implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) SetThreadID(threadID string) {
	// Mock implementation - just store it if needed for testing
}

// GetFunctionCallStatus implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) GetFunctionCallStatus(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
	if m.parent.ShouldReturnApproval {
		now := time.Now()
		approved := true
		status := humanlayerapi.NewNullableFunctionCallStatus(&humanlayerapi.FunctionCallStatus{
			RequestedAt: *humanlayerapi.NewNullableTime(&now),
			RespondedAt: *humanlayerapi.NewNullableTime(&now),
			Approved:    *humanlayerapi.NewNullableBool(&approved),
		})
		return &humanlayerapi.FunctionCallOutput{
			Status: *status,
		}, 200, nil
	}

	if m.parent.ShouldReturnRejection {
		now := time.Now()
		approved := false
		status := humanlayerapi.NewNullableFunctionCallStatus(&humanlayerapi.FunctionCallStatus{
			RequestedAt: *humanlayerapi.NewNullableTime(&now),
			RespondedAt: *humanlayerapi.NewNullableTime(&now),
			Approved:    *humanlayerapi.NewNullableBool(&approved),
			Comment:     *humanlayerapi.NewNullableString(&m.parent.StatusComment),
		})
		return &humanlayerapi.FunctionCallOutput{
			Status: *status,
		}, 200, nil
	}

	return nil, m.parent.StatusCode, m.parent.ReturnError
}

// RequestApproval implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) RequestApproval(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
	// Store the values in the parent for test verification
	m.parent.LastAPIKey = m.apiKey
	m.parent.LastCallID = m.callID
	m.parent.LastRunID = m.runID
	m.parent.LastFunction = m.functionName
	m.parent.LastArguments = m.functionArgs

	if m.parent.ShouldFail {
		return nil, m.parent.StatusCode, m.parent.ReturnError
	}

	// Return a successful mock response
	return &humanlayerapi.FunctionCallOutput{}, m.parent.StatusCode, nil
}

// RequestHumanContact implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) RequestHumanContact(ctx context.Context, userMsg string) (*humanlayerapi.HumanContactOutput, int, error) {
	if m.parent.ShouldFail {
		return nil, m.parent.StatusCode, m.parent.ReturnError
	}

	// Return a successful mock response
	output := humanlayerapi.NewHumanContactOutput(m.runID, m.callID, *humanlayerapi.NewHumanContactSpecOutput(userMsg))
	return output, m.parent.StatusCode, nil
}

// GetHumanContactStatus implements HumanLayerClientWrapper
func (m *MockHumanLayerClientWrapper) GetHumanContactStatus(ctx context.Context) (*humanlayerapi.HumanContactOutput, int, error) {
	return nil, m.parent.StatusCode, m.parent.ReturnError
}
