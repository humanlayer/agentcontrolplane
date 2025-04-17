package validation

import (
	"fmt"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// ValidateTaskInput ensures exactly one of userMessage or contextWindow is provided
// and validates contextWindow contents if present
func ValidateTaskInput(userMessage string, contextWindow []acp.Message) error {
	if userMessage != "" && len(contextWindow) > 0 {
		return fmt.Errorf("only one of userMessage or contextWindow can be provided")
	}
	if userMessage == "" && len(contextWindow) == 0 {
		return fmt.Errorf("one of userMessage or contextWindow must be provided")
	}

	if len(contextWindow) > 0 {
		hasUserMessage := false
		for _, msg := range contextWindow {
			if !acp.ValidMessageRoles[msg.Role] {
				return fmt.Errorf("invalid role in contextWindow: %s", msg.Role)
			}
			if msg.Role == acp.MessageRoleUser {
				hasUserMessage = true
			}
		}
		if !hasUserMessage {
			return fmt.Errorf("contextWindow must contain at least one user message")
		}
	}
	return nil
}

// GetUserMessagePreview generates a preview from userMessage or the last user message in contextWindow
func GetUserMessagePreview(userMessage string, contextWindow []acp.Message) string {
	var preview string
	if userMessage != "" {
		preview = userMessage
	} else if len(contextWindow) > 0 {
		for i := len(contextWindow) - 1; i >= 0; i-- {
			if contextWindow[i].Role == acp.MessageRoleUser {
				preview = contextWindow[i].Content
				break
			}
		}
	}
	if len(preview) > 50 {
		preview = preview[:47] + "..."
	}
	return preview
}
