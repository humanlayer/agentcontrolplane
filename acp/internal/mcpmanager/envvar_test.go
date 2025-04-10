//go:build secret
// +build secret

// This file is only built when the 'secret' build tag is used
// It contains tests for the secret handling functionality

package mcpmanager

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

// MockRESTMapper is a minimal implementation of apimeta.RESTMapper for testing
type MockRESTMapper struct{}

func (m *MockRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*apimeta.RESTMapping, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m *MockRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	return nil, nil
}

func (m *MockRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{}, nil
}

func (m *MockRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	return nil, nil
}

func (m *MockRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*apimeta.RESTMapping, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRESTMapper) ResourceSingularizer(resource string) (string, error) {
	return "", nil
}

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
func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	// Only handle Secret resources
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return fmt.Errorf("not a secret: got %T", obj)
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
	return &MockStatusWriter{}
}

// MockStatusWriter is a minimal implementation of client.StatusWriter
type MockStatusWriter struct{}

func (m *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockClient) Scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	// Add core types to scheme
	corev1.AddToScheme(scheme)
	return scheme
}

// Additional methods needed by the client.Client interface
func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m *MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return true, nil
}

func (m *MockClient) RESTMapper() apimeta.RESTMapper {
	return &MockRESTMapper{}
}

func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	return &MockSubResourceClient{}
}

// MockSubResourceClient is a minimal implementation of client.SubResourceClient
type MockSubResourceClient struct{}

func (m *MockSubResourceClient) Get(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceGetOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockSubResourceClient) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockSubResourceClient) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return fmt.Errorf("not implemented")
}

func (m *MockSubResourceClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return fmt.Errorf("not implemented")
}

var _ = Describe("Environment Variable Handling", func() {
	var (
		manager    *MCPServerManager
		mockClient *MockClient
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient = NewMockClient()

		// Add test secrets to the mock client
		mockClient.AddSecret("default", "test-secret", map[string][]byte{
			"api-key": []byte("secret-value"),
		})

		// Create the manager with the mock client
		manager = NewMCPServerManagerWithClient(mockClient)
	})

	Describe("convertEnvVars", func() {
		It("should process direct environment variables", func() {
			// Create test env vars with direct values
			envVars := []acp.EnvVar{
				{
					Name:  "TEST_ENV1",
					Value: "value1",
				},
				{
					Name:  "TEST_ENV2",
					Value: "value2",
				},
			}

			// Process env vars
			result, err := manager.convertEnvVars(ctx, envVars, "default")

			// Verify results
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainElement("TEST_ENV1=value1"))
			Expect(result).To(ContainElement("TEST_ENV2=value2"))
		})

		It("should process environment variables from secrets", func() {
			// Create reference to the test secret
			envVars := []acp.EnvVar{
				{
					Name: "API_KEY",
					ValueFrom: &acp.EnvVarSource{
						SecretKeyRef: &acp.SecretKeySelector{
							Name: "test-secret",
							Key:  "api-key",
						},
					},
				},
			}

			// Process env vars
			result, err := manager.convertEnvVars(ctx, envVars, "default")

			// Verify results
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainElement("API_KEY=secret-value"))
		})

		It("should handle mixed direct values and secret references", func() {
			// Create test env vars with both types
			envVars := []acp.EnvVar{
				{
					Name:  "DIRECT_VAR",
					Value: "direct-value",
				},
				{
					Name: "SECRET_VAR",
					ValueFrom: &acp.EnvVarSource{
						SecretKeyRef: &acp.SecretKeySelector{
							Name: "test-secret",
							Key:  "api-key",
						},
					},
				},
			}

			// Process env vars
			result, err := manager.convertEnvVars(ctx, envVars, "default")

			// Verify results
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainElement("DIRECT_VAR=direct-value"))
			Expect(result).To(ContainElement("SECRET_VAR=secret-value"))
		})

		It("should return error for non-existent secret", func() {
			// Create reference to a non-existent secret
			envVars := []acp.EnvVar{
				{
					Name: "MISSING_SECRET",
					ValueFrom: &acp.EnvVarSource{
						SecretKeyRef: &acp.SecretKeySelector{
							Name: "non-existent-secret",
							Key:  "api-key",
						},
					},
				},
			}

			// Process env vars
			_, err := manager.convertEnvVars(ctx, envVars, "default")

			// Verify error
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get secret"))
		})

		It("should return error for non-existent key in secret", func() {
			// Create reference to a non-existent key
			envVars := []acp.EnvVar{
				{
					Name: "MISSING_KEY",
					ValueFrom: &acp.EnvVarSource{
						SecretKeyRef: &acp.SecretKeySelector{
							Name: "test-secret",
							Key:  "non-existent-key",
						},
					},
				},
			}

			// Process env vars
			_, err := manager.convertEnvVars(ctx, envVars, "default")

			// Verify error
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found in secret"))
		})

		It("should return error when client is nil and secret references are used", func() {
			// Create manager without client
			managerNoClient := NewMCPServerManager() // No client provided

			// Create reference to a secret
			envVars := []acp.EnvVar{
				{
					Name: "API_KEY",
					ValueFrom: &acp.EnvVarSource{
						SecretKeyRef: &acp.SecretKeySelector{
							Name: "test-secret",
							Key:  "api-key",
						},
					},
				},
			}

			// Process env vars
			_, err := managerNoClient.convertEnvVars(ctx, envVars, "default")

			// Verify error
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no Kubernetes client available"))
		})
	})
})
