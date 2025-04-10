package adapters

import (
	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// CastOpenAIToolCallsToKubechain converts OpenAI tool calls to TaskRun tool calls
func CastOpenAIToolCallsToKubechain(openaiToolCalls []acp.ToolCall) []acp.ToolCall {
	toolCalls := make([]acp.ToolCall, 0, len(openaiToolCalls))
	for _, tc := range openaiToolCalls {
		toolCall := acp.ToolCall{
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
