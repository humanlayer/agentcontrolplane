/*
Copyright 2025 the Agent Control Plane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/controller/agent"
	"github.com/humanlayer/agentcontrolplane/acp/internal/controller/llm"
	"github.com/humanlayer/agentcontrolplane/acp/internal/controller/mcpserver"
	"github.com/humanlayer/agentcontrolplane/acp/internal/controller/task"
	"github.com/humanlayer/agentcontrolplane/acp/internal/controller/toolcall"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
)

// TestFramework provides a unified test environment with all controllers running
type TestFramework struct {
	TestEnv      *envtest.Environment
	Config       *rest.Config
	Client       client.Client
	Manager      manager.Manager
	ctx          context.Context
	cancel       context.CancelFunc
	managerReady chan struct{}
	startOnce    sync.Once
}

// NewTestFramework creates a new test framework instance
func NewTestFramework() *TestFramework {
	return &TestFramework{
		managerReady: make(chan struct{}),
	}
}

// Start initializes envtest environment and starts all controllers
func (f *TestFramework) Start() error {
	var err error
	f.startOnce.Do(func() {
		err = f.doStart()
	})
	return err
}

func (f *TestFramework) doStart() error {
	// Create context
	f.ctx, f.cancel = context.WithCancel(context.Background())

	// Setup envtest environment
	f.TestEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// Find binary directory dynamically
	if binDir := getFirstFoundEnvTestBinaryDir(); binDir != "" {
		f.TestEnv.BinaryAssetsDirectory = binDir
	}

	// Start test environment
	var err error
	f.Config, err = f.TestEnv.Start()
	if err != nil {
		return fmt.Errorf("failed to start test environment: %w", err)
	}

	// Add schemes
	err = acp.AddToScheme(scheme.Scheme)
	if err != nil {
		return fmt.Errorf("failed to add schemes: %w", err)
	}

	// Create client
	f.Client, err = client.New(f.Config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Create manager with proper options
	f.Manager, err = ctrl.NewManager(f.Config, ctrl.Options{
		Scheme:                 scheme.Scheme,
		HealthProbeBindAddress: "0",   // Disable health probes
		LeaderElection:         false, // Disable leader election in tests
		Metrics: metricsserver.Options{
			BindAddress: "0", // Disable metrics
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	// Setup all controllers
	if err = f.setupControllers(); err != nil {
		return fmt.Errorf("failed to setup controllers: %w", err)
	}

	// Start manager in background
	go func() {
		defer close(f.managerReady)
		if err := f.Manager.Start(f.ctx); err != nil {
			fmt.Printf("Manager error: %v\n", err)
		}
	}()

	// Give the manager time to start up
	time.Sleep(1 * time.Second)

	// Wait for the manager's cache to sync
	if !f.Manager.GetCache().WaitForCacheSync(f.ctx) {
		return fmt.Errorf("failed to wait for cache sync")
	}

	return nil
}

// Stop cleans up the test environment
func (f *TestFramework) Stop() error {
	if f.cancel != nil {
		f.cancel()
	}
	if f.TestEnv != nil {
		return f.TestEnv.Stop()
	}
	return nil
}

// GetClient returns the Kubernetes client
func (f *TestFramework) GetClient() client.Client {
	return f.Client
}

// GetContext returns the context
func (f *TestFramework) GetContext() context.Context {
	return f.ctx
}

// GetManager returns the controller manager
func (f *TestFramework) GetManager() manager.Manager {
	return f.Manager
}

// WaitForControllersReady waits for controllers to be ready to process events
func (f *TestFramework) WaitForControllersReady(ctx context.Context) error {
	// Wait for the manager ready signal
	select {
	case <-f.managerReady:
		// Manager has been started
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for controllers")
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for controllers to be ready")
	}

	// Additional time for controllers to initialize
	time.Sleep(100 * time.Millisecond)
	return nil
}

// setupControllers initializes all required controllers
func (f *TestFramework) setupControllers() error {
	// Create shared MCP manager for testing
	mcpManagerInstance := mcpmanager.NewMCPServerManagerWithClient(f.Manager.GetClient())

	// Create no-op tracer for testing
	noopTracer := noop.NewTracerProvider().Tracer("test")

	// Setup LLM controller
	if err := (&llm.LLMReconciler{
		Client: f.Manager.GetClient(),
		Scheme: f.Manager.GetScheme(),
	}).SetupWithManager(f.Manager); err != nil {
		return fmt.Errorf("failed to setup LLM controller: %w", err)
	}

	// Setup Agent controller using factory method
	agentReconciler, err := agent.NewAgentReconcilerForManager(f.Manager)
	if err != nil {
		return fmt.Errorf("failed to create agent reconciler: %w", err)
	}
	if err := agentReconciler.SetupWithManager(f.Manager); err != nil {
		return fmt.Errorf("failed to setup Agent controller: %w", err)
	}

	// Setup Task controller with required dependencies
	if err := (&task.TaskReconciler{
		Client:     f.Manager.GetClient(),
		Scheme:     f.Manager.GetScheme(),
		MCPManager: mcpManagerInstance,
		Tracer:     noopTracer,
	}).SetupWithManager(f.Manager); err != nil {
		return fmt.Errorf("failed to setup Task controller: %w", err)
	}

	// Setup MCPServer controller
	if err := (&mcpserver.MCPServerReconciler{
		Client: f.Manager.GetClient(),
		Scheme: f.Manager.GetScheme(),
	}).SetupWithManager(f.Manager); err != nil {
		return fmt.Errorf("failed to setup MCPServer controller: %w", err)
	}

	// Setup ToolCall controller with required dependencies
	if err := (&toolcall.ToolCallReconciler{
		Client:     f.Manager.GetClient(),
		Scheme:     f.Manager.GetScheme(),
		MCPManager: mcpManagerInstance,
		Tracer:     noopTracer,
	}).SetupWithManager(f.Manager); err != nil {
		return fmt.Errorf("failed to setup ToolCall controller: %w", err)
	}

	return nil
}

// Helper function from existing patterns
func getFirstFoundEnvTestBinaryDir() string {
	// This mirrors the pattern from the existing suite_test.go files
	basePath := filepath.Join("..", "..", "..", "bin", "k8s")
	entries, err := filepath.Glob(filepath.Join(basePath, "*"))
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if info, err := filepath.Glob(filepath.Join(entry, "*")); err == nil && len(info) > 0 {
			return entry
		}
	}
	return ""
}
