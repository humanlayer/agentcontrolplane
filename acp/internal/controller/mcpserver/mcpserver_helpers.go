package mcpserver

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// validateMCPServerSpec validates server configuration with actionable error messages
func validateMCPServerSpec(spec acp.MCPServerSpec) error {
	if spec.Transport != "stdio" && spec.Transport != "http" {
		return fmt.Errorf("transport must be 'stdio' or 'http', got '%s'", spec.Transport)
	}

	if spec.Transport == "stdio" && spec.Command == "" {
		return fmt.Errorf("command is required for stdio transport - specify the executable path or command")
	}

	if spec.Transport == "http" && spec.URL == "" {
		return fmt.Errorf("url is required for http transport - specify the MCP server endpoint")
	}

	return nil
}

// processEnvVars handles environment variable resolution with clear error messages
func processEnvVars(ctx context.Context, client client.Client, envVars []acp.EnvVar, namespace string) ([]string, error) {
	if len(envVars) == 0 {
		return nil, nil
	}

	env := make([]string, 0, len(envVars))
	for _, e := range envVars {
		if e.Name == "" {
			continue // Skip invalid env vars
		}

		// Direct value (simple case)
		if e.Value != "" {
			env = append(env, fmt.Sprintf("%s=%s", e.Name, e.Value))
			continue
		}

		// Secret reference (complex case)
		if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
			value, err := resolveSecretValue(ctx, client, e.ValueFrom.SecretKeyRef, namespace)
			if err != nil {
				return nil, fmt.Errorf("env var %s: %w", e.Name, err)
			}
			env = append(env, fmt.Sprintf("%s=%s", e.Name, value))
		}
	}
	return env, nil
}

// resolveSecretValue gets a value from a Kubernetes secret
func resolveSecretValue(ctx context.Context, client client.Client, secretRef *acp.SecretKeyRef, namespace string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("cannot resolve secret %s - no Kubernetes client", secretRef.Name)
	}

	var secret corev1.Secret
	if err := client.Get(ctx, types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: namespace,
	}, &secret); err != nil {
		return "", fmt.Errorf("secret %s not found in namespace %s", secretRef.Name, namespace)
	}

	secretValue, exists := secret.Data[secretRef.Key]
	if !exists {
		return "", fmt.Errorf("key %s not found in secret %s", secretRef.Key, secretRef.Name)
	}

	return string(secretValue), nil
}

// validateContactChannelReference validates that the approval contact channel exists and is ready
func validateContactChannelReference(ctx context.Context, client client.Client, mcpServer *acp.MCPServer) error {
	if mcpServer.Spec.ApprovalContactChannel == nil {
		return nil // No contact channel required
	}

	approvalContactChannel := &acp.ContactChannel{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      mcpServer.Spec.ApprovalContactChannel.Name,
		Namespace: mcpServer.Namespace,
	}, approvalContactChannel)
	if err != nil {
		return fmt.Errorf("ContactChannel %q not found: %w", mcpServer.Spec.ApprovalContactChannel.Name, err)
	}

	if !approvalContactChannel.Status.Ready {
		return fmt.Errorf("ContactChannel %q is not ready", mcpServer.Spec.ApprovalContactChannel.Name)
	}

	return nil
}

// toolsChanged compares two tool lists to see if they differ
func toolsChanged(oldTools, newTools []acp.MCPTool) bool {
	if len(oldTools) != len(newTools) {
		return true
	}

	// Create a simple map for comparison
	oldNames := make(map[string]bool, len(oldTools))
	for _, tool := range oldTools {
		oldNames[tool.Name] = true
	}

	// Check if any new tool is missing from old list
	for _, tool := range newTools {
		if !oldNames[tool.Name] {
			return true
		}
	}

	return false
}
