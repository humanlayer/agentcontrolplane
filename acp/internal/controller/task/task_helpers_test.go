package task

import (
	"regexp"
	"testing"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	"github.com/humanlayer/agentcontrolplane/acp/internal/validation"
)

func TestBuildInitialContextWindow(t *testing.T) {
	tests := []struct {
		name           string
		contextWindow  []acp.Message
		systemPrompt   string
		userMessage    string
		expectedLen    int
		expectedFirst  string // content of first message
		expectedSecond string // content of second message (if exists)
	}{
		{
			name:           "empty context window creates system and user messages",
			contextWindow:  []acp.Message{},
			systemPrompt:   "You are a helpful assistant",
			userMessage:    "Hello world",
			expectedLen:    2,
			expectedFirst:  "You are a helpful assistant",
			expectedSecond: "Hello world",
		},
		{
			name: "context window with system message preserves it",
			contextWindow: []acp.Message{
				{Role: acp.MessageRoleSystem, Content: "Custom system"},
				{Role: acp.MessageRoleUser, Content: "User query"},
			},
			systemPrompt:   "You are a helpful assistant",
			userMessage:    "Hello world",
			expectedLen:    2,
			expectedFirst:  "Custom system",
			expectedSecond: "User query",
		},
		{
			name: "context window without system message gets one prepended",
			contextWindow: []acp.Message{
				{Role: acp.MessageRoleUser, Content: "User query"},
				{Role: acp.MessageRoleUser, Content: "Follow-up"},
			},
			systemPrompt:   "You are a helpful assistant",
			userMessage:    "Hello world",
			expectedLen:    3,
			expectedFirst:  "You are a helpful assistant",
			expectedSecond: "User query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildInitialContextWindow(tt.contextWindow, tt.systemPrompt, tt.userMessage)

			if len(result) != tt.expectedLen {
				t.Errorf("Expected %d messages, got %d", tt.expectedLen, len(result))
			}

			if len(result) > 0 && result[0].Content != tt.expectedFirst {
				t.Errorf("Expected first message content %q, got %q", tt.expectedFirst, result[0].Content)
			}

			if len(result) > 1 && result[1].Content != tt.expectedSecond {
				t.Errorf("Expected second message content %q, got %q", tt.expectedSecond, result[1].Content)
			}

			// First message should always be system
			if len(result) > 0 && result[0].Role != acp.MessageRoleSystem {
				t.Errorf("Expected first message to be system role, got %s", result[0].Role)
			}
		})
	}
}

func TestBuildToolTypeMap(t *testing.T) {
	tests := []struct {
		name     string
		tools    []llmclient.Tool
		expected map[string]acp.ToolType
	}{
		{
			name:     "empty tools returns empty map",
			tools:    []llmclient.Tool{},
			expected: map[string]acp.ToolType{},
		},
		{
			name: "single tool creates correct mapping",
			tools: []llmclient.Tool{
				{
					Function:    llmclient.ToolFunction{Name: "fetch"},
					ACPToolType: acp.ToolTypeMCP,
				},
			},
			expected: map[string]acp.ToolType{
				"fetch": acp.ToolTypeMCP,
			},
		},
		{
			name: "multiple tools create correct mappings",
			tools: []llmclient.Tool{
				{
					Function:    llmclient.ToolFunction{Name: "fetch"},
					ACPToolType: acp.ToolTypeMCP,
				},
				{
					Function:    llmclient.ToolFunction{Name: "human_contact"},
					ACPToolType: acp.ToolTypeHumanContact,
				},
				{
					Function:    llmclient.ToolFunction{Name: "delegate_to_agent__sub1"},
					ACPToolType: acp.ToolTypeDelegateToAgent,
				},
			},
			expected: map[string]acp.ToolType{
				"fetch":                   acp.ToolTypeMCP,
				"human_contact":           acp.ToolTypeHumanContact,
				"delegate_to_agent__sub1": acp.ToolTypeDelegateToAgent,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildToolTypeMap(tt.tools)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected map length %d, got %d", len(tt.expected), len(result))
			}

			for key, expectedType := range tt.expected {
				if actualType, exists := result[key]; !exists {
					t.Errorf("Expected key %q to exist in result map", key)
				} else if actualType != expectedType {
					t.Errorf("Expected %q to map to %v, got %v", key, expectedType, actualType)
				}
			}
		})
	}
}

func TestReconstructSpanContext(t *testing.T) {
	tests := []struct {
		name      string
		traceID   string
		spanID    string
		expectErr bool
	}{
		{
			name:      "valid trace and span IDs",
			traceID:   "0af7651916cd43dd8448eb211c80319c",
			spanID:    "b7ad6b7169203331",
			expectErr: false,
		},
		{
			name:      "invalid trace ID returns error",
			traceID:   "invalid-trace-id",
			spanID:    "b7ad6b7169203331",
			expectErr: true,
		},
		{
			name:      "invalid span ID returns error",
			traceID:   "0af7651916cd43dd8448eb211c80319c",
			spanID:    "invalid-span-id",
			expectErr: true,
		},
		{
			name:      "empty trace ID returns error",
			traceID:   "",
			spanID:    "b7ad6b7169203331",
			expectErr: true,
		},
		{
			name:      "empty span ID returns error",
			traceID:   "0af7651916cd43dd8448eb211c80319c",
			spanID:    "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reconstructSpanContext(tt.traceID, tt.spanID)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if !result.IsValid() {
				t.Errorf("Expected valid span context but got invalid")
			}

			// Verify the reconstructed context has correct properties
			if result.TraceID().String() != tt.traceID {
				t.Errorf("Expected trace ID %q, got %q", tt.traceID, result.TraceID().String())
			}

			if result.SpanID().String() != tt.spanID {
				t.Errorf("Expected span ID %q, got %q", tt.spanID, result.SpanID().String())
			}

			if !result.IsSampled() {
				t.Errorf("Expected sampled span context")
			}

			if !result.IsRemote() {
				t.Errorf("Expected remote span context")
			}
		})
	}
}

func TestGenerateK8sRandomString(t *testing.T) {
	tests := []struct {
		name        string
		length      int
		expectError bool
	}{
		{
			name:        "valid length 6",
			length:      6,
			expectError: false,
		},
		{
			name:        "valid length 8", 
			length:      8,
			expectError: false,
		},
		{
			name:        "invalid length 0 defaults to 6",
			length:      0,
			expectError: false,
		},
		{
			name:        "invalid length 10 defaults to 6",
			length:      10,
			expectError: false,
		},
	}

	// k8s naming convention: lowercase letters and numbers, starts with letter
	k8sPattern := regexp.MustCompile(`^[a-z][a-z0-9]*$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validation.GenerateK8sRandomString(tt.length)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if err != nil {
				return // Skip further checks if error expected
			}

			// Check length (should be input length or default 6)
			expectedLen := tt.length
			if tt.length < 1 || tt.length > 8 {
				expectedLen = 6
			}
			if len(result) != expectedLen {
				t.Errorf("Expected length %d, got %d (result: %s)", expectedLen, len(result), result)
			}

			// Check k8s naming convention
			if !k8sPattern.MatchString(result) {
				t.Errorf("Result %q does not match k8s naming convention (must start with letter, only lowercase letters and numbers)", result)
			}

			// First character must be a letter
			if len(result) > 0 && (result[0] < 'a' || result[0] > 'z') {
				t.Errorf("First character %c is not a lowercase letter", result[0])
			}

			// All characters must be lowercase letters or numbers
			for i, char := range result {
				if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
					t.Errorf("Character %c at position %d is not a lowercase letter or number", char, i)
				}
			}
		})
	}

	// Test uniqueness by generating multiple strings
	t.Run("generates unique strings", func(t *testing.T) {
		generated := make(map[string]bool)
		for i := 0; i < 100; i++ {
			result, err := validation.GenerateK8sRandomString(6)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if generated[result] {
				t.Errorf("Generated duplicate string: %s", result)
			}
			generated[result] = true
		}
	})
}
