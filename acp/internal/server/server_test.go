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
			Expect(errorResponse["error"]).To(Equal("userMessage is required"))
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
	return e.Client.Create(ctx, obj, opts...)
}

func (e *errorK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return e.Client.Get(ctx, key, obj, opts...)
}