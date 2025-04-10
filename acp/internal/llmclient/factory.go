package llmclient

import (
	"context"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// NewLLMClient creates a new LLM client based on the LLM configuration
func NewLLMClient(ctx context.Context, llm acp.LLM, apiKey string) (LLMClient, error) {
	return NewLangchainClient(ctx, llm.Spec.Provider, apiKey, llm.Spec.Parameters)
}
