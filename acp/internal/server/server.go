package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/validation"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateTaskRequest defines the structure of the request body for creating a task
type CreateTaskRequest struct {
	Namespace     string        `json:"namespace,omitempty"`     // Optional, defaults to "default"
	AgentName     string        `json:"agentName"`               // Required
	UserMessage   string        `json:"userMessage,omitempty"`   // Optional if contextWindow is provided
	ContextWindow []acp.Message `json:"contextWindow,omitempty"` // Optional if userMessage is provided
}

// APIServer represents the REST API server
type APIServer struct {
	client     client.Client
	httpServer *http.Server
	router     *gin.Engine
}

// NewAPIServer creates a new API server
func NewAPIServer(client client.Client, port string) *APIServer {
	router := gin.Default()
	server := &APIServer{
		client: client,
		router: router,
		httpServer: &http.Server{
			Addr:    port,
			Handler: router,
		},
	}

	// Register routes
	server.registerRoutes()

	return server
}

// registerRoutes sets up all API endpoints
func (s *APIServer) registerRoutes() {
	// Health check endpoint (unversioned)
	s.router.GET("/status", s.getStatus)

	// API v1 routes
	v1 := s.router.Group("/v1")

	// Task endpoints
	tasks := v1.Group("/tasks")
	tasks.GET("", s.listTasks)
	tasks.GET("/:id", s.getTask)
	tasks.POST("", s.createTask)
}

// Start begins listening for requests in a goroutine
func (s *APIServer) Start(ctx context.Context) error {
	errChan := make(chan error, 1)

	go func() {
		log.FromContext(ctx).Info("Starting API server", "port", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.FromContext(ctx).Error(err, "API server failed")
			errChan <- err
		}
	}()

	// Optional: wait for either context cancellation or server error
	select {
	case err := <-errChan:
		return errors.Wrap(err, "server error")
	case <-ctx.Done():
		return s.httpServer.Shutdown(context.Background())
	}
}

// Stop gracefully shuts down the server
func (s *APIServer) Stop(ctx context.Context) error {
	log.FromContext(ctx).Info("Stopping API server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}

// API handler methods
func (s *APIServer) getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": "v1alpha1",
	})
}

func (s *APIServer) listTasks(c *gin.Context) {
	ctx := c.Request.Context()
	logger := log.FromContext(ctx)

	// Get namespace from query parameter or use default
	namespace := c.DefaultQuery("namespace", "")

	// Initialize task list
	var taskList acp.TaskList

	// List tasks
	listOpts := []client.ListOption{}
	if namespace != "" {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}

	if err := s.client.List(ctx, &taskList, listOpts...); err != nil {
		logger.Error(err, "Failed to list tasks")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list tasks: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, taskList.Items)
}

func (s *APIServer) getTask(c *gin.Context) {
	ctx := c.Request.Context()
	logger := log.FromContext(ctx)
	id := c.Param("id")
	namespace := c.DefaultQuery("namespace", "default")

	// Initialize task
	var task acp.Task

	// Get the task
	if err := s.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: id}, &task); err != nil {
		logger.Error(err, "Failed to get task", "name", id, "namespace", namespace)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task not found: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// createTask handles the creation of a new task
func (s *APIServer) createTask(c *gin.Context) {
	ctx := c.Request.Context()
	logger := log.FromContext(ctx)

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.AgentName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agentName is required"})
		return
	}

	if err := validation.ValidateTaskInput(req.UserMessage, req.ContextWindow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Check if agent exists
	var agent acp.Agent
	err := s.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: req.AgentName}, &agent)
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		logger.Error(err, "Failed to check agent existence")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check agent existence: " + err.Error()})
		return
	}

	// Generate (mostly) unique task name
	generatedName := req.AgentName + "-task-" + uuid.New().String()[:8]

	task := &acp.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generatedName,
			Namespace: namespace,
			Labels: map[string]string{
				"acp.humanlayer.dev/agent": req.AgentName,
			},
		},
		Spec: acp.TaskSpec{
			AgentRef:      acp.LocalObjectReference{Name: req.AgentName},
			UserMessage:   req.UserMessage,
			ContextWindow: req.ContextWindow,
		},
	}

	if err := s.client.Create(ctx, task); err != nil {
		if statusErr := new(apierrors.StatusError); errors.As(err, &statusErr) {
			status := statusErr.Status()
			c.JSON(int(status.Code), gin.H{"error": status.Message})
		} else {
			logger.Error(err, "Failed to create task")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, task)
}
