# SmallChain Development Guide

## Root-Level Makefile Commands

The root-level Makefile provides convenient commands for managing the entire project without having to change directories. You can run all commands from the project root.

### Pattern-Matching Commands
- `make kubechain-<command>`: Run any target from the kubechain Makefile (e.g., `make kubechain-fmt`)
- `make example-<command>`: Run any target from the kubechain-example Makefile (e.g., `make example-kind-up`)

### Cluster Management
- `make cluster-up`: Create the Kind cluster
- `make cluster-down`: Delete the Kind cluster

### Operator Management
- `make build-operator`: Build the Kubechain operator binary
- `make deploy-operator`: Deploy the Kubechain operator to the local Kind cluster
- `make undeploy-operator`: Undeploy the operator and remove CRDs

### Resource Management
- `make deploy-samples`: Deploy sample resources to the cluster
- `make undeploy-samples`: Remove sample resources
- `make show-samples`: Show status of sample resources
- `make watch-samples`: Watch status of sample resources with continuous updates

### UI and Observability
- `make deploy-ui`: Deploy the Kubechain UI
- `make deploy-otel`: Deploy the observability stack
- `make undeploy-otel`: Remove the observability stack
- `make otel-access`: Display access instructions for monitoring stack

### Testing
- `make test-operator`: Run unit tests for the operator
- `make test-e2e`: Run end-to-end tests (requires a running cluster)

### All-in-One Commands
- `make setup-all`: Set up the entire environment (cluster, operator, samples, UI, observability)
- `make clean-all`: Clean up everything (samples, operator, observability, cluster)

### Help
- `make help`: Display all available commands with descriptions

## Go (Kubechain) Commands

You can run these commands directly in the kubechain directory or use the pattern-matching syntax from the root:

- Build: `cd kubechain && make build` or `make kubechain-build`
- Format: `cd kubechain && make fmt` or `make kubechain-fmt`
- Lint: `cd kubechain && make lint` or `make kubechain-lint`
- Run tests: `cd kubechain && make test` or `make kubechain-test`
- Run single test: `cd kubechain && go test -v ./internal/controller/llm -run TestLLMController`
- Run e2e tests: `cd kubechain && make test-e2e` or `make kubechain-test-e2e`

## Makefiles Overview

### Main Kubechain Makefile (/kubechain/Makefile)

#### Development Commands
- `make fmt`: Format Go code
- `make vet`: Run Go vet
- `make lint`: Run golangci-lint on code
- `make lint-fix`: Run golangci-lint and fix issues
- `make test`: Run unit tests
- `make test-e2e`: Run end-to-end tests (requires a running Kind cluster)
- `make manifests`: Generate Kubernetes manifests (CRDs, RBAC) - **Important:** Run this after modifying CRD types or controller RBAC annotations
- `make generate`: Generate Go code (DeepCopy methods) - **Important:** Run this after adding new struct fields

#### Build Commands
- `make build`: Build the manager binary
- `make run`: Run the controller locally
- `make docker-build`: Build the controller Docker image
- `make docker-push`: Push the controller Docker image
- `make docker-load-kind`: Load Docker image into Kind cluster
- `make build-installer`: Generate install.yaml in the dist directory

#### Deployment Commands
- `make install`: Install CRDs into cluster
- `make uninstall`: Uninstall CRDs from cluster
- `make deploy`: Deploy controller to cluster (builds and pushes image)
- `make deploy-local-kind`: Deploy controller to local Kind cluster
- `make deploy-samples`: Deploy sample resources
- `make undeploy-samples`: Remove sample resources
- `make undeploy`: Undeploy controller
- `make show-samples`: Show status of sample resources
- `make watch-samples`: Watch status of sample resources with continuous updates

### Kubechain Example Makefile (/kubechain-example/Makefile)

#### Cluster Management
- `make kind-up`: Create Kind cluster
- `make kind-down`: Delete Kind cluster

#### Application Deployment
- `make operator-build`: Build Kubechain operator Docker image
- `make operator-deploy`: Deploy Kubechain operator to cluster
- `make ui-deploy`: Deploy Kubechain UI

#### Observability Stack
- `make otel-stack`: Deploy full observability stack (Prometheus, OpenTelemetry, Grafana, Tempo, Loki)
- `make otel-stack-down`: Remove observability stack
- `make otel-test`: Run test to generate OpenTelemetry data
- `make otel-access`: Display access instructions for monitoring stack

Individual components can be managed separately:
- `make prometheus-up/down`
- `make grafana-up/down`
- `make otel-up/down`
- `make tempo-up/down`
- `make loki-up/down`

## Documentation

The project includes detailed documentation in the `/kubechain/docs/` directory:

- [MCP Server Guide](/kubechain/docs/mcp-server.md) - Working with Model Control Protocol servers
- [CRD Reference](/kubechain/docs/crd-reference.md) - Complete reference for all Custom Resource Definitions
- [Kubebuilder Guide](/kubechain/docs/kubebuilder-guide.md) - How to develop with Kubebuilder in this project

## Typical Workflow

### Local Development with Kind Cluster

#### Using Root Makefile (Recommended)
1. Create local Kubernetes cluster: `make cluster-up`
2. Deploy the operator (includes installing CRDs): `make deploy-operator`
3. Deploy observability stack: `make deploy-otel`
4. Deploy sample resources: `make deploy-samples`
5. Watch resources: `make watch-samples`

Alternatively, use: `make setup-all` to perform steps 1-4 in a single command.

#### Using Directory Makefiles (Alternative)
1. Create local Kubernetes cluster: `cd kubechain-example && make kind-up`
2. Install CRDs: `cd kubechain && make install`
3. Build and deploy the controller: `cd kubechain && make deploy-local-kind`
4. Deploy observability stack: `cd kubechain-example && make otel-stack`
5. Deploy sample resources: `cd kubechain && make deploy-samples`
6. Watch resources: `cd kubechain && make watch-samples`

### Clean Up

#### Using Root Makefile (Recommended)
1. Clean up everything: `make clean-all`

Alternatively, clean up components individually:
1. Remove sample resources: `make undeploy-samples`
2. Undeploy the operator: `make undeploy-operator`
3. Remove observability stack: `make undeploy-otel`
4. Delete cluster: `make cluster-down`

#### Using Directory Makefiles (Alternative)
1. Remove sample resources: `cd kubechain && make undeploy-samples`
2. Undeploy the controller: `cd kubechain && make undeploy`
3. Remove observability stack: `cd kubechain-example && make otel-stack-down`
4. Delete cluster: `cd kubechain-example && make kind-down`

## Code Style Guidelines
### Go
- Follow standard Go code style with `gofmt`
- Use meaningful error handling with context
- Use dependency injection for controllers
- Test with Ginkgo/Gomega framework
- Document public functions with godoc

### Kubebuilder and CRD Development
- All resources should be in the `kubechain.humanlayer.dev` API group
- Use proper kubebuilder annotations for validation and RBAC
- Add RBAC annotations to all controllers to generate proper permissions
- Run `make manifests` after modifying CRD types or controller annotations
- Run `make generate` after adding new struct fields to generate DeepCopy methods
- When creating new resources, use `kubebuilder create api --group kubechain --version acp --kind YourResource --namespaced true --resource true --controller true`
- Ensure the PROJECT file contains entries for all resources before running `make manifests`
- Follow the detailed guidance in the [Kubebuilder Guide](/kubechain/docs/kubebuilder-guide.md)

### Kubernetes Resource Design

#### Don't use "config" in field names:

Bad:

```
spec:
  slackConfig:
    #...
  emailConfig:
    #...
```

Good:

```
spec:
  slack:
    # ...
  email:
    # ...
```

#### Prefer nil-able sub-objects over "type" fields:

This is more guidelines than rules, just consider it in cases when you a Resource that is a union type. There's
no great answer here because of how Go handles unions. (maybe the state-of-the-art has progressed since the last time I checked) -- dex

Bad:

```
spec: 
  type: slack
  slack:
    channelOrUserID: C1234567890
```

Good:

```
spec: 
  slack:
    channelOrUserID: C1234567890
```

In code, instead of 

```
switch (resource.Spec.Type) {
    case "slack":
        // ...
    case "email":
        // ...
}
```

check which object is non-nil and use that:

```
if resource.Spec.Slack != nil {
    // ...
} else if resource.Spec.Email != nil {
    // ...
} else if {
    // ...
}
```

### Markdown
- When writing markdown code blocks, do not indent the block, just use backticks to offset the code

## Testing Guidelines

Testing is a critical part of the development process, especially for Kubernetes controllers that manage complex state machines. This section outlines best practices for testing controllers, developing end-to-end tests, and mocking external dependencies.

### Kubernetes Controller Testing
- Use state-based testing to verify controller behavior
- Test each state transition independently
- Organize tests with focused, modular test setup
- Use test fixtures for consistent resource creation
- Write tests that serve as documentation of controller behavior

#### State-Based Testing Approach
- Controllers in Kubernetes are state machines; tests should reflect this
- Organize tests by state transitions with Context blocks named "StateA -> StateB"
- Each test should focus on a single state transition, not complete workflows
- Use per-test setup/teardown with defer pattern rather than BeforeEach/AfterEach
- Create modular test fixtures that can set up resources in specific states

Example state transition test structure:
```go
Context("'':'' -> Pending:Pending", func() {
    It("initializes to Pending:Pending and sets required fields", func() {
        // Set up resources needed for this specific test
        resource := testFixture.Setup(ctx)
        defer testFixture.Teardown(ctx)

        // Execute reconciliation
        result, err := reconciler.Reconcile(ctx, request)
        
        // Verify reconciliation was successful
        Expect(err).NotTo(HaveOccurred())
        
        // Verify expected state transition
        Expect(resource.Status.Status).To(Equal(myresource.StatusTypePending))
        Expect(resource.Status.Phase).To(Equal(myresource.PhasePending))
        Expect(resource.Status.StatusDetail).To(Equal("Initializing"))
        Expect(resource.Status.StartTime).NotTo(BeNil())
    })
})
```

#### Test Fixture Pattern
- Create test fixture structs for each resource type with Setup/Teardown methods
- Implement SetupWithStatus methods to create resources in specific states
- Provide sensible defaults for test resources
- Implement helper functions like setupSuiteObjects to create dependency chains
- Use reconciler factory functions to simplify test setup

Example test fixture:
```go
type TestResource struct {
    name     string
    resource *kubechain.Resource
}

func (t *TestResource) Setup(ctx context.Context) *kubechain.Resource {
    // Create the resource with default values
    resource := &kubechain.Resource{
        ObjectMeta: metav1.ObjectMeta{
            Name:      t.name,
            Namespace: "default",
        },
        Spec: kubechain.ResourceSpec{
            // Default values
        },
    }
    Expect(k8sClient.Create(ctx, resource)).To(Succeed())
    t.resource = resource
    return resource
}

func (t *TestResource) SetupWithStatus(ctx context.Context, status kubechain.ResourceStatus) *kubechain.Resource {
    resource := t.Setup(ctx)
    resource.Status = status
    Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())
    t.resource = resource
    return resource
}

func (t *TestResource) Teardown(ctx context.Context) {
    // Delete resource and handle potential errors (e.g., if already deleted)
    err := k8sClient.Delete(ctx, t.resource)
    if err != nil && !apierrors.IsNotFound(err) {
        Expect(err).NotTo(HaveOccurred())
    }
}
```

#### Organizing Tests by State Transitions
- Map out all valid state transitions for your controller
- Create a Context block for each state transition
- Include both happy path and error path transitions
- Use descriptive names for Context blocks that clearly show the transition
- When testing a multi-step workflow, break it into individual state transitions

Examples of state transition Context blocks:
- `Context("'':'' -> Pending:Pending")` - Initial reconciliation
- `Context("Pending:Pending -> Ready:Pending")` - Setup completion
- `Context("Ready:Pending -> Ready:Running")` - Start processing
- `Context("Ready:Running -> Ready:Succeeded")` - Successful completion
- `Context("Ready:Pending -> Error:Failed")` - Error handling
- `Context("Error:Failed -> Ready:Pending")` - Recovery attempts

#### Test Implementation Best Practices
- Use descriptive By() statements to explain test steps
- Ensure each test verifies both the state and any side effects
- Assert on specific fields that should change during the transition
- Use positive assertions (prefer `Expect(x).To(Equal(y))` over `Expect(x).NotTo(Equal(z))`) for clarity and readability
- Test event recording when events are part of the controller behavior
- Verify controller return values (Requeue, RequeueAfter)
- For tool calls or API interactions, use mock clients with verification
- Separate resource setup from test assertions

#### Avoid in Controller Tests
- Do not test multiple state transitions in a single test
- Avoid monolithic BeforeEach/AfterEach with shared test state
- Don't create resources that aren't needed for the specific test
- Don't test implementation details, focus on behavior
- Avoid testing the complete end-to-end flow in a single test

### End-to-End Testing
- Use E2E tests for verifying complete workflows across multiple controllers
- Keep unit tests focused on single-controller behavior
- Test controller collaborations, not just individual controllers
- Include full reconciliation cycles and verify expected steady state
- Test actual resource creation and status propagation

### Mocking External Dependencies
- Mock external API calls and HTTP services in controller tests
- Implement mock clients that return predetermined responses
- Verify calls to external services with expectations on arguments
- Use mock secrets for credentials in tests
- Consider using controller runtime fake clients for complex scenarios
- Use the `gomock` package (github.com/golang/mock/gomock) for generating mocks of interfaces
- For HTTP services, use httptest package from the standard library

### Status vs Phase in Controllers

When designing controllers, distinguish between Status and Phase:

- **Status** indicates the health or readiness of a resource. It answers: "Is the resource working correctly?"
  - Use StatusType values like "Ready", "Error", "Pending", "AwaitingHumanApproval"
  - Status reflects the current operational state of the resource
  - Status changes are typically cross-cutting (error handling, initialization)
  - Example values: "Ready", "Error", "Pending", "AwaitingHumanApproval", "ErrorRequestingHumanApproval"

- **Phase** indicates the progress of a resource in its lifecycle. It answers: "What stage of processing is the resource in?"
  - Use PhaseType values like "Pending", "Running", "Succeeded", "Failed", "AwaitingHumanInput"
  - Phase reflects the workflow stage of the resource
  - Phase changes represent forward progression through a workflow
  - Example values: "Pending", "Running", "Succeeded", "Failed", "AwaitingHumanInput", "AwaitingSubAgent"

#### Guidelines for choosing between Status and Phase

1. Use **Status** for a state when:
   - It indicates whether the resource is operational or not
   - It represents a cross-cutting concern affecting all states (like errors)
   - It focuses on readiness rather than progress

2. Use **Phase** for a state when:
   - It's part of a sequential progression
   - It represents a distinct stage in a workflow
   - It indicates what the resource is currently doing

3. When naming test cases, use the "Status:Phase -> Status:Phase" format to clearly communicate transitions:
   ```go
   Context("Pending:Pending -> Ready:Pending", func() {
       It("moves from Pending:Pending to Ready:Pending during setup", func() {
           // Test implementation
       })
   })
   ```

#### Implementation Guidelines

1. **Preserve Status During Phase Transitions**: When implementing workflow progression that only changes the Phase:
   ```go
   // Good: Only update Phase when transitioning to a new workflow stage
   // while preserving current Status (health)
   trtc.Status.Phase = acp.TaskRunToolCallPhaseAwaitingHumanApproval
   trtc.Status.StatusDetail = "Waiting for human approval"
   
   // Avoid: Don't modify Status when the change is just about workflow progress
   // trtc.Status.Status = someNewStatus // Don't do this when only the Phase is changing
   ```

2. **Change Status Only When Health State Changes**: Status should change only when the health or readiness of the resource changes:
   ```go
   // When a resource encounters an error
   trtc.Status.Status = acp.TaskRunToolCallStatusTypeError
   trtc.Status.Error = err.Error()
   
   // When a resource becomes ready
   trtc.Status.Status = acp.TaskRunToolCallStatusTypeReady
   ```

3. **Use Error Status with Descriptive Phase Values**: When handling errors, set Status to Error and use Phase to describe the specific error scenario:
   ```go
   // Helper function for setting error states
   func (r *MyReconciler) setStatusError(ctx context.Context, phase MyResourcePhase, eventType string, resource *MyResource, err error) {
       resource.Status.Status = MyResourceStatusTypeError // Always use Error status
       resource.Status.Phase = phase                     // Use specific Phase to describe the error
       resource.Status.Error = err.Error()
       r.recorder.Event(resource, corev1.EventTypeWarning, eventType, err.Error())
   }
   ```

4. **Early Return for Terminal States**: For resources that have workflow-based lifecycles with terminal states, add an early check in the Reconcile function to avoid unnecessary processing:
   ```go
   // For workflow-based resources like TaskRunToolCall that reach terminal states
   if resource.Status.Status == MyResourceStatusTypeError || 
      resource.Status.Phase == MyResourcePhaseSucceeded || 
      resource.Status.Phase == MyResourcePhaseFailed {
       logger.Info("Resource in terminal state, nothing to do", 
           "status", resource.Status.Status, 
           "phase", resource.Status.Phase)
       return ctrl.Result{}, nil
   }
   
   // Note: This pattern may not apply to long-running resources like Servers or Agents
   // that need to maintain their state continuously
   ```

5. **Test Error Transitions with Status:Phase Format**: When testing error transitions, follow the Status:Phase naming convention:
   ```go
   Context("Ready:Pending -> Error:ErrorRequestingApproval", func() {
       It("transitions to Error:ErrorRequestingApproval when API call fails", func() {
           // Test implementation
       })
   })
   ```