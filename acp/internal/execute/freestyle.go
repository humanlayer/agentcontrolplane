package execute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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
