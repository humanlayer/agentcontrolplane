package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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
	{
		// Task endpoints
		tasks := v1.Group("/tasks")
		{
			tasks.GET("", s.listTasks)
			tasks.GET("/:id", s.getTask)
		}
	}
}

// Start begins listening for requests in a goroutine
func (s *APIServer) Start(ctx context.Context) error {
	go func() {
		log.FromContext(ctx).Info("Starting API server", "port", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.FromContext(ctx).Error(err, "API server failed")
		}
	}()
	return nil
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
