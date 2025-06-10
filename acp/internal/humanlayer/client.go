package humanlayer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	externalapi "github.com/humanlayer/agentcontrolplane/acp/internal/externalAPI"
)

type FunctionCallStatus struct {
	RequestedAt      *time.Time `json:"requested_at"`
	RespondedAt      *time.Time `json:"responded_at"`
	Approved         *bool      `json:"approved"`
	Comment          *string    `json:"comment"`
	RejectOptionName *string    `json:"reject_option_name"`
}

type FunctionCall struct {
	RunID  string              `json:"run_id"`
	CallID string              `json:"call_id"`
	Spec   map[string]any      `json:"spec"` // Assuming spec is a map
	Status *FunctionCallStatus `json:"status"`
}

type Approved struct {
	Approved bool    `json:"approved"`
	Comment  *string `json:"comment"`
}

type Rejected struct {
	Approved bool   `json:"approved"`
	Comment  string `json:"comment"`
}

func (fcs *FunctionCallStatus) AsCompleted() (interface{}, error) {
	if fcs.Approved == nil {
		return nil, fmt.Errorf("FunctionCallStatus.AsCompleted() called before approval")
	}

	if *fcs.Approved {
		return Approved{Approved: *fcs.Approved, Comment: fcs.Comment}, nil
	}

	if !*fcs.Approved && fcs.Comment == nil {
		return nil, fmt.Errorf("FunctionCallStatus.Rejected with no comment")
	}

	return Rejected{Approved: *fcs.Approved, Comment: *fcs.Comment}, nil
}

// Client implements the external API client for HumanLayer
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new HumanLayer API client
func NewClient(apiKey string) externalapi.Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.humanlayer.dev/humanlayer/v1/function_calls",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Call executes a function call to the HumanLayer API
func (c *Client) Call(
	ctx context.Context,
	runID,
	callID string,
	spec map[string]interface{},
) (json.RawMessage, error) {

	// Ensure kwargs is properly structured
	if fn, ok := spec["fn"].(string); ok && fn == "approve_tool_call" {
		if _, exists := spec["kwargs"]; !exists {
			// If kwargs doesn't exist, create it
			spec["kwargs"] = map[string]interface{}{}
		}

		// If kwargs is nil, recreate it
		if spec["kwargs"] == nil {
			spec["kwargs"] = map[string]interface{}{
				"tool_name": "unknown", // Default values
				"task_run":  runID,
				"namespace": "default",
			}
		}

		// Ensure kwargs is properly typed
		kwargs, ok := spec["kwargs"].(map[string]interface{})
		if !ok {
			// Convert or recreate kwargs
			spec["kwargs"] = map[string]interface{}{
				"tool_name": "unknown", // Default values
				"task_run":  runID,
				"namespace": "default",
			}
		} else if len(kwargs) == 0 {
			// If kwargs is empty, add default values
			kwargs["tool_name"] = "unknown"
			kwargs["task_run"] = runID
			kwargs["namespace"] = "default"
			spec["kwargs"] = kwargs
		}
	}

	// Now do the API call with the fixed spec

	// Prepare the request payload
	payload := map[string]interface{}{
		"run_id":  runID,
		"call_id": callID,
		"spec":    spec,
	}

	// Convert payload to JSON
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL,
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for non-success status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned non-success status: %d, body: %s",
			resp.StatusCode, string(respBody))
	}

	// Return raw JSON response
	return json.RawMessage(respBody), nil
}

// ClientFactory creates a HumanLayer client from secret data
func ClientFactory(secretData map[string][]byte, apiKeyField string) (externalapi.Client, error) {
	apiKeyBytes, exists := secretData[apiKeyField]
	if !exists {
		return nil, fmt.Errorf("API key not found in secret")
	}

	apiKey := string(apiKeyBytes)
	if apiKey == "" {
		return nil, fmt.Errorf("empty API key in secret")
	}

	return NewClient(apiKey), nil
}

// Initialize the client in an init function
func init() {
	externalapi.DefaultRegistry.Register("humanlayer-function-call", ClientFactory)
}

// RegisterClient adds the HumanLayer client to the external API registry
func RegisterClient() {
	externalapi.DefaultRegistry.Register("humanlayer-function-call", ClientFactory)
}
