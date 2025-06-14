# E2E Test Framework Plan - Controller-Based Testing

## Objective
Create a new e2e testing framework that uses a temporary isolated Kubernetes cluster with real controllers running, using Go code for assertions instead of shell commands.

after reading this, read the WHOLE acp/docs/getting-started.md file to understand the manual testing process we're eventually replacing.

## Background
Current e2e tests:
- Use shell commands (`kubectl`) for all operations
- Depend on external cluster state
- Hard to debug and maintain
- Slow due to process spawning

Desired approach:
- Use envtest to create isolated API server
- Run controllers against the test cluster
- Use go k8s client for resource creation
- Assert with Go code (like controller tests)

## Key Differences from Controller Tests
- Controller instantiate a single controller and Reconcile explicitly, we'll run the controller in their reconcile loop in the background 
- Controller tests focus on single controller, we test full integration
- We need to start all controllers in the test setup, and tear them down after the suite

## Implementation Plan

### 1. Create New Test Package Structure
```
acp/test/e2e/
├── e2e_suite_test.go    (existing shell-based)
├── framework/                  (new)
    ├── framework.go            (test helpers)
    ├── getting_started/        (first test)
    │   ├── suite_test.go           (test setup)
    │   └── test_getting_started.go (actual test)
    └── mcp_tools/                  (EVENTUALLY - second test)
        ├── suite_test.go           (test setup)
        └── test_mcp_tools.go       (actual test with Describe)
    ... more test dirs
```

### 2. Test Framework Design

```go
// framework/framework.go
package framework

import (
    "context"
    "path/filepath"
    
    "k8s.io/client-go/kubernetes/scheme"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/envtest"
    ctrl "sigs.k8s.io/controller-runtime"
    
    acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
    // Import all controllers
)

type TestFramework struct {
    ctx       context.Context
    cancel    context.CancelFunc
    testEnv   *envtest.Environment
    k8sClient client.Client
    mgr       ctrl.Manager
}

func NewTestFramework() *TestFramework {
    return &TestFramework{}
}

func (tf *TestFramework) Start() error {
    // 1. Setup envtest
    tf.testEnv = &envtest.Environment{
        // may need to change the paths / number of ".." segments
        CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
        ErrorIfCRDPathMissing: true,
        BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s", "1.32.0-darwin-arm64"),
    }
    
    // 2. Start test environment
    cfg, err := tf.testEnv.Start()
    
    // 3. Create manager
    tf.mgr, err = ctrl.NewManager(cfg, ctrl.Options{
        Scheme: scheme.Scheme,
    })
    
    // 4. Setup all controllers
    // This is the key difference - we start real controllers
    if err = (&controllers.LLMReconciler{
        Client: tf.mgr.GetClient(),
        Scheme: tf.mgr.GetScheme(),
    }).SetupWithManager(tf.mgr); err != nil {
        return err
    }
    // IMPORTANT: Repeat the above for Agent, Task, MCPServer, ToolCall controllers
    
    
    // 5. Start manager in goroutine
    go func() {
        if err := tf.mgr.Start(tf.ctx); err != nil {
            panic(err)
        }
    }()
    
    // 6. Create client for tests
    tf.k8sClient = tf.mgr.GetClient()
    
    return nil
}
```

### 3. Basic Test Implementation

```go
// framework/basic_test.go
package framework

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"
    
    acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

var _ = Describe("Basic Agent Task Flow", func() {
    var (
        namespace string
        secret    *corev1.Secret
        llm       *acp.LLM
        agent     *acp.Agent
        task      *acp.Task
    )
    
    BeforeEach(func() {
        // Create unique namespace for test isolation
        namespace = "test-" + uuid.New().String()[:8]
        ns := &corev1.Namespace{
            ObjectMeta: metav1.ObjectMeta{Name: namespace},
        }
        Expect(k8sClient.Create(ctx, ns)).To(Succeed())
        
        // Create OpenAI secret
        secret = &corev1.Secret{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "openai",
                Namespace: namespace,
            },
            StringData: map[string]string{
                "OPENAI_API_KEY": os.Getenv("OPENAI_API_KEY"),
            },
        }
        Expect(k8sClient.Create(ctx, secret)).To(Succeed())
    })
    
    AfterEach(func() {
        // Clean up namespace (cascades to all resources)
        ns := &corev1.Namespace{
            ObjectMeta: metav1.ObjectMeta{Name: namespace},
        }
        Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
    })
    
    It("should create agent and process task successfully", func() {
        By("creating an LLM resource")
        llm = &acp.LLM{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "gpt-4o",
                Namespace: namespace,
            },
            Spec: acp.LLMSpec{
                Provider: "openai",
                Parameters: map[string]interface{}{
                    "model": "gpt-4o",
                },
                APIKeyFrom: &acp.SecretKeyRef{
                    Name: "openai",
                    Key:  "OPENAI_API_KEY",
                },
            },
        }
        Expect(k8sClient.Create(ctx, llm)).To(Succeed())
        
        By("waiting for LLM to be ready")
        Eventually(func() bool {
            err := k8sClient.Get(ctx, client.ObjectKeyFromObject(llm), llm)
            if err != nil {
                return false
            }
            return llm.Status.Ready
        }, timeout, interval).Should(BeTrue())
        
        By("creating an Agent resource")
        agent = &acp.Agent{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "my-assistant",
                Namespace: namespace,
            },
            Spec: acp.AgentSpec{
                LLMRef: acp.LocalObjectReference{
                    Name: "gpt-4o",
                },
                System: "You are a helpful assistant.",
            },
        }
        Expect(k8sClient.Create(ctx, agent)).To(Succeed())
        
        By("waiting for Agent to be ready")
        Eventually(func() bool {
            err := k8sClient.Get(ctx, client.ObjectKeyFromObject(agent), agent)
            if err != nil {
                return false
            }
            return agent.Status.Ready
        }, timeout, interval).Should(BeTrue())
        
        By("creating a Task")
        task = &acp.Task{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "hello-world",
                Namespace: namespace,
            },
            Spec: acp.TaskSpec{
                AgentRef: acp.LocalObjectReference{
                    Name: "my-assistant",
                },
                UserMessage: "What is the capital of the moon?",
            },
        }
        Expect(k8sClient.Create(ctx, task)).To(Succeed())
        
        By("waiting for Task to complete")
        Eventually(func() string {
            err := k8sClient.Get(ctx, client.ObjectKeyFromObject(task), task)
            if err != nil {
                return ""
            }
            return task.Status.Status
        }, timeout, interval).Should(Equal("Completed"))
        
        By("verifying the task response")
        Expect(task.Status.ContextWindow).To(HaveLen(2))
        Expect(strings.ToLower(task.Status.ContextWindow[1].Content)).To(ContainSubstring("moon"))
        Expect(strings.ToLower(task.Status.ContextWindow[1].Content)).To(ContainSubstring("does not have a capital"))

        // use the k8sClient to check for events in the apiserver
        events := &corev1.EventList{}
        Expect(k8sClient.List(ctx, events)).To(Succeed())
        Expect(events.Items).To(HaveLen(1))
        Expect(events.Items[0].Reason).To(Equal("TaskCompleted")) // might need to tweak this assertion
        Expect(events.Items[0].Message).To(ContainSubstring("Task completed successfully")) // might need to tweak this assertion
    })
})
```

### 4. Suite Setup

```go
// framework/suite_test.go
package framework

import (
    "testing"
    "time"
    
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var (
    tf       *TestFramework
    ctx      context.Context
    k8sClient client.Client
    
    timeout  = time.Second * 30
    interval = time.Millisecond * 250
)

func TestFramework(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "E2E Framework Suite")
}

var _ = BeforeSuite(func() {
    tf = NewTestFramework()
    Expect(tf.Start()).To(Succeed())
    
    ctx = tf.ctx
    k8sClient = tf.k8sClient
})

var _ = AfterSuite(func() {
    Expect(tf.Stop()).To(Succeed())
})
```

### 5. Key Implementation Challenges

#### Controller Manager Setup
Need to properly initialize all controllers with their dependencies:
- MCPManager for MCPServer controller
- Event recorder for all controllers
- Proper scheme registration

#### Real Services
For true e2e tests, we need to decide on:
- **LLM calls**: Use OpenAI API, anthropic API, using vars from env
- **MCP servers**: Use real MCP servers and lifecycle them with real components
- **HumanLayer API**: Use HumanLayer API with os.getenv("HUMANLAYER_API_KEY")

#### Timing and Eventual Consistency
- Controllers run asynchronously
- Need proper Eventually() timeouts
- May need to add delays for controller processing
- isolating each getting-started section in its own dir/cluster/suite ensures tests can be run in parallel cleanly and quickly

### 6. Advantages Over Current Approach

1. **Speed**: No process spawning, direct API calls
2. **Debugging**: Can set breakpoints, see full stack traces
3. **Isolation**: Each test gets its own namespace
4. **Reliability**: No dependency on external cluster state
5. **Maintainability**: Go code is easier to refactor than shell scripts

### 7. Migration Strategy

1. Start with new framework package alongside existing e2e tests
2. Implement basic test case (LLM → Agent → Task)
3. Add more complex scenarios incrementally
4. Eventually deprecate shell-based tests

### 8. Future Enhancements

- Parallel test execution (each in its own namespace)
- Test helpers for common operations
- Performance benchmarks
- Chaos testing (kill controllers mid-operation)
- Multi-controller interaction tests

## Technical Decisions

### Why envtest?
- Provides real Kubernetes API server
- Lightweight compared to kind
- Fast startup/shutdown
- Good integration with controller-runtime

### Why real controllers?
- Tests actual reconciliation loops
- Catches integration issues
- More realistic than mocked dependencies

### Test Data Management
- Use unique namespaces for isolation
- Clean up after each test
- Consider test fixtures for complex scenarios

## Success Criteria

1. Framework starts successfully with all controllers
2. Basic test (LLM → Agent → Task) passes
3. Tests run faster than shell-based equivalent
4. Easy to add new test cases
5. Clear error messages on failures

This approach will work because:
- envtest provides a real API server
- controller-runtime managers can run multiple controllers
- We control the full lifecycle in the test
- we fully test external dependencies
