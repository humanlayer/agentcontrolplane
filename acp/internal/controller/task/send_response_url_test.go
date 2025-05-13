package task

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	humanlayerapi "github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// initTestReconciler creates a minimal TaskReconciler for testing
func initTestReconciler(t *testing.T) (*TaskReconciler, context.Context) {
	// Initialize logger
	logger := zap.New(zap.UseDevMode(true))
	ctx := context.Background()
	ctx = log.IntoContext(ctx, logger)

	// Create a reconciler
	scheme := runtime.NewScheme()
	err := acp.AddToScheme(scheme)
	assert.NoError(t, err, "Failed to add API schema")
	
	return &TaskReconciler{
		Scheme:   scheme,
		recorder: record.NewFakeRecorder(10),
	}, ctx
}

func TestSendFinalResultToResponseUrl(t *testing.T) {
	// Create a channel to synchronize between test and handler
	requestReceived := make(chan struct{})
	
	// Track the received request for verification
	var receivedRequest humanlayerapi.HumanContactInput
	var receivedMutex sync.Mutex

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and content type
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Decode the request body
		decoder := json.NewDecoder(r.Body)
		var req humanlayerapi.HumanContactInput
		err := decoder.Decode(&req)
		assert.NoError(t, err)

		// Store the request for later verification
		receivedMutex.Lock()
		receivedRequest = req
		receivedMutex.Unlock()

		// Send a success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))

		// Notify that request was received
		close(requestReceived)
	}))
	defer server.Close()

	// Create a reconciler
	reconciler, ctx := initTestReconciler(t)

	// Test sending result
	testMsg := "This is the final task result"
	err := reconciler.sendFinalResultToResponseUrl(ctx, server.URL, testMsg)
	assert.NoError(t, err)

	// Wait for the request to be processed with a timeout
	select {
	case <-requestReceived:
		// Request was received, continue with assertions
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for request to be received")
	}

	// Verify the request content
	receivedMutex.Lock()
	defer receivedMutex.Unlock()

	// Verify run_id and call_id are set
	assert.NotEmpty(t, receivedRequest.GetRunId())
	assert.NotEmpty(t, receivedRequest.GetCallId())

	// Verify the message content
	assert.Equal(t, testMsg, receivedRequest.Spec.Msg)
}

// Test handling of error responses
func TestSendFinalResultToResponseUrl_ErrorResponse(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"something went wrong"}`))
	}))
	defer server.Close()

	// Create a reconciler
	reconciler, ctx := initTestReconciler(t)

	// Test sending result
	err := reconciler.sendFinalResultToResponseUrl(ctx, server.URL, "test message")
	
	// Should return an error due to non-200 response
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "received non-success status code: 500")
}

// Test handling of connection errors
func TestSendFinalResultToResponseUrl_ConnectionError(t *testing.T) {
	// Create a reconciler
	reconciler, ctx := initTestReconciler(t)

	// Use an invalid URL to cause a connection error
	err := reconciler.sendFinalResultToResponseUrl(ctx, "http://localhost:1", "test message")
	
	// Should return an error due to connection failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send HTTP request")
}