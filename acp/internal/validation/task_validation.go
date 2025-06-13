package validation

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateTaskMessageInput ensures exactly one of userMessage or contextWindow is provided
// and validates contextWindow contents if present. This function validates the input
// mechanism used to create a Task (either through userMessage or contextWindow).
func ValidateTaskMessageInput(userMessage string, contextWindow []acp.Message) error {
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

// GenerateK8sRandomString returns a k8s-compliant secure random string (6-8 chars, lowercase letters and numbers, starts with letter)
func GenerateK8sRandomString(n int) (string, error) {
	if n < 1 || n > 8 {
		n = 6 // Default to 6 characters for k8s style
	}

	const letters = "abcdefghijklmnopqrstuvwxyz"
	const alphanumeric = "abcdefghijklmnopqrstuvwxyz0123456789"

	ret := make([]byte, n)

	// First character must be a letter (k8s naming convention)
	num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
	if err != nil {
		return "", err
	}
	ret[0] = letters[num.Int64()]

	// Remaining characters can be letters or numbers
	for i := 1; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphanumeric))))
		if err != nil {
			return "", err
		}
		ret[i] = alphanumeric[num.Int64()]
	}
	return string(ret), nil
}

// ValidateContactChannelRef validates that the referenced ContactChannel exists and is ready
func ValidateContactChannelRef(ctx context.Context, k8sClient client.Client, task *acp.Task) error {
	if task.Spec.ContactChannelRef == nil {
		return nil // No contactChannelRef is valid
	}

	var contactChannel acp.ContactChannel
	err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: task.Namespace,
		Name:      task.Spec.ContactChannelRef.Name,
	}, &contactChannel)
	if err != nil {
		return fmt.Errorf("referenced ContactChannel %q not found: %w", task.Spec.ContactChannelRef.Name, err)
	}

	if !contactChannel.Status.Ready {
		return fmt.Errorf("referenced ContactChannel %q is not ready (status: %s)",
			task.Spec.ContactChannelRef.Name, contactChannel.Status.Status)
	}

	return nil
}
