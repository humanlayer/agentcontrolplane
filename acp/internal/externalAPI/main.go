package externalapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// ClientFactory is a function type for creating external API clients
type ClientFactory func(secretData map[string][]byte, apiKeyField string) (Client, error)

// Registry manages external API client creation
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ClientFactory
}

// NewRegistry creates a new client registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ClientFactory),
	}
}

// Register adds a new client factory to the registry
func (r *Registry) Register(toolName string, factory ClientFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[toolName] = factory
}

// GetClient retrieves and instantiates a client for a specific tool
func (r *Registry) GetClient(
	toolName string,
	k8sClient client.Client,
	namespace string,
	credentialsRef *acp.SecretKeyRef,
) (Client, error) {
	r.mu.RLock()
	factory, exists := r.factories[toolName]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no client factory registered for tool: %s", toolName)
	}

	// Fetch the secret
	var secret corev1.Secret
	if err := k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      credentialsRef.Name,
	}, &secret); err != nil {
		return nil, fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// Use the factory to create the client
	return factory(secret.Data, credentialsRef.Key)
}

// DefaultRegistry is a global registry for external API clients
var DefaultRegistry = NewRegistry()

// Client defines the interface for external API interactions
type Client interface {
	// Call executes a function call with the given parameters
	Call(ctx context.Context, runID, callID string, spec map[string]interface{}) (json.RawMessage, error)
}
