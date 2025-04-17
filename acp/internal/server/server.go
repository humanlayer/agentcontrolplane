package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
	// Health check endpoint
	s.router.GET("/status", s.getStatus)

	// Task endpoints
	tasks := s.router.Group("/tasks")
	{
		tasks.GET("", s.listTasks)
		tasks.GET("/:id", s.getTask)
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
		"status": "ok",
		"version": "v1alpha1",
	})
}

func (s *APIServer) listTasks(c *gin.Context) {
	// Using stub data
	tasks := []gin.H{
		{
			"id":        "task-1",
			"name":      "Sample Task 1",
			"status":    "Running",
			"createdAt": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		},
		{
			"id":        "task-2",
			"name":      "Sample Task 2",
			"status":    "Completed",
			"createdAt": time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
		},
	}
	
	c.JSON(http.StatusOK, tasks)
}

func (s *APIServer) getTask(c *gin.Context) {
	id := c.Param("id")
	
	// Return stub data based on the requested ID
	task := gin.H{
		"id":          id,
		"name":        "Sample Task " + id,
		"status":      "Running",
		"description": "This is a stub task with ID " + id,
		"createdAt":   time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"details": gin.H{
			"agent":     "agent-123",
			"priority":  "high",
			"attempts":  2,
			"namespace": c.DefaultQuery("namespace", "default"),
		},
	}
	
	c.JSON(http.StatusOK, task)
}