package llmclient

import (
	"context"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// MockLLMClient is a mock implementation of LLMClient for testing
type MockLLMClient struct {
	Response              *kubechainv1alpha1.Message
	Error                 error
	Calls                 []MockCall
	ValidateTools         func(tools []Tool) error
	ValidateContextWindow func(contextWindow []kubechainv1alpha1.Message) error
}

type MockCall struct {
	Messages []kubechainv1alpha1.Message
	Tools    []Tool
}

// SendRequest implements the LLMClient interface
func (m *MockLLMClient) SendRequest(ctx context.Context, messages []kubechainv1alpha1.Message, tools []Tool) (*kubechainv1alpha1.Message, error) {
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
		return &kubechainv1alpha1.Message{
			Role:    "assistant",
			Content: "Mock response",
		}, nil
	}

	return m.Response, m.Error
}
