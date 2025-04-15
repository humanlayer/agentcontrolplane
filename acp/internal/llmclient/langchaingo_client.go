package llmclient

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/googleai/vertex"
	"github.com/tmc/langchaingo/llms/mistral"
	"github.com/tmc/langchaingo/llms/openai"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// LangchainClient implements the LLMClient interface using langchaingo

var _ LLMClient = &LangchainClient{}

type LangchainClient struct {
	model llms.Model
}

// NewLangchainClient creates a new client using the specified provider and credentials
func NewLangchainClient(ctx context.Context, provider string, apiKey string, modelConfig acp.BaseConfig) (LLMClient, error) {
	var model llms.Model
	var err error

	switch provider {
	case "openai":
		opts := []openai.Option{openai.WithToken(apiKey)}
		if modelConfig.Model != "" {
			opts = append(opts, openai.WithModel(modelConfig.Model))
		}
		if modelConfig.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(modelConfig.BaseURL))
		}
		model, err = openai.New(opts...)
	case "anthropic":
		opts := []anthropic.Option{anthropic.WithToken(apiKey)}
		if modelConfig.Model != "" {
			opts = append(opts, anthropic.WithModel(modelConfig.Model))
		}
		if modelConfig.BaseURL != "" {
			opts = append(opts, anthropic.WithBaseURL(modelConfig.BaseURL))
		}
		model, err = anthropic.New(opts...)
	case "mistral":
		opts := []mistral.Option{mistral.WithAPIKey(apiKey)}
		if modelConfig.Model != "" {
			opts = append(opts, mistral.WithModel(modelConfig.Model))
		}
		if modelConfig.BaseURL != "" {
			opts = append(opts, mistral.WithEndpoint(modelConfig.BaseURL))
		}
		model, err = mistral.New(opts...)
	case "google":
		opts := []googleai.Option{googleai.WithAPIKey(apiKey)}
		if modelConfig.Model != "" {
			opts = append(opts, googleai.WithDefaultModel(modelConfig.Model))
		}
		model, err = googleai.New(context.Background(), opts...)
	case "vertex":
		opts := []googleai.Option{googleai.WithCredentialsJSON([]byte(apiKey))}
		if modelConfig.Model != "" {
			opts = append(opts, googleai.WithDefaultModel(modelConfig.Model))
		}
		model, err = vertex.New(context.Background(), opts...)
	default:
		return nil, fmt.Errorf("unsupported provider: %s. Supported providers are: openai, anthropic, mistral, google, vertex", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s client: %w", provider, err)
	}

	return &LangchainClient{model: model}, nil
}

// SendRequest implements the LLMClient interface
func (c *LangchainClient) SendRequest(ctx context.Context, messages []acp.Message, tools []Tool) (*acp.Message, error) {
	logger := log.FromContext(ctx)

	// Convert messages to langchaingo format
	langchainMessages := convertToLangchainMessages(messages)

	// Convert tools to langchaingo format
	langchainTools := convertToLangchainTools(tools)

	// Prepare options
	options := []llms.CallOption{}
	if len(langchainTools) > 0 {
		options = append(options, llms.WithTools(langchainTools))
		logger.V(1).Info("Sending tools to LLM",
			"modelType", fmt.Sprintf("%T", c.model),
			"toolCount", len(langchainTools))
	}

	// Make the API call
	response, err := c.model.GenerateContent(ctx, langchainMessages, options...)
	if err != nil {
		return nil, &LLMRequestError{
			StatusCode: response.StatusCode,
			Message:    fmt.Sprintf("langchain API call failed: %v", err),
			Err:        err,
		}
	}

	// Log response characteristics for debugging
	if len(response.Choices) > 1 {
		logger.V(1).Info("LLM returned multiple choices",
			"choiceCount", len(response.Choices))
	}

	// Convert response back to ACP format
	return convertFromLangchainResponse(response), nil
}

// convertToLangchainMessages converts ACP messages to langchaingo format
func convertToLangchainMessages(messages []acp.Message) []llms.MessageContent {
	langchainMessages := make([]llms.MessageContent, 0, len(messages))

	for _, message := range messages {
		var role llms.ChatMessageType

		// Convert role
		switch message.Role {
		case "system":
			role = llms.ChatMessageTypeSystem
		case "user":
			role = llms.ChatMessageTypeHuman
		case "assistant":
			role = llms.ChatMessageTypeAI
		case "tool":
			role = llms.ChatMessageTypeTool
		default:
			role = llms.ChatMessageTypeHuman
		}

		// Create a message content with text and/or tool calls
		msgContent := llms.MessageContent{
			Role: role,
		}

		// Add text content if present
		if message.Content != "" {
			msgContent.Parts = append(msgContent.Parts, llms.TextContent{
				Text: message.Content,
			})
		}

		// Add tool calls if present
		for _, toolCall := range message.ToolCalls {
			msgContent.Parts = append(msgContent.Parts, llms.ToolCall{
				ID:   toolCall.ID,
				Type: toolCall.Type,
				FunctionCall: &llms.FunctionCall{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			})
		}

		// Add tool response if present
		if message.ToolCallID != "" {
			// For tool role, only have a ToolCallResponse part
			if role == llms.ChatMessageTypeTool {
				msgContent.Parts = []llms.ContentPart{
					llms.ToolCallResponse{
						ToolCallID: message.ToolCallID,
						Content:    message.Content,
					},
				}
			} else {
				// For other roles, append the tool call response
				msgContent.Parts = append(msgContent.Parts, llms.ToolCallResponse{
					ToolCallID: message.ToolCallID,
					Content:    message.Content,
				})
			}
		}

		langchainMessages = append(langchainMessages, msgContent)
	}

	return langchainMessages
}

// convertToLangchainTools converts ACP tools to langchaingo format
func convertToLangchainTools(tools []Tool) []llms.Tool {
	langchainTools := make([]llms.Tool, 0, len(tools))

	for _, tool := range tools {
		langchainTools = append(langchainTools, llms.Tool{
			Type: tool.Type,
			Function: &llms.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		})
	}

	return langchainTools
}

// convertFromLangchainResponse converts a langchaingo response to ACP format.
// It handles different response structures from various LLM providers by
// collecting all tool calls from all choices.
func convertFromLangchainResponse(response *llms.ContentResponse) *acp.Message {
	// Get logger for this context - using package logger since we don't have access to ctx
	logger := log.Log.WithName("langchaingo")

	// Create base message with assistant role
	message := &acp.Message{
		Role: "assistant",
	}

	// Handle empty response
	if len(response.Choices) == 0 {
		logger.V(1).Info("LLM returned an empty response with no choices")
		message.Content = ""
		return message
	}

	// Extract all tool calls across all choices (provider-agnostic)
	var toolCalls []acp.MessageToolCall
	var contentText string
	var hasContent bool

	// Process all choices to collect content and tool calls
	for i, choice := range response.Choices {
		// Extract content from the first non-empty choice
		if !hasContent && choice.Content != "" {
			contentText = choice.Content
			hasContent = true
			logger.V(2).Info("Found content in choice",
				"choiceIndex", i,
				"contentPreview", truncateString(choice.Content, 50))
		}

		// Extract tool calls from this choice
		if len(choice.ToolCalls) > 0 {
			logger.V(2).Info("Found tool calls in choice",
				"choiceIndex", i,
				"toolCallCount", len(choice.ToolCalls))

			for _, tc := range choice.ToolCalls {
				toolCalls = append(toolCalls, acp.MessageToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: acp.ToolCallFunction{
						Name:      tc.FunctionCall.Name,
						Arguments: tc.FunctionCall.Arguments,
					},
				})
			}
		}
	}

	// Prioritize tool calls if present
	if len(toolCalls) > 0 {
		if hasContent {
			logger.V(1).Info("LLM returned both content and tool calls - prioritizing tool calls")
		}

		message.ToolCalls = toolCalls
		// Clear content when there are tool calls to ensure controller
		// takes the tool call execution path
		message.Content = ""
		return message
	}

	// Fall back to content if available
	if hasContent {
		message.Content = contentText
		return message
	}

	// Handle edge case where no content or tool calls were found
	logger.V(1).Info("LLM returned choices with neither content nor tool calls")
	message.Content = ""
	return message
}

// truncateString truncates a string to the specified length if needed
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}
