package llmclient

import (
	"context"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// NewLLMClient creates a new LLM client based on the LLM configuration
func NewLLMClient(ctx context.Context, llm kubechainv1alpha1.LLM, apiKey string) (LLMClient, error) {
	return NewLangchainClient(ctx, llm.Spec.Provider, apiKey, llm.Spec.Parameters)
}
