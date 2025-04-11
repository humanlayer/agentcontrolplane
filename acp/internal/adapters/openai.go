package adapters

import (
	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// CastOpenAIToolCallsToACP converts OpenAI tool calls to TaskRun tool calls
func CastOpenAIToolCallsToACP(openaiToolCalls []acp.MessageToolCall) []acp.MessageToolCall {
	toolCalls := make([]acp.MessageToolCall, 0, len(openaiToolCalls))
	for _, tc := range openaiToolCalls {
		toolCall := acp.MessageToolCall{
			ID: tc.ID,
			Function: acp.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
			Type: tc.Type,
		}
		toolCalls = append(toolCalls, toolCall)
	}
	return toolCalls
}
