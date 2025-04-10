//go:build mock
// +build mock

// This file is only built when the 'mock' build tag is used
// It contains the mock K8s client implementation for testing secret handling

package mcpmanager

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockClient is a minimal implementation of client.Client for testing
type MockClient struct {
	secrets map[types.NamespacedName]*corev1.Secret
}

// NewMockClient creates a new mock client
func NewMockClient() *MockClient {
	return &MockClient{
		secrets: make(map[types.NamespacedName]*corev1.Secret),
	}
}

// AddSecret adds a secret to the mock client
func (m *MockClient) AddSecret(namespace, name string, data map[string][]byte) {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	m.secrets[key] = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: data,
	}
}

// Get implements client.Client.Get
func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	// Only handle Secret resources
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return fmt.Errorf("not a secret")
	}

	// Look up the secret
	nsName := types.NamespacedName{Namespace: key.Namespace, Name: key.Name}
	s, exists := m.secrets[nsName]
	if !exists {
		return fmt.Errorf("secret not found: %s/%s", key.Namespace, key.Name)
	}

	// Copy data to the result
	secret.Data = s.Data
	secret.ObjectMeta = s.ObjectMeta
	return nil
}

// Stub implementations for the rest of client.Client interface
func (m *MockClient) Create(context.Context, client.Object, ...client.CreateOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockClient) Delete(context.Context, client.Object, ...client.DeleteOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockClient) Update(context.Context, client.Object, ...client.UpdateOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockClient) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockClient) DeleteAllOf(context.Context, client.Object, ...client.DeleteAllOfOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockClient) List(context.Context, client.ObjectList, ...client.ListOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockClient) Status() client.StatusWriter {
	return nil
}

func (m *MockClient) Scheme() *runtime.Scheme {
	return nil
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	return nil
}
