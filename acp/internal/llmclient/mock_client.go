package llmclient

import (
	"context"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// MockLLMClient is a mock implementation of LLMClient for testing
type MockLLMClient struct {
	Response              *acp.Message
	Error                 error
	Calls                 []MockCall
	ValidateTools         func(tools []Tool) error
	ValidateContextWindow func(contextWindow []acp.Message) error
}

type MockCall struct {
	Messages []acp.Message
	Tools    []Tool
}

// SendRequest implements the LLMClient interface
func (m *MockLLMClient) SendRequest(ctx context.Context, messages []acp.Message, tools []Tool) (*acp.Message, error) {
	m.Calls = append(m.Calls, MockCall{
		Messages: messages,
		Tools:    tools,
	})

	if m.ValidateTools != nil {
		if err := m.ValidateTools(tools); err != nil {
			return nil, err
		}
	}

	if m.ValidateContextWindow != nil {
		if err := m.ValidateContextWindow(messages); err != nil {
			return nil, err
		}
	}

	if m.Error != nil {
		return m.Response, m.Error
	}

	if m.Response == nil {
		return &acp.Message{
			Role:    "assistant",
			Content: "Mock response",
		}, nil
	}

	return m.Response, m.Error
}
