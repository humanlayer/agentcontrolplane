package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = Describe("API Server", func() {
	var (
		ctx       context.Context
		k8sClient client.Client
		server    *APIServer
		router    *gin.Engine
		recorder  *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		ctx = context.Background()
		// Create a scheme with our API types registered
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(acp.AddToScheme(scheme)).To(Succeed())

		// Create a fake client
		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		// Create the API server with the client
		server = NewAPIServer(k8sClient, ":8080")
		router = server.router
		recorder = httptest.NewRecorder()
	})

	Describe("ensureNamespaceExists", func() {
		It("should do nothing if namespace already exists", func() {
			// Create a namespace
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-namespace",
				},
			}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

			// Call ensureNamespaceExists
			err := server.ensureNamespaceExists(ctx, "existing-namespace")
			Expect(err).NotTo(HaveOccurred())

			// Verify no new namespace was created (count should still be 1)
			var namespaceList corev1.NamespaceList
			Expect(k8sClient.List(ctx, &namespaceList)).To(Succeed())
			count := 0
			for _, ns := range namespaceList.Items {
				if ns.Name == "existing-namespace" {
					count++
				}
			}
			Expect(count).To(Equal(1))
		})

		It("should create a namespace if it doesn't exist", func() {
			// Call ensureNamespaceExists for a new namespace
			err := server.ensureNamespaceExists(ctx, "new-namespace")
			Expect(err).NotTo(HaveOccurred())

			// Verify namespace was created
			var namespace corev1.Namespace
			err = k8sClient.Get(ctx, client.ObjectKey{Name: "new-namespace"}, &namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(namespace.Name).To(Equal("new-namespace"))
		})

		It("should handle Kubernetes API errors", func() {
			// Create a client that returns an error for Get operations
			errorClient := &errorK8sClient{Client: k8sClient}
			errorClient.returnErrorOnGetNamespace = true
			errorServer := NewAPIServer(errorClient, ":8080")

			// Call ensureNamespaceExists
			err := errorServer.ensureNamespaceExists(ctx, "error-namespace")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to check namespace existence"))
		})
	})

	Describe("POST /v1/tasks", func() {
		It("should create a new task with valid input", func() {
			// Create an agent first
			agent := &acp.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: acp.AgentSpec{},
			}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())

			// Create the request body
			reqBody := CreateTaskRequest{
				AgentName:   "test-agent",
				UserMessage: "Hello, agent!",
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			// Create a test request
			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Serve the request
			router.ServeHTTP(recorder, req)

			// Verify the response
			Expect(recorder.Code).To(Equal(http.StatusCreated))

			// Parse the response
			var responseTask acp.Task
			err = json.Unmarshal(recorder.Body.Bytes(), &responseTask)
			Expect(err).NotTo(HaveOccurred())

			// Verify the task was created with expected values
			Expect(responseTask.Spec.AgentRef.Name).To(Equal("test-agent"))
			Expect(responseTask.Spec.UserMessage).To(Equal("Hello, agent!"))
			Expect(responseTask.Namespace).To(Equal("default"))
			Expect(responseTask.Labels["acp.humanlayer.dev/agent"]).To(Equal("test-agent"))

			// Verify task is in the Kubernetes store
			var storedTask acp.Task
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      responseTask.Name,
				Namespace: "default",
			}, &storedTask)).To(Succeed())

			// Verify task name format (should follow the pattern {agentName}-task-{uuid[:8]})
			Expect(responseTask.Name).To(HavePrefix("test-agent-task-"))
		})

		It("should validate required fields", func() {
			// Missing agent name
			reqBody := CreateTaskRequest{
				UserMessage: "Hello, agent!",
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			var errorResponse map[string]string
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("agentName is required"))

			// Missing user message
			reqBody = CreateTaskRequest{
				AgentName: "test-agent",
			}
			jsonBody, err = json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req = httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("one of userMessage or contextWindow must be provided"))
		})

		It("should create a task with only contextWindow", func() {
			agent := &acp.Agent{ObjectMeta: metav1.ObjectMeta{Name: "test-agent", Namespace: "default"}, Spec: acp.AgentSpec{}}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())
			reqBody := CreateTaskRequest{
				AgentName:     "test-agent",
				ContextWindow: []acp.Message{{Role: "system", Content: "System"}, {Role: "user", Content: "User query"}},
			}
			jsonBody, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusCreated))
			var task acp.Task
			err := json.Unmarshal(recorder.Body.Bytes(), &task)
			Expect(err).NotTo(HaveOccurred())
			Expect(task.Spec.UserMessage).To(BeEmpty())
			Expect(task.Spec.ContextWindow).To(HaveLen(2))
		})

		It("should fail if both are provided", func() {
			reqBody := CreateTaskRequest{
				AgentName:     "test-agent",
				UserMessage:   "Test",
				ContextWindow: []acp.Message{{Role: "user", Content: "Query"}},
			}
			jsonBody, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			var errorResponse map[string]string
			err := json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("only one of userMessage or contextWindow can be provided"))
		})

		It("should fail if contextWindow has invalid roles", func() {
			reqBody := CreateTaskRequest{
				AgentName:     "test-agent",
				ContextWindow: []acp.Message{{Role: "invalid", Content: "Invalid"}},
			}
			jsonBody, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			var errorResponse map[string]string
			err := json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("invalid role in contextWindow: invalid"))
		})

		It("should return 404 if the agent does not exist", func() {
			reqBody := CreateTaskRequest{
				AgentName:   "non-existent-agent",
				UserMessage: "Hello, agent!",
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusNotFound))
			var errorResponse map[string]string
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("Agent not found"))
		})

		It("should use a custom namespace when provided", func() {
			// Create an agent in the custom namespace
			customNamespace := "custom-namespace"

			// Create the namespace first
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: customNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

			agent := &acp.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "custom-agent",
					Namespace: customNamespace,
				},
				Spec: acp.AgentSpec{},
			}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())

			// Create the request with the custom namespace
			reqBody := CreateTaskRequest{
				Namespace:   customNamespace,
				AgentName:   "custom-agent",
				UserMessage: "Hello from custom namespace!",
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusCreated))

			var responseTask acp.Task
			err = json.Unmarshal(recorder.Body.Bytes(), &responseTask)
			Expect(err).NotTo(HaveOccurred())

			// Verify the task was created in the custom namespace
			Expect(responseTask.Namespace).To(Equal(customNamespace))

			// Verify task is in the Kubernetes store in the custom namespace
			var storedTask acp.Task
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      responseTask.Name,
				Namespace: customNamespace,
			}, &storedTask)).To(Succeed())

			// Verify task name format (should follow the pattern {agentName}-task-{uuid[:8]})
			Expect(responseTask.Name).To(HavePrefix("custom-agent-task-"))
		})

		It("should reject invalid JSON", func() {
			// Invalid JSON format
			reqBody := `{"agentName": "test-agent", "userMessage": "hello"`
			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			var errorResponse map[string]string
			err := json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(ContainSubstring("Invalid request body"))
		})

		It("should handle Kubernetes API errors appropriately", func() {
			// Create a client that returns an error for Create
			errorClient := &errorK8sClient{Client: k8sClient}
			errorServer := NewAPIServer(errorClient, ":8080")
			errorRouter := errorServer.router

			// Create an agent
			agent := &acp.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent-error",
					Namespace: "default",
				},
			}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())

			reqBody := CreateTaskRequest{
				AgentName:   "test-agent-error",
				UserMessage: "This should trigger an error",
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			errorRouter.ServeHTTP(recorder, req)

			// Expect Internal Server Error
			Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})

// Custom client that forces an error on Create
type errorK8sClient struct {
	client.Client
	returnErrorOnGetNamespace bool
}

func (e *errorK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	// Return an error for Task objects
	if _, ok := obj.(*acp.Task); ok {
		return &errors.StatusError{
			ErrStatus: metav1.Status{
				Status:  "Failure",
				Message: "Simulated internal server error",
				Reason:  "InternalError",
				Code:    http.StatusInternalServerError,
			},
		}
	}
	// Return an error for Namespace objects if specified
	if _, ok := obj.(*corev1.Namespace); ok && e.returnErrorOnGetNamespace {
		return &errors.StatusError{
			ErrStatus: metav1.Status{
				Status:  "Failure",
				Message: "Simulated internal server error",
				Reason:  "InternalError",
				Code:    http.StatusInternalServerError,
			},
		}
	}
	return e.Client.Create(ctx, obj, opts...)
}

func (e *errorK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if e.returnErrorOnGetNamespace && key.Name != "" {
		if _, ok := obj.(*corev1.Namespace); ok {
			return &errors.StatusError{
				ErrStatus: metav1.Status{
					Status:  "Failure",
					Message: "Simulated get namespace error",
					Reason:  "InternalError",
					Code:    http.StatusInternalServerError,
				},
			}
		}
	}
	return e.Client.Get(ctx, key, obj, opts...)
}

// Tests for Agent API endpoints
var _ = Describe("Agent API", func() {
	var (
		ctx       context.Context
		k8sClient client.Client
		server    *APIServer
		router    *gin.Engine
		recorder  *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		ctx = context.Background()
		// Create a scheme with our API types registered
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(acp.AddToScheme(scheme)).To(Succeed())

		// Create a fake client
		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		// Create the API server with the client
		server = NewAPIServer(k8sClient, ":8080")
		router = server.router
		recorder = httptest.NewRecorder()
	})

	// Helper function to create an LLM for tests
	createTestLLM := func(name, namespace string) *acp.LLM {
		llm := &acp.LLM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: acp.LLMSpec{
				Provider: "test-provider",
				Parameters: acp.BaseConfig{
					Model: "test-model",
				},
			},
		}
		Expect(k8sClient.Create(ctx, llm)).To(Succeed())
		return llm
	}

	Describe("POST /v1/agents", func() {
		It("should create a new agent with valid input", func() {
			// Create an LLM first
			createTestLLM("test-llm", "default")

			// Create the request body
			reqBody := CreateAgentRequest{
				Name: "test-agent",
				LLM: LLMDefinition{
					Name:     "test-llm",
					Provider: "openai",
					Model:    "gpt-4",
					APIKey:   "sk-test-key",
				},
				SystemPrompt: "You are a test agent",
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			// Create a test request
			req := httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Serve the request
			router.ServeHTTP(recorder, req)

			// Verify the response
			Expect(recorder.Code).To(Equal(http.StatusCreated))

			// Parse the response
			var response AgentResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			// Verify the agent was created with expected values
			Expect(response.Name).To(Equal("test-agent"))
			Expect(response.LLM).To(Equal("test-llm"))
			Expect(response.SystemPrompt).To(Equal("You are a test agent"))
			Expect(response.Namespace).To(Equal("default"))

			// Verify agent is in the Kubernetes store
			var storedAgent acp.Agent
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-agent",
				Namespace: "default",
			}, &storedAgent)).To(Succeed())
		})

		It("should create an agent with MCP servers", func() {
			// Create an LLM first
			createTestLLM("test-llm-2", "default")

			// Create the request body with MCP servers
			reqBody := CreateAgentRequest{
				Name: "test-agent-mcp",
				LLM: LLMDefinition{
					Name:     "test-llm-2",
					Provider: "anthropic",
					Model:    "claude-3",
					APIKey:   "sk-test-key-2",
				},
				SystemPrompt: "You are a test agent with MCP servers",
				MCPServers: map[string]MCPServerConfig{
					"stdio": {
						Transport: "stdio",
						Command:   "python",
						Args:      []string{"-m", "test_script.py"},
						Env:       map[string]string{"TEST_ENV": "value"},
						Secrets:   map[string]string{"API_KEY": "test-key"},
					},
					"http": {
						Transport: "http",
						URL:       "http://localhost:8000",
						Env:       map[string]string{"SERVER_URL": "value"},
					},
				},
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			// Create a test request
			req := httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Serve the request
			router.ServeHTTP(recorder, req)

			// Verify the response
			Expect(recorder.Code).To(Equal(http.StatusCreated))

			// Parse the response
			var response AgentResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			// Verify the agent was created with expected values
			Expect(response.Name).To(Equal("test-agent-mcp"))
			Expect(response.MCPServers).To(HaveLen(2))
			Expect(response.MCPServers).To(HaveKey("stdio"))
			Expect(response.MCPServers).To(HaveKey("http"))

			// Verify MCP servers are in the Kubernetes store
			var stdioMCP acp.MCPServer
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-agent-mcp-stdio",
				Namespace: "default",
			}, &stdioMCP)).To(Succeed())

			var httpMCP acp.MCPServer
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-agent-mcp-http",
				Namespace: "default",
			}, &httpMCP)).To(Succeed())

			// Verify secret was created
			var secret corev1.Secret
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-agent-mcp-stdio-secrets",
				Namespace: "default",
			}, &secret)).To(Succeed())
			Expect(string(secret.Data["API_KEY"])).To(Equal("test-key"))
		})

		It("should validate required fields", func() {
			// Missing name
			reqBody := CreateAgentRequest{
				LLM: LLMDefinition{
					Provider: "openai",
					Model:    "gpt-4",
					APIKey:   "sk-test-key",
				},
				SystemPrompt: "Test prompt",
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			var errorResponse map[string]string
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("llm fields (name, provider, model, apiKey) are required"))

			// Missing LLM
			reqBody = CreateAgentRequest{
				Name: "test-agent",
				LLM: LLMDefinition{
					Name:     "", // Missing name
					Provider: "openai",
					Model:    "gpt-4",
					APIKey:   "sk-test-key",
				},
				SystemPrompt: "Test prompt",
			}
			jsonBody, err = json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req = httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("llm fields (name, provider, model, apiKey) are required"))
		})

		It("should return 404 if the LLM does not exist", func() {
			reqBody := CreateAgentRequest{
				Name: "test-agent",
				LLM: LLMDefinition{
					Name:     "non-existent-llm",
					Provider: "openai",
					Model:    "gpt-4",
					APIKey:   "sk-test-key",
				},
				SystemPrompt: "Test prompt",
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusNotFound))
			var errorResponse map[string]string
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("LLM not found"))
		})

		It("should validate MCP server configurations", func() {
			// Create an LLM first
			createTestLLM("test-llm-4", "default")

			// Test with invalid transport type
			reqBody := CreateAgentRequest{
				Name: "test-agent-invalid-mcp",
				LLM: LLMDefinition{
					Name:     "test-llm-4",
					Provider: "openai", 
					Model:    "gpt-4",
					APIKey:   "sk-test-key-4",
				},
				SystemPrompt: "Test agent",
				MCPServers: map[string]MCPServerConfig{
					"invalid": {
						Transport: "invalid-transport", // Not "stdio" or "http"
						Command:   "python",
						Args:      []string{"-m", "script.py"},
					},
				},
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			var errorResponse map[string]string
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(ContainSubstring("invalid transport"))

			// Test without transport (should default to stdio)
			reqBody = CreateAgentRequest{
				Name: "test-agent-default-transport",
				LLM: LLMDefinition{
					Name:     "test-llm-4b",
					Provider: "openai",
					Model:    "gpt-4",
					APIKey:   "sk-test-key-4b",
				},
				SystemPrompt: "Test agent",
				MCPServers: map[string]MCPServerConfig{
					"default": {
						// No transport specified (should default to stdio)
						Command: "python",
						Args:    []string{"-m", "script.py"},
					},
				},
			}
			jsonBody, err = json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req = httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			// Should succeed with default stdio transport
			Expect(recorder.Code).To(Equal(http.StatusCreated))

			// Verify MCP server is in Kubernetes with stdio transport
			var mcpServer acp.MCPServer
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-agent-default-transport-default",
				Namespace: "default",
			}, &mcpServer)).To(Succeed())
			Expect(mcpServer.Spec.Transport).To(Equal("stdio"))

			// Test missing command for stdio transport
			reqBody = CreateAgentRequest{
				Name: "test-agent-missing-command",
				LLM: LLMDefinition{
					Name:     "test-llm-4c",
					Provider: "openai",
					Model:    "gpt-4",
					APIKey:   "sk-test-key-4c",
				},
				SystemPrompt: "Test agent",
				MCPServers: map[string]MCPServerConfig{
					"stdio": {
						Transport: "stdio",
						// Missing command and args
					},
				},
			}
			jsonBody, err = json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req = httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(ContainSubstring("command and args required"))

			// Test missing URL for http transport
			reqBody = CreateAgentRequest{
				Name: "test-agent-missing-url",
				LLM: LLMDefinition{
					Name:     "test-llm-4d",
					Provider: "openai",
					Model:    "gpt-4",
					APIKey:   "sk-test-key-4d",
				},
				SystemPrompt: "Test agent",
				MCPServers: map[string]MCPServerConfig{
					"http": {
						Transport: "http",
						// Missing URL
					},
				},
			}
			jsonBody, err = json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req = httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(ContainSubstring("url required"))
		})

		It("should return 409 if the agent already exists", func() {
			// Create an LLM
			createTestLLM("test-llm-3", "default")

			// Create an agent
			agent := &acp.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-agent",
					Namespace: "default",
				},
				Spec: acp.AgentSpec{
					LLMRef: acp.LocalObjectReference{Name: "test-llm-3"},
					System: "Existing agent",
				},
			}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())

			// Try to create the same agent again
			reqBody := CreateAgentRequest{
				Name: "existing-agent",
				LLM: LLMDefinition{
					Name:     "test-llm-3",
					Provider: "openai",
					Model:    "gpt-4",
					APIKey:   "sk-test-key-3",
				},
				SystemPrompt: "Test prompt",
			}
			jsonBody, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusConflict))
			var errorResponse map[string]string
			err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("Agent already exists"))
		})
	})

	Describe("GET /v1/agents", func() {
		It("should return a list of agents", func() {
			// Create namespace
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"},
			}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

			// Create an LLM
			createTestLLM("test-llm-5", "test-namespace")

			// Create a few agents
			agent1 := &acp.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent-1",
					Namespace: "test-namespace",
				},
				Spec: acp.AgentSpec{
					LLMRef: acp.LocalObjectReference{Name: "test-llm-5"},
					System: "Agent 1",
				},
			}
			Expect(k8sClient.Create(ctx, agent1)).To(Succeed())

			agent2 := &acp.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent-2",
					Namespace: "test-namespace",
				},
				Spec: acp.AgentSpec{
					LLMRef: acp.LocalObjectReference{Name: "test-llm-5"},
					System: "Agent 2",
				},
			}
			Expect(k8sClient.Create(ctx, agent2)).To(Succeed())

			// Make the request
			req := httptest.NewRequest(http.MethodGet, "/v1/agents?namespace=test-namespace", nil)
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			// Verify the response
			Expect(recorder.Code).To(Equal(http.StatusOK))

			// Parse the response
			var response []AgentResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			// Verify the response contains the agents
			Expect(response).To(HaveLen(2))
			agentNames := []string{response[0].Name, response[1].Name}
			Expect(agentNames).To(ContainElements("test-agent-1", "test-agent-2"))
		})

		It("should require a namespace parameter", func() {
			req := httptest.NewRequest(http.MethodGet, "/v1/agents", nil)
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			var errorResponse map[string]string
			err := json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("namespace query parameter is required"))
		})
	})

	Describe("GET /v1/agents/:name", func() {
		It("should return a specific agent", func() {
			// Create namespace
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "get-namespace"},
			}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

			// Create an LLM
			createTestLLM("get-llm", "get-namespace")

			// Create an agent
			agent := &acp.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "get-agent",
					Namespace: "get-namespace",
				},
				Spec: acp.AgentSpec{
					LLMRef: acp.LocalObjectReference{Name: "get-llm"},
					System: "Get Agent",
				},
			}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())

			// Make the request
			req := httptest.NewRequest(http.MethodGet, "/v1/agents/get-agent?namespace=get-namespace", nil)
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			// Verify the response
			Expect(recorder.Code).To(Equal(http.StatusOK))

			// Parse the response
			var response AgentResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			// Verify the agent details
			Expect(response.Name).To(Equal("get-agent"))
			Expect(response.Namespace).To(Equal("get-namespace"))
			Expect(response.LLM).To(Equal("get-llm"))
			Expect(response.SystemPrompt).To(Equal("Get Agent"))
		})

		It("should return 404 for non-existent agent", func() {
			req := httptest.NewRequest(http.MethodGet, "/v1/agents/non-existent-agent?namespace=default", nil)
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusNotFound))
			var errorResponse map[string]string
			err := json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("Agent not found"))
		})

		It("should require a namespace parameter", func() {
			req := httptest.NewRequest(http.MethodGet, "/v1/agents/some-agent", nil)
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			var errorResponse map[string]string
			err := json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorResponse["error"]).To(Equal("namespace query parameter is required"))
		})

		Describe("PUT /v1/agents/:name", func() {
			It("should update an existing agent", func() {
				// Create an LLM
				createTestLLM("test-llm-update", "default")

				// Create an agent first
				agent := &acp.Agent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "update-agent",
						Namespace: "default",
					},
					Spec: acp.AgentSpec{
						LLMRef: acp.LocalObjectReference{Name: "test-llm-update"},
						System: "Old prompt",
					},
				}
				Expect(k8sClient.Create(ctx, agent)).To(Succeed())

				// Prepare the update request
				reqBody := UpdateAgentRequest{
					LLM:          "test-llm-update",
					SystemPrompt: "New prompt",
				}
				jsonBody, err := json.Marshal(reqBody)
				Expect(err).NotTo(HaveOccurred())

				// Create a test request
				req := httptest.NewRequest(http.MethodPut, "/v1/agents/update-agent?namespace=default", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				recorder = httptest.NewRecorder()

				// Serve the request
				router.ServeHTTP(recorder, req)

				// Verify the response
				Expect(recorder.Code).To(Equal(http.StatusOK))

				// Parse the response
				var response AgentResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &response)
				Expect(err).NotTo(HaveOccurred())

				// Verify response has updated values
				Expect(response.Name).To(Equal("update-agent"))
				Expect(response.SystemPrompt).To(Equal("New prompt"))

				// Verify agent was updated in Kubernetes
				var updatedAgent acp.Agent
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      "update-agent",
					Namespace: "default",
				}, &updatedAgent)).To(Succeed())

				Expect(updatedAgent.Spec.System).To(Equal("New prompt"))
			})

			It("should update MCP servers (add, update, remove)", func() {
				// Create an LLM
				createTestLLM("test-llm-mcp-update", "default")

				// Create an MCP server that will be removed during update
				oldMCP := &acp.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mcp-update-agent-old",
						Namespace: "default",
					},
					Spec: acp.MCPServerSpec{
						Transport: "stdio",
						Command:   "oldcmd",
						Args:      []string{"oldarg"},
					},
				}
				Expect(k8sClient.Create(ctx, oldMCP)).To(Succeed())

				// Create an agent with an initial MCP server
				agent := &acp.Agent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mcp-update-agent",
						Namespace: "default",
					},
					Spec: acp.AgentSpec{
						LLMRef: acp.LocalObjectReference{Name: "test-llm-mcp-update"},
						System: "Original prompt",
						MCPServers: []acp.LocalObjectReference{
							{Name: "mcp-update-agent-old"},
						},
					},
				}
				Expect(k8sClient.Create(ctx, agent)).To(Succeed())

				// Prepare update request with new MCP server configuration
				reqBody := UpdateAgentRequest{
					LLM:          "test-llm-mcp-update",
					SystemPrompt: "Updated prompt",
					MCPServers: map[string]MCPServerConfig{
						"new": {
							Transport: "http",
							URL:       "http://newserver.com",
						},
						"existing": {
							Transport: "stdio",
							Command:   "updated-cmd",
							Args:      []string{"arg1", "arg2"},
							Secrets:   map[string]string{"API_KEY": "secret-value"},
						},
					},
				}
				jsonBody, err := json.Marshal(reqBody)
				Expect(err).NotTo(HaveOccurred())

				// Create a test request
				req := httptest.NewRequest(http.MethodPut, "/v1/agents/mcp-update-agent?namespace=default", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				recorder = httptest.NewRecorder()

				// Serve the request
				router.ServeHTTP(recorder, req)

				// Verify the response
				Expect(recorder.Code).To(Equal(http.StatusOK))

				// Parse the response
				var response AgentResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &response)
				Expect(err).NotTo(HaveOccurred())

				// Verify response has updated values
				Expect(response.SystemPrompt).To(Equal("Updated prompt"))
				Expect(response.MCPServers).To(HaveKey("new"))
				Expect(response.MCPServers).To(HaveKey("existing"))
				Expect(response.MCPServers).NotTo(HaveKey("old"))

				// Verify agent was updated in Kubernetes with new MCP servers
				var updatedAgent acp.Agent
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      "mcp-update-agent",
					Namespace: "default",
				}, &updatedAgent)).To(Succeed())

				Expect(updatedAgent.Spec.System).To(Equal("Updated prompt"))
				Expect(len(updatedAgent.Spec.MCPServers)).To(Equal(2))

				// Extract MCP server names from the updated agent
				mcpServerNames := []string{}
				for _, mcpRef := range updatedAgent.Spec.MCPServers {
					mcpServerNames = append(mcpServerNames, mcpRef.Name)
				}

				// Verify both new MCP servers exist in the agent's references
				Expect(mcpServerNames).To(ContainElement("mcp-update-agent-new"))
				Expect(mcpServerNames).To(ContainElement("mcp-update-agent-existing"))

				// Old MCP server should no longer be in the agent's references
				Expect(mcpServerNames).NotTo(ContainElement("mcp-update-agent-old"))

				// Verify new MCP server was created
				var newMCP acp.MCPServer
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      "mcp-update-agent-new",
					Namespace: "default",
				}, &newMCP)).To(Succeed())
				Expect(newMCP.Spec.Transport).To(Equal("http"))
				Expect(newMCP.Spec.URL).To(Equal("http://newserver.com"))

				// Verify MCP server with secrets
				var existingMCP acp.MCPServer
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      "mcp-update-agent-existing",
					Namespace: "default",
				}, &existingMCP)).To(Succeed())
				Expect(existingMCP.Spec.Transport).To(Equal("stdio"))
				Expect(existingMCP.Spec.Command).To(Equal("updated-cmd"))
				Expect(existingMCP.Spec.Args).To(ConsistOf("arg1", "arg2"))

				// Verify the secret was created
				var secret corev1.Secret
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      "mcp-update-agent-existing-secrets",
					Namespace: "default",
				}, &secret)).To(Succeed())
				Expect(string(secret.Data["API_KEY"])).To(Equal("secret-value"))

				// Verify old MCP server was deleted
				var oldMCPCheck acp.MCPServer
				err = k8sClient.Get(ctx, types.NamespacedName{
					Name:      "mcp-update-agent-old",
					Namespace: "default",
				}, &oldMCPCheck)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})

			It("should return 404 if agent doesn't exist", func() {
				// Prepare update request
				reqBody := UpdateAgentRequest{
					LLM:          "non-existent-llm",
					SystemPrompt: "Test prompt",
				}
				jsonBody, err := json.Marshal(reqBody)
				Expect(err).NotTo(HaveOccurred())

				// Create a test request for non-existent agent
				req := httptest.NewRequest(http.MethodPut, "/v1/agents/non-existent-agent?namespace=default", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				recorder = httptest.NewRecorder()

				// Serve the request
				router.ServeHTTP(recorder, req)

				// Verify we get 404 Not Found
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
				var errorResponse map[string]string
				err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
				Expect(err).NotTo(HaveOccurred())
				Expect(errorResponse["error"]).To(Equal("Agent not found"))
			})

			It("should return 404 if LLM doesn't exist", func() {
				// Create an agent first
				agent := &acp.Agent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-agent",
						Namespace: "default",
					},
					Spec: acp.AgentSpec{
						LLMRef: acp.LocalObjectReference{Name: "existing-llm"},
						System: "Existing prompt",
					},
				}
				Expect(k8sClient.Create(ctx, agent)).To(Succeed())

				// Prepare update request with non-existent LLM
				reqBody := UpdateAgentRequest{
					LLM:          "non-existent-llm",
					SystemPrompt: "Updated prompt",
				}
				jsonBody, err := json.Marshal(reqBody)
				Expect(err).NotTo(HaveOccurred())

				// Create a test request
				req := httptest.NewRequest(http.MethodPut, "/v1/agents/existing-agent?namespace=default", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				recorder = httptest.NewRecorder()

				// Serve the request
				router.ServeHTTP(recorder, req)

				// Verify we get 404 Not Found for LLM
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
				var errorResponse map[string]string
				err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
				Expect(err).NotTo(HaveOccurred())
				Expect(errorResponse["error"]).To(Equal("LLM not found"))
			})

			It("should return 400 for invalid MCP server configuration", func() {
				// Create an LLM
				createTestLLM("test-llm-invalid", "default")

				// Create an agent
				agent := &acp.Agent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "agent-invalid-mcp",
						Namespace: "default",
					},
					Spec: acp.AgentSpec{
						LLMRef: acp.LocalObjectReference{Name: "test-llm-invalid"},
						System: "Original prompt",
					},
				}
				Expect(k8sClient.Create(ctx, agent)).To(Succeed())

				// Prepare update request with invalid MCP server config (missing URL for http transport)
				reqBody := UpdateAgentRequest{
					LLM:          "test-llm-invalid",
					SystemPrompt: "Updated prompt",
					MCPServers: map[string]MCPServerConfig{
						"invalid": {
							Transport: "http",
							// URL is required for http transport but missing
						},
					},
				}
				jsonBody, err := json.Marshal(reqBody)
				Expect(err).NotTo(HaveOccurred())

				// Create a test request
				req := httptest.NewRequest(http.MethodPut, "/v1/agents/agent-invalid-mcp?namespace=default", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				recorder = httptest.NewRecorder()

				// Serve the request
				router.ServeHTTP(recorder, req)

				// Verify we get 400 Bad Request
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
				var errorResponse map[string]string
				err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
				Expect(err).NotTo(HaveOccurred())
				Expect(errorResponse["error"]).To(ContainSubstring("Invalid MCP server config"))
				Expect(errorResponse["error"]).To(ContainSubstring("url required"))
			})

			It("should require a namespace parameter", func() {
				// Prepare update request
				reqBody := UpdateAgentRequest{
					LLM:          "test-llm",
					SystemPrompt: "Test prompt",
				}
				jsonBody, err := json.Marshal(reqBody)
				Expect(err).NotTo(HaveOccurred())

				// Create a test request without namespace
				req := httptest.NewRequest(http.MethodPut, "/v1/agents/some-agent", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				recorder = httptest.NewRecorder()

				// Serve the request
				router.ServeHTTP(recorder, req)

				// Verify we get 400 Bad Request
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
				var errorResponse map[string]string
				err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
				Expect(err).NotTo(HaveOccurred())
				Expect(errorResponse["error"]).To(Equal("namespace query parameter is required"))
			})
		})
	})
})

var _ = Describe("Namespace Auto-Creation", func() {
	var (
		ctx       context.Context
		k8sClient client.Client
		server    *APIServer
		router    *gin.Engine
		recorder  *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		ctx = context.Background()
		// Create a scheme with our API types registered
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(acp.AddToScheme(scheme)).To(Succeed())

		// Create a fake client
		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		// Create the API server with the client
		server = NewAPIServer(k8sClient, ":8080")
		router = server.router
		recorder = httptest.NewRecorder()
	})

	It("should automatically create a non-existent namespace for agent creation", func() {
		// Create an LLM in the namespace we'll use
		llm := &acp.LLM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "auto-ns-llm",
				Namespace: "auto-created-namespace",
			},
			Spec: acp.LLMSpec{
				Provider: "test-provider",
				Parameters: acp.BaseConfig{
					Model: "test-model",
				},
			},
		}
		Expect(k8sClient.Create(ctx, llm)).To(Succeed())

		// Create agent in a namespace that doesn't exist yet
		reqBody := CreateAgentRequest{
			Namespace: "auto-created-namespace",
			Name:      "auto-ns-agent",
			LLM: LLMDefinition{
				Name:     "auto-ns-llm",
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "sk-test-key-auto",
			},
			SystemPrompt: "Testing auto namespace creation",
		}
		jsonBody, err := json.Marshal(reqBody)
		Expect(err).NotTo(HaveOccurred())

		req := httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		// Verify the agent was created successfully
		Expect(recorder.Code).To(Equal(http.StatusCreated))

		// Verify the namespace exists
		var namespace corev1.Namespace
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "auto-created-namespace"}, &namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(namespace.Name).To(Equal("auto-created-namespace"))

		// Verify the agent was created in the namespace
		var agent acp.Agent
		err = k8sClient.Get(ctx, client.ObjectKey{
			Namespace: "auto-created-namespace",
			Name:      "auto-ns-agent",
		}, &agent)
		Expect(err).NotTo(HaveOccurred())
		Expect(agent.Namespace).To(Equal("auto-created-namespace"))
	})

	It("should automatically create a non-existent namespace for task creation", func() {
		// Create an agent first
		agent := &acp.Agent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-test-agent",
				Namespace: "task-namespace",
			},
			Spec: acp.AgentSpec{},
		}

		// Create the task namespace
		taskNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "task-namespace",
			},
		}
		Expect(k8sClient.Create(ctx, taskNs)).To(Succeed())
		Expect(k8sClient.Create(ctx, agent)).To(Succeed())

		// Create a task in a namespace that doesn't exist yet
		reqBody := CreateTaskRequest{
			Namespace:   "task-namespace",
			AgentName:   "task-test-agent",
			UserMessage: "Testing auto namespace creation for tasks",
		}
		jsonBody, err := json.Marshal(reqBody)
		Expect(err).NotTo(HaveOccurred())

		req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		// Verify the task was created successfully
		Expect(recorder.Code).To(Equal(http.StatusCreated))

		// Confirm a task was created
		var taskList acp.TaskList
		Expect(k8sClient.List(ctx, &taskList, client.InNamespace("task-namespace"))).To(Succeed())
		Expect(len(taskList.Items)).To(Equal(1))
	})

	It("should handle namespace creation errors", func() {
		// Create an error client that fails on namespace operations
		errorClient := &errorK8sClient{Client: k8sClient, returnErrorOnGetNamespace: true}
		errorServer := NewAPIServer(errorClient, ":8080")
		errorRouter := errorServer.router

		// Create the request body
		reqBody := CreateAgentRequest{
			Namespace: "error-namespace",
			Name:      "error-ns-agent",
			LLM: LLMDefinition{
				Name:     "error-llm",
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "sk-test-key-error",
			},
			SystemPrompt: "Testing namespace error handling",
		}
		jsonBody, err := json.Marshal(reqBody)
		Expect(err).NotTo(HaveOccurred())

		req := httptest.NewRequest(http.MethodPost, "/v1/agents", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		recorder = httptest.NewRecorder()
		errorRouter.ServeHTTP(recorder, req)

		// Verify we get an internal server error
		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		var errorResponse map[string]string
		err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
		Expect(err).NotTo(HaveOccurred())
		Expect(errorResponse["error"]).To(ContainSubstring("Failed to ensure namespace exists"))
	})
})
