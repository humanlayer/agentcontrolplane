package task

import (
	"fmt"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	"go.opentelemetry.io/otel/trace"
)

// buildInitialContextWindow constructs the context window with proper system message handling
// This is a pure function that takes input parameters and returns a context window without side effects
func buildInitialContextWindow(contextWindow []acp.Message, systemPrompt, userMessage string) []acp.Message {
	var initialContextWindow []acp.Message

	if len(contextWindow) > 0 {
		// Copy existing context window
		initialContextWindow = append([]acp.Message{}, contextWindow...)

		// Check if system message already exists
		hasSystemMessage := false
		for _, msg := range initialContextWindow {
			if msg.Role == acp.MessageRoleSystem {
				hasSystemMessage = true
				break
			}
		}

		// Prepend system message if not present
		if !hasSystemMessage {
			initialContextWindow = append([]acp.Message{
				{Role: acp.MessageRoleSystem, Content: systemPrompt},
			}, initialContextWindow...)
		}
	} else {
		// Create new context window with system and user messages
		initialContextWindow = []acp.Message{
			{Role: acp.MessageRoleSystem, Content: systemPrompt},
			{Role: acp.MessageRoleUser, Content: userMessage},
		}
	}

	return initialContextWindow
}

// buildToolTypeMap creates a quick lookup map for tool types from a slice of tools
// This is a pure function for fast tool type resolution during tool call creation
func buildToolTypeMap(tools []llmclient.Tool) map[string]acp.ToolType {
	toolTypeMap := make(map[string]acp.ToolType)
	for _, tool := range tools {
		toolTypeMap[tool.Function.Name] = tool.ACPToolType
	}
	return toolTypeMap
}

// reconstructSpanContext safely reconstructs trace span context from string IDs
// This is a pure function that handles the conversion without side effects
func reconstructSpanContext(traceID, spanID string) (trace.SpanContext, error) {
	traceIDParsed, err := trace.TraceIDFromHex(traceID)
	if err != nil {
		return trace.SpanContext{}, err
	}

	spanIDParsed, err := trace.SpanIDFromHex(spanID)
	if err != nil {
		return trace.SpanContext{}, err
	}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceIDParsed,
		SpanID:     spanIDParsed,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})

	if !sc.IsValid() {
		return trace.SpanContext{}, fmt.Errorf("invalid span context")
	}

	return sc, nil
}
