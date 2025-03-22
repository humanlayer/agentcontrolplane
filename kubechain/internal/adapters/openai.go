package adapters

import (
	"github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// CastOpenAIToolCallsToKubechain converts OpenAI tool calls to TaskRun tool calls
func CastOpenAIToolCallsToKubechain(openaiToolCalls []v1alpha1.ToolCall) []kubechainv1alpha1.ToolCall {
	var toolCalls []kubechainv1alpha1.ToolCall
	for _, tc := range openaiToolCalls {
		toolCall := kubechainv1alpha1.ToolCall{
			ID: tc.ID,
			Function: kubechainv1alpha1.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
			Type: string(tc.Type),
		}
		toolCalls = append(toolCalls, toolCall)
	}
	return toolCalls
}
