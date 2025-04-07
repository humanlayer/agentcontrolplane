package adapters

import (
	"encoding/json"
	"fmt"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
)

// ConvertMCPToolsToLLMClientTools converts KubeChain MCPTool objects to LLM client tool format
func ConvertMCPToolsToLLMClientTools(mcpTools []kubechainv1alpha1.MCPTool, serverName string) []llmclient.Tool {
	clientTools := make([]llmclient.Tool, 0, len(mcpTools))

	for _, tool := range mcpTools {
		// Create a function definition
		toolFunction := llmclient.ToolFunction{
			Name:        fmt.Sprintf("%s__%s", serverName, tool.Name),
			Description: tool.Description,
		}

		// Convert the input schema if available
		if tool.InputSchema.Raw != nil {
			var params llmclient.ToolFunctionParameters
			if err := json.Unmarshal(tool.InputSchema.Raw, &params); err == nil {
				toolFunction.Parameters = params
			} else {
				// Default to a simple object schema if none provided
				toolFunction.Parameters = llmclient.ToolFunctionParameters{
					Type:       "object",
					Properties: map[string]llmclient.ToolFunctionParameter{},
				}
			}
		} else {
			// Default to a simple object schema if none provided
			toolFunction.Parameters = llmclient.ToolFunctionParameters{
				Type:       "object",
				Properties: map[string]llmclient.ToolFunctionParameter{},
			}
		}

		// Create the tool with the function definition
		clientTools = append(clientTools, llmclient.Tool{
			Type:              "function",
			Function:          toolFunction,
			KubechainToolType: kubechainv1alpha1.ToolTypeMCP,
		})
	}

	return clientTools
}

// ParseToolArgumentsToMap converts the JSON arguments string to a map
func ParseToolArgumentsToMap(arguments string) (map[string]interface{}, error) {
	var argsMap map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &argsMap); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}
	return argsMap, nil
}
