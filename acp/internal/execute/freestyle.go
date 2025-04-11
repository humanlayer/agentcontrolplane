package execute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	freestyleAPIEndpoint = "https://api.freestyle.sh/execute/v1/script"
)

// FreestyleConfig represents the configuration for a Freestyle script execution
type FreestyleConfig struct {
	EnvVars                  map[string]string `json:"envVars,omitempty"`
	NodeModules              map[string]string `json:"nodeModules,omitempty"`
	Tags                     []string          `json:"tags,omitempty"`
	Timeout                  *time.Duration    `json:"timeout,omitempty"`
	PeerDependencyResolution bool              `json:"peerDependencyResolution,omitempty"`
	NetworkPermissions       interface{}       `json:"networkPermissions,omitempty"`
	CustomHeaders            map[string]string `json:"customHeaders,omitempty"`
	Proxy                    string            `json:"proxy,omitempty"`
}

// FreestyleRequest represents the request body for the Freestyle API
type FreestyleRequest struct {
	Script string          `json:"script"`
	Config FreestyleConfig `json:"config"`
}

// FreestyleResponse represents the response from the Freestyle API
type FreestyleResponse struct {
	// Add response fields as needed based on the API response
	// This is a placeholder and should be updated with actual response structure
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// FreestyleClient is a client for interacting with the Freestyle API
type FreestyleClient struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

// NewFreestyleClient creates a new Freestyle client
func NewFreestyleClient(apiKey string) *FreestyleClient {
	return &FreestyleClient{
		apiKey:  apiKey,
		client:  &http.Client{},
		baseURL: freestyleAPIEndpoint,
	}
}

// ExecuteScript executes a JavaScript script using the Freestyle API
func (c *FreestyleClient) ExecuteScript(ctx context.Context, script string, config FreestyleConfig) (*FreestyleResponse, error) {
	req := FreestyleRequest{
		Script: script,
		Config: config,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var freestyleResp FreestyleResponse
	if err := json.NewDecoder(resp.Body).Decode(&freestyleResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &freestyleResp, nil
}

// SecretSelector represents a selector for Kubernetes secrets
type SecretSelector struct {
	Name        string            `json:"name,omitempty"`
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// CollectAvailableExecuteSecrets collects all available secrets that match the given selectors
// and returns a list of environment variable names that can be used in script execution
func CollectAvailableExecuteSecrets(ctx context.Context, r client.Client, namespace string, secretSelectors []SecretSelector) ([]string, error) {
	logger := log.FromContext(ctx)
	allowedSecrets := make([]string, 0)

	for _, selector := range secretSelectors {
		if selector.Name != "" {
			secret := &corev1.Secret{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: selector.Name}, secret); err != nil {
				logger.Error(err, "Failed to get secret", "name", selector.Name)
				continue
			}
			// get the keys that look like env vars
			for key := range secret.Data {
				if match, _ := regexp.MatchString(`^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$`, key); match {
					allowedSecrets = append(allowedSecrets, key)
				}
			}
		} else if selector.MatchLabels != nil {
			// use a label selector to get a SecretList that matches the labels
			secretList := &corev1.SecretList{}
			if err := r.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels(selector.MatchLabels)); err != nil {
				logger.Error(err, "Failed to list secrets", "labels", selector.MatchLabels)
				continue
			}
			for _, secret := range secretList.Items {
				// get the keys that look like env vars
				for key := range secret.Data {
					if match, _ := regexp.MatchString(`^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$`, key); match {
						allowedSecrets = append(allowedSecrets, key)
					}
				}
			}
		}
	}

	return allowedSecrets, nil
}
