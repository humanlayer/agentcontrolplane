package task

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	humanlayerapi "github.com/humanlayer/agentcontrolplane/acp/internal/humanlayerapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// initTestReconciler creates a minimal TaskReconciler for testing
func initTestReconciler() (*TaskReconciler, context.Context) {
	// Initialize logger
	logger := zap.New(zap.UseDevMode(true))
	ctx := context.Background()
	ctx = log.IntoContext(ctx, logger)

	// Create a reconciler
	scheme := runtime.NewScheme()
	err := acp.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred(), "Failed to add API schema")

	return &TaskReconciler{
		Scheme:   scheme,
		recorder: record.NewFakeRecorder(10),
	}, ctx
}

var _ = Describe("ResponseURL Functionality", func() {
	Context("when sending results to responseURL", func() {
		It("successfully sends the result and verifies content", func() {
			// Create a channel to synchronize between test and handler
			requestReceived := make(chan struct{})

			// Track the received request for verification
			var receivedRequest humanlayerapi.HumanContactInput
			var receivedMutex sync.Mutex

			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method and content type
				Expect(r.Method).To(Equal("POST"))
				Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

				// Decode the request body
				decoder := json.NewDecoder(r.Body)
				var req humanlayerapi.HumanContactInput
				err := decoder.Decode(&req)
				Expect(err).NotTo(HaveOccurred())

				// Store the request for later verification
				receivedMutex.Lock()
				receivedRequest = req
				receivedMutex.Unlock()

				// Send a success response
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"success"}`))

				// Notify that request was received
				close(requestReceived)
			}))
			defer server.Close()

			// Create a reconciler
			reconciler, ctx := initTestReconciler()

			// Test sending result
			testMsg := "This is the final task result"
			err := reconciler.sendFinalResultToResponseURL(ctx, server.URL, testMsg)
			Expect(err).NotTo(HaveOccurred())

			// Wait for the request to be processed with a timeout
			Eventually(requestReceived).Should(BeClosed(), "Timed out waiting for request to be received")

			// Verify the request content
			receivedMutex.Lock()
			defer receivedMutex.Unlock()

			// Verify run_id and call_id are set
			Expect(receivedRequest.GetRunId()).NotTo(BeEmpty())
			Expect(receivedRequest.GetCallId()).NotTo(BeEmpty())

			// Verify the message content
			Expect(receivedRequest.Spec.Msg).To(Equal(testMsg))
		})

		It("handles error responses appropriately", func() {
			// Create a test server that returns an error
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"something went wrong"}`))
			}))
			defer server.Close()

			// Create a reconciler
			reconciler, ctx := initTestReconciler()

			// Test sending result
			err := reconciler.sendFinalResultToResponseURL(ctx, server.URL, "test message")

			// Should return an error due to non-200 response
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("HTTP error from responseURL (status 500)"))
		})

		It("handles connection errors appropriately", func() {
			// Create a reconciler
			reconciler, ctx := initTestReconciler()

			// Use an invalid URL to cause a connection error
			err := reconciler.sendFinalResultToResponseURL(ctx, "http://localhost:1", "test message")

			// Should return an error due to connection failure
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to send HTTP request"))
		})
	})
})
