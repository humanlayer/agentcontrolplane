package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var logLevel = "OFF"

func logDebug(format string, v ...interface{}) {
	if logLevel == "DEBUG" {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func logInfo(format string, v ...interface{}) {
	if logLevel == "DEBUG" || logLevel == "INFO" {
		log.Printf("[INFO] "+format, v...)
	}
}

// Client represents a HumanLayer API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewClient creates a new HumanLayer API client
func NewClient(apiKey string) *Client {
	return &Client{
		baseURL: "https://api.humanlayer.dev",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
	}
}

// FunctionCall represents a function call
type FunctionCall struct {
	CallID       string                 `json:"call_id"`
	Status       json.RawMessage        `json:"status"`
	FunctionName string                 `json:"function_name"`
	Arguments    map[string]interface{} `json:"arguments"`
}

// HumanContact represents a human contact request
type HumanContact struct {
	RunID  string                 `json:"run_id"`
	CallID string                 `json:"call_id"`
	Spec   map[string]interface{} `json:"spec"`
	Status *HumanContactStatus    `json:"status,omitempty"`
}

// HumanContactStatus represents the status of a human contact request
type HumanContactStatus struct {
	RequestedAt  json.RawMessage        `json:"requested_at,omitempty"`
	RespondedAt  json.RawMessage        `json:"responded_at,omitempty"`
	Response     *string                `json:"response,omitempty"`
	UserInfo     map[string]interface{} `json:"user_info,omitempty"`
	SlackContext map[string]interface{} `json:"slack_context,omitempty"`
}

// ListPendingFunctionCalls retrieves all pending function calls
func (c *Client) ListPendingFunctionCalls(ctx context.Context) ([]FunctionCall, error) {
	logInfo("Listing pending function calls: %s/humanlayer/v1/agent/function_calls/pending", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/humanlayer/v1/agent/function_calls/pending", c.baseURL), nil)
	if err != nil {
		log.Printf("[ERROR] Creating request: %v", err)
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] Executing request: %v", err)
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	logInfo("Received status code: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Unexpected status code: %d", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var calls []FunctionCall
	if err := json.NewDecoder(resp.Body).Decode(&calls); err != nil {
		log.Printf("[ERROR] Decoding response: %v", err)
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	logInfo("Decoded %d function calls", len(calls))
	for i, call := range calls {
		logDebug("Call %d: call_id=%s, status=%s", i, call.CallID, string(call.Status))
	}
	return calls, nil
}

// ListPendingHumanContacts retrieves all pending human contact requests
func (c *Client) ListPendingHumanContacts(ctx context.Context) ([]HumanContact, error) {
	logInfo("Listing pending human contacts: %s/humanlayer/v1/agent/human_contacts/pending", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/humanlayer/v1/agent/human_contacts/pending", c.baseURL), nil)
	if err != nil {
		log.Printf("[ERROR] Creating request: %v", err)
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] Executing request: %v", err)
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	logInfo("Received status code: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Unexpected status code: %d", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var contacts []HumanContact
	if err := json.NewDecoder(resp.Body).Decode(&contacts); err != nil {
		log.Printf("[ERROR] Decoding response: %v", err)
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	logInfo("Decoded %d human contacts", len(contacts))
	for i, contact := range contacts {
		logDebug("Contact %d: call_id=%s, run_id=%s", i, contact.CallID, contact.RunID)
	}
	return contacts, nil
}

// RespondToFunctionCall responds to a function call
func (c *Client) RespondToFunctionCall(ctx context.Context, callID string, approve bool, comment string) error {
	logInfo("Responding to function call: %s, approve=%v, comment=%q", callID, approve, comment)
	body := map[string]interface{}{
		"approved": approve,
		"comment":  comment,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Printf("[ERROR] Marshaling request: %v", err)
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/humanlayer/v1/agent/function_calls/%s/respond", c.baseURL, callID), bytes.NewReader(jsonBody))
	if err != nil {
		log.Printf("[ERROR] Creating request: %v", err)
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] Executing request: %v", err)
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	logInfo("Received status code: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Unexpected status code: %d", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	logInfo("Responded to function call successfully")
	return nil
}

// RespondToHumanContact responds to a human contact request
func (c *Client) RespondToHumanContact(ctx context.Context, callID string, response string) error {
	logInfo("Responding to human contact: %s, response=%q", callID, response)
	body := map[string]interface{}{
		"response": response,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Printf("[ERROR] Marshaling request: %v", err)
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/humanlayer/v1/agent/human_contacts/%s/respond", c.baseURL, callID), bytes.NewReader(jsonBody))
	if err != nil {
		log.Printf("[ERROR] Creating request: %v", err)
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] Executing request: %v", err)
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	logInfo("Received status code: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Unexpected status code: %d", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	logInfo("Responded to human contact successfully")
	return nil
}

func main() {
	operation := flag.String("o", "", "Operation to perform (list-pending-function-calls, list-pending-human-contacts, respond-function-call, respond-human-contact, get-human-contact)")
	callID := flag.String("call-id", "", "Call ID for respond/get operation")
	approve := flag.Bool("approve", false, "Whether to approve the function call")
	comment := flag.String("comment", "", "Comment for the response")
	response := flag.String("response", "", "Response for the human contact")
	n := flag.Int("n", 5, "Number of calls to print (default 5)")
	logLevelFlag := flag.String("log-level", "OFF", "Log level: OFF, INFO, or DEBUG (default OFF)")
	flag.Parse()

	logLevel = *logLevelFlag

	apiKey := os.Getenv("HUMANLAYER_API_KEY")
	if apiKey == "" {
		log.Println("[ERROR] HUMANLAYER_API_KEY environment variable is required")
		os.Exit(1)
	}

	client := NewClient(apiKey)
	ctx := context.Background()

	logInfo("Operation: %s", *operation)

	switch *operation {
	case "list-pending-function-calls":
		calls, err := client.ListPendingFunctionCalls(ctx)
		if err != nil {
			log.Printf("[ERROR] Listing function calls: %v", err)
			os.Exit(1)
		}
		logInfo("Outputting %d function calls as JSON", len(calls))
		if *n > 0 && len(calls) > *n {
			calls = calls[len(calls)-*n:]
		}
		json.NewEncoder(os.Stdout).Encode(calls)

	case "list-pending-human-contacts":
		contacts, err := client.ListPendingHumanContacts(ctx)
		if err != nil {
			log.Printf("[ERROR] Listing human contacts: %v", err)
			os.Exit(1)
		}
		logInfo("Outputting %d human contacts as JSON", len(contacts))
		if *n > 0 && len(contacts) > *n {
			contacts = contacts[len(contacts)-*n:]
		}
		json.NewEncoder(os.Stdout).Encode(contacts)

	case "respond-function-call":
		if *callID == "" {
			log.Println("[ERROR] call-id is required for respond operation")
			os.Exit(1)
		}
		if err := client.RespondToFunctionCall(ctx, *callID, *approve, *comment); err != nil {
			log.Printf("[ERROR] Responding to function call: %v", err)
			os.Exit(1)
		}
		logInfo("Response submitted successfully")
		fmt.Println("Response submitted successfully")

	case "respond-human-contact":
		if *callID == "" {
			log.Println("[ERROR] call-id is required for respond-human-contact operation")
			os.Exit(1)
		}
		if *response == "" {
			log.Println("[ERROR] response is required for respond-human-contact operation")
			os.Exit(1)
		}
		if err := client.RespondToHumanContact(ctx, *callID, *response); err != nil {
			log.Printf("[ERROR] Responding to human contact: %v", err)
			os.Exit(1)
		}
		logInfo("Human contact response submitted successfully")
		fmt.Println("Human contact response submitted successfully")

	default:
		log.Printf("[ERROR] Unknown operation: %s", *operation)
		os.Exit(1)
	}
}
