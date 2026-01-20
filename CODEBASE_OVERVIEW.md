# Agent Control Plane (ACP) - Codebase Overview & Onboarding Guide

**Last Updated:** 2026-01-20
**Purpose:** Comprehensive guide for understanding the ACP codebase structure and architecture

---

## Table of Contents
1. [Quick Start](#quick-start)
2. [What is ACP?](#what-is-acp)
3. [Repository Structure](#repository-structure)
4. [Custom Resources (The Data Model)](#custom-resources-the-data-model)
5. [Controllers (The Control Logic)](#controllers-the-control-logic)
6. [Internal Packages](#internal-packages)
7. [Testing Strategy](#testing-strategy)
8. [Build & Development Workflow](#build--development-workflow)
9. [Key Architectural Patterns](#key-architectural-patterns)
10. [Essential Files for Deep Dives](#essential-files-for-deep-dives)

---

## Quick Start

### The 5-Minute Mental Model

ACP is a **Kubernetes operator** that turns Kubernetes into an orchestration platform for LLM agents. Think of it as:

```
Agent (LLM + Tools + Prompt) → Task (Conversation) → ToolCalls (Actions)
```

**Example Flow:**
1. Create an `Agent` resource (defines which LLM, which tools, system prompt)
2. Create a `Task` resource (starts a conversation with the agent)
3. The operator orchestrates the LLM conversation loop, creating `ToolCall` resources as needed
4. The agent uses MCP servers to execute tools, can contact humans for approval, and produces results

### Quick Navigation

| I want to... | Go to... |
|--------------|----------|
| Understand the data model | [api/v1alpha1/](acp/api/v1alpha1/) |
| See how agents work | [internal/controller/agent/](acp/internal/controller/agent/) |
| Understand task orchestration | [internal/controller/task/](acp/internal/controller/task/) |
| Learn about tool execution | [internal/controller/toolcall/](acp/internal/controller/toolcall/) |
| Run locally | `make setup` from root |
| Run tests | `make -C acp test` |
| View examples | [config/samples/](acp/config/samples/) or [acp-example/agents/](acp-example/agents/) |

---

## What is ACP?

**Agent Control Plane** is a Kubernetes operator for managing Large Language Model (LLM) workflows. It provides:

- **Declarative Agent Definitions**: Define agents as Kubernetes resources
- **Multi-Provider LLM Support**: OpenAI, Anthropic, Vertex AI, Mistral
- **Extensible Tooling**: MCP (Model Control Protocol) servers for tool integration
- **Human-in-the-Loop**: HumanLayer integration for approvals and inputs
- **Conversation Management**: Automatic context window management and message history
- **Observability**: Full OpenTelemetry integration with traces and metrics
- **High Availability**: Distributed locking for multi-pod deployments

### Technology Stack

- **Language:** Go 1.23+
- **Framework:** Kubebuilder + controller-runtime
- **LLM Client:** `github.com/tmc/langchaingo`
- **MCP Client:** `github.com/mark3labs/mcp-go`
- **Web Framework:** Gin (for REST API)
- **Testing:** Ginkgo + Gomega + envtest
- **Telemetry:** OpenTelemetry (OTLP)

---

## Repository Structure

```
agentcontrolplane/
├── acp/                    # Main operator implementation
│   ├── api/v1alpha1/      # Custom Resource Definitions (CRDs)
│   ├── cmd/               # Application entry point (main.go)
│   ├── config/            # Kubernetes manifests and kustomize configs
│   ├── internal/          # Internal packages (not exported)
│   │   ├── controller/    # Controller implementations
│   │   ├── llmclient/     # LLM provider abstraction
│   │   ├── mcpmanager/    # MCP server connection management
│   │   ├── humanlayer/    # HumanLayer API client
│   │   ├── adapters/      # Tool format conversions
│   │   ├── server/        # REST API server
│   │   ├── validation/    # Resource validation
│   │   └── otel/          # OpenTelemetry setup
│   ├── docs/              # Documentation
│   ├── test/              # Test suites (e2e, utils)
│   ├── examples/          # Example YAML configurations
│   └── Makefile           # Build and development commands
│
├── acp-example/           # Example configurations and observability
│   ├── agents/            # Sample agent definitions
│   ├── kind/              # Kind cluster configs
│   ├── grafana/           # Grafana dashboards
│   ├── loki/              # Logging configuration
│   ├── otel/              # OpenTelemetry collector
│   └── prometheus/        # Prometheus monitoring
│
├── hack/                  # Development scripts and AI personas
│   ├── agent-developer.md         # Developer persona
│   ├── agent-integration-tester.md # Testing persona
│   ├── agent-merger.md            # Merger persona
│   └── agent-multiplan-manager.md # Manager persona
│
├── Makefile               # Root makefile (orchestrates acp/ and acp-example/)
├── CLAUDE.md              # AI assistant instructions
└── README.md              # Project README
```

### Directory Deep Dive

#### `/acp` - The Operator

This is where all the operator code lives. Structure follows Kubebuilder conventions:

- **api/v1alpha1/**: CRD type definitions (Go structs → Kubernetes resources)
- **cmd/**: Application entry point - initializes everything
- **config/**: Kubernetes manifests generated by Kubebuilder
- **internal/**: Private implementation packages
- **test/**: Test framework and utilities
- **docs/**: User and developer documentation

#### `/acp-example` - Observability & Examples

This provides a complete local development environment with:

- Grafana dashboards for visualizing agent execution
- Tempo for distributed tracing
- Loki for log aggregation
- Prometheus for metrics
- Sample agent definitions you can deploy immediately

#### `/hack` - AI Assistant Personas

Contains specialized personas for AI assistants (like Claude) to adopt when working on the codebase. Each persona has specific rules and workflows.

---

## Custom Resources (The Data Model)

**Location:** `acp/api/v1alpha1/`

ACP introduces 6 custom Kubernetes resources. Understanding these is key to understanding the system.

### 1. **LLM** (`llm_types.go`, 216 lines)

Represents an LLM provider configuration.

**Purpose:** Abstract LLM provider details (API keys, model names, parameters)

**Key Fields:**
- `Provider`: `openai`, `anthropic`, `vertexai`, `mistral`
- `Model`: Model identifier (e.g., `gpt-4`, `claude-sonnet-4`)
- `APIKeySecretRef`: Reference to Kubernetes Secret with API key
- `BaseURL`: Optional custom endpoint
- `Temperature`, `TopP`, `MaxTokens`: Generation parameters

**Example:**
```yaml
apiVersion: acp.cprime.dev/v1alpha1
kind: LLM
metadata:
  name: claude-sonnet
spec:
  provider: anthropic
  model: claude-sonnet-4-20250514
  apiKeySecretRef:
    name: anthropic-api-key
    key: api-key
```

**Controller:** `internal/controller/llm/` - Validates config and tests connectivity

---

### 2. **MCPServer** (`mcpserver_types.go`, 145 lines)

Represents a Model Control Protocol server that provides tools.

**Purpose:** Configure external tool providers (file systems, APIs, databases, etc.)

**Key Fields:**
- `Transport`: `stdio` (local process) or `http` (remote server)
- `Command`, `Args`, `Env`: For stdio transport
- `URL`: For HTTP transport
- `Status.Tools`: Discovered tools from the server

**Example:**
```yaml
apiVersion: acp.cprime.dev/v1alpha1
kind: MCPServer
metadata:
  name: filesystem
spec:
  transport: stdio
  command: npx
  args:
    - "-y"
    - "@modelcontextprotocol/server-filesystem"
    - "/tmp/workspace"
```

**Controller:** `internal/controller/mcpserver/` - Manages connections and tool discovery

---

### 3. **ContactChannel** (`contactchannel_types.go`, 139 lines)

Represents a human contact method for approvals/inputs.

**Purpose:** Enable human-in-the-loop workflows via HumanLayer

**Key Fields:**
- `ChannelType`: `slack`, `email`, `sms`, `whatsapp`
- `Slack.ChannelOrUser`, `Slack.BotToken`, `Slack.Context`
- `Email.Address`
- `SMS.PhoneNumber`
- `WhatsApp.PhoneNumber`

**Example:**
```yaml
apiVersion: acp.cprime.dev/v1alpha1
kind: ContactChannel
metadata:
  name: ops-team-slack
spec:
  channelType: slack
  slack:
    channelOrUser: "#ops-alerts"
    botTokenSecretRef:
      name: slack-bot-token
      key: token
```

**Controller:** `internal/controller/contactchannel/` - Validates channel configuration

---

### 4. **Agent** (`agent_types.go`, 127 lines)

Represents an LLM agent (LLM + Tools + System Prompt).

**Purpose:** Define reusable agent configurations

**Key Fields:**
- `LLMRef`: Reference to an LLM resource
- `SystemPrompt`: Instructions for the agent
- `MCPServerRefs`: List of MCP servers (tools available to agent)
- `ContactChannelRefs`: List of contact channels for human interaction
- `SubAgentRefs`: Other agents this agent can delegate to

**Example:**
```yaml
apiVersion: acp.cprime.dev/v1alpha1
kind: Agent
metadata:
  name: code-reviewer
spec:
  llmRef:
    name: claude-sonnet
  systemPrompt: |
    You are a code review assistant. Review code for:
    - Security issues
    - Performance problems
    - Best practice violations
  mcpServerRefs:
    - name: filesystem
    - name: github
  contactChannelRefs:
    - name: dev-team-slack
```

**Controller:** `internal/controller/agent/` - Validates dependencies exist

---

### 5. **Task** (`task_types.go`, 229 lines)

Represents a conversation with an agent.

**Purpose:** Execute agent workflows, manage conversation state

**Key Fields:**
- `AgentRef`: Which agent to use
- `Input`: Initial message/request
- `Messages`: Full conversation history (user, assistant, tool results)
- `ToolCalls`: List of ToolCall resources created during execution
- `State`: Current execution state (Pending, ReadyForLLM, ToolCallsPending, etc.)
- `SpanContext`: OpenTelemetry trace context (distributed tracing)

**Example:**
```yaml
apiVersion: acp.cprime.dev/v1alpha1
kind: Task
metadata:
  name: review-pr-123
spec:
  agentRef:
    name: code-reviewer
  input: "Review pull request #123 in the main repository"
```

**Controller:** `internal/controller/task/` - **Most complex controller**, orchestrates the LLM conversation loop

**State Machine:**
```
Initializing → Pending → ReadyForLLM → ToolCallsPending → [loop] → FinalAnswer/Error
```

---

### 6. **ToolCall** (`toolcall_types.go`, 148 lines)

Represents a single tool invocation.

**Purpose:** Execute individual tool calls, handle human approvals

**Key Fields:**
- `ToolCallID`: Unique identifier (from LLM response)
- `ToolName`: Name of the tool to invoke
- `Arguments`: JSON arguments for the tool
- `MCPServerRef`: Which MCP server provides this tool
- `ContactChannelRef`: For human approval tools
- `State`: Execution state (Pending, Executing, AwaitingApproval, Completed, etc.)
- `Result`: Tool execution result

**Example (created automatically by Task controller):**
```yaml
apiVersion: acp.cprime.dev/v1alpha1
kind: ToolCall
metadata:
  name: review-pr-123-tool-1
spec:
  taskRef:
    name: review-pr-123
  toolCallID: "call_abc123"
  toolName: "read_file"
  arguments: '{"path": "src/main.go"}'
  mcpServerRef:
    name: filesystem
```

**Controller:** `internal/controller/toolcall/` - Executes tools via MCP or HumanLayer

---

## Controllers (The Control Logic)

**Location:** `acp/internal/controller/`

Each controller follows the **Reconciliation Pattern** from Kubernetes:

1. **Watch** for changes to its CRD
2. **Reconcile** when events occur (create, update, delete)
3. **Update status** to reflect current state
4. **Requeue** if needed for retries or waiting

### Common Controller Structure

Every controller directory has:

```
<resource>/
├── <resource>_controller.go      # Main reconciler with Reconcile() method
├── state_machine.go               # State transition logic (Process method)
├── <resource>_controller_test.go  # Unit tests
└── suite_test.go                  # Test suite setup (Ginkgo)
```

### Controller Details

#### 1. **LLM Controller** (`internal/controller/llm/`)

**Complexity:** Low
**Responsibilities:**
- Validate LLM provider configuration
- Test API connectivity (optional)
- Mark as Ready when valid

**Key Files:**
- `llm_controller.go` - Main reconciler

**State Machine:** Simple validation → Ready

---

#### 2. **MCPServer Controller** (`internal/controller/mcpserver/`)

**Complexity:** Medium
**Responsibilities:**
- Establish connection to MCP server (stdio or HTTP)
- Discover available tools
- Maintain connection pool
- Update status with tool list

**Key Files:**
- `mcpserver_controller.go` - Main reconciler
- `mcpserver_helpers.go` - Connection management
- Uses shared `MCPManager` instance

**State Machine:** Initializing → Connecting → ToolDiscovery → Ready

---

#### 3. **ContactChannel Controller** (`internal/controller/contactchannel/`)

**Complexity:** Low
**Responsibilities:**
- Validate contact channel configuration
- Test connectivity to HumanLayer API
- Mark as Ready when valid

**Key Files:**
- `contactchannel_controller.go` - Main reconciler

**State Machine:** Simple validation → Ready

---

#### 4. **Agent Controller** (`internal/controller/agent/`)

**Complexity:** Low
**Responsibilities:**
- Validate that referenced LLM exists
- Validate that referenced MCP servers exist
- Validate that referenced contact channels exist
- Collect available tools from all MCP servers
- Mark as Ready when all dependencies are ready

**Key Files:**
- `agent_controller.go` - Main reconciler
- `agent_helpers.go` - Dependency validation

**State Machine:** Validating → CollectingTools → Ready

---

#### 5. **Task Controller** (`internal/controller/task/`)

**Complexity:** High (88 Go files in internal/)
**Responsibilities:**
- Orchestrate LLM conversation loop
- Send messages to LLM
- Parse tool calls from LLM responses
- Create ToolCall resources
- Wait for tool call completion
- Add tool results back to conversation
- Detect final answer
- Implement distributed locking (multi-pod coordination)
- Manage context window

**Key Files:**
- `task_controller.go` - Main reconciler (entry point)
- `state_machine.go` - State transition logic (heart of the controller)
- `task_helpers.go` - Helper functions
- `collect_tools.go` - Collect tools from agent's MCP servers
- `types/update_types.go` - Type definitions for updates
- `llm_helpers.go` - LLM interaction helpers
- `toolcall_helpers.go` - ToolCall management helpers
- Internal subdirectory with 80+ files for detailed logic

**State Machine (Detailed):**
```
Initializing
  → Validate agent exists and is ready
  → Initialize message history
  ↓
Pending
  → Wait for agent to be ready
  ↓
ReadyForLLM
  → Acquire distributed lock (if multi-pod)
  → Collect tools from agent
  → Send messages to LLM
  → Parse LLM response
  ↓
ToolCallsPending (if LLM requested tools)
  → Create ToolCall resources
  → Wait for all tool calls to complete
  → Add tool results to conversation
  → Go back to ReadyForLLM
  ↓
FinalAnswer (if LLM returned text response)
  → Extract final answer
  → Mark task as complete
  ↓
Error (if something failed)
  → Record error message
  → Mark task as failed
```

**Distributed Locking:**
- Uses Kubernetes Lease resources
- Prevents concurrent LLM calls for the same task (avoids duplicate tool calls)
- Configurable lease duration (default: 30s)
- Automatic lease renewal during LLM calls
- See: `acp/docs/distributed-locking.md`

---

#### 6. **ToolCall Controller** (`internal/controller/toolcall/`)

**Complexity:** Medium
**Responsibilities:**
- Execute MCP tool calls
- Handle human approval workflows (via HumanLayer)
- Store tool results
- Update status

**Key Files:**
- `toolcall_controller.go` - Main reconciler
- `executor.go` - Tool execution logic
- `state_machine.go` - State transitions

**State Machine:**
```
Pending
  → Determine if human approval required
  ↓
AwaitingApproval (if approval needed)
  → Submit to HumanLayer
  → Poll for approval
  ↓
Executing
  → Invoke MCP tool via MCPManager
  → Store result
  ↓
Completed (success)
  or
Failed (error)
```

**Human Approval Flow:**
1. Check if tool requires approval (configured in agent)
2. Submit approval request to HumanLayer API
3. Poll HumanLayer for approval status
4. If approved → execute tool
5. If denied → mark as failed with denial reason

---

## Internal Packages

**Location:** `acp/internal/`

These packages provide shared functionality used by controllers.

### 1. **llmclient** (`internal/llmclient/`)

**Purpose:** Abstract LLM provider differences behind a common interface

**Key Files:**
- `llm_client.go` (156 lines) - Interface definition
  ```go
  type LLMClient interface {
      Chat(ctx context.Context, messages []schema.Message, tools []Tool, options ...llms.CallOption) (*schema.Message, error)
  }
  ```
- `langchaingo_client.go` (220 lines) - Implementation using `tmc/langchaingo`
- `factory.go` (89 lines) - Factory pattern for creating clients

**Supported Providers:**
- OpenAI (via langchaingo/llms/openai)
- Anthropic (via langchaingo/llms/anthropic)
- Vertex AI (via langchaingo/llms/vertexai)
- Mistral (via langchaingo/llms/mistral)

**Usage Pattern:**
```go
client, err := llmclient.NewLLMClient(ctx, k8sClient, llmResource)
response, err := client.Chat(ctx, messages, tools)
```

---

### 2. **mcpmanager** (`internal/mcpmanager/`)

**Purpose:** Manage connections to MCP servers and execute tool calls

**Key Files:**
- `mcpmanager.go` (450+ lines) - Main manager implementation

**Responsibilities:**
- **Connection Pooling**: Maintain connections to MCP servers
- **Tool Discovery**: Call `tools/list` RPC to discover available tools
- **Tool Invocation**: Execute `tools/call` RPC with arguments
- **Secret Resolution**: Replace `${SECRET_NAME}` in environment variables
- **Error Handling**: Retry logic for transient failures

**Key Methods:**
```go
// Get or create MCP client for a server
GetOrCreateClient(ctx context.Context, mcpServer *acpv1alpha1.MCPServer) (*mcp.StdioMCPClient, error)

// Discover tools from MCP server
ListTools(ctx context.Context, mcpServer *acpv1alpha1.MCPServer) ([]mcp.Tool, error)

// Execute a tool call
CallTool(ctx context.Context, mcpServer *acpv1alpha1.MCPServer, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error)
```

**MCP Client Library:** `github.com/mark3labs/mcp-go`

---

### 3. **humanlayer** (`internal/humanlayer/`)

**Purpose:** Integrate with HumanLayer API for human approvals

**Key Files:**
- `client.go` (280 lines) - Main client implementation
- `hlclient.go` (45 lines) - Client interface
- `mock_hlclient.go` - Mock for testing (generated by gomock)

**Related Package:** `internal/humanlayerapi/` - Auto-generated API client from OpenAPI spec

**Key Methods:**
```go
// Submit a function call for approval
RequestApproval(ctx context.Context, functionCall FunctionCall, contactChannel ContactChannel) (ApprovalResponse, error)

// Check approval status
GetApprovalStatus(ctx context.Context, approvalID string) (ApprovalStatus, error)
```

**HumanLayer Integration:**
- Submit tool call for approval
- Poll for approval status
- Support multiple contact channels (Slack, Email, SMS, WhatsApp)
- Handle approval/denial with reasons

---

### 4. **adapters** (`internal/adapters/`)

**Purpose:** Convert between different tool representations

**Key Files:**
- `mcp_adapter.go` (120 lines) - Converts MCP tools to LLM tool schema

**Why Needed:**
- MCP tools use JSON Schema for parameters
- LLMs expect specific tool formats (OpenAI function calling, Anthropic tools, etc.)
- Adapter translates MCP tool definitions → langchaingo.Tool format

---

### 5. **server** (`internal/server/`)

**Purpose:** Provide REST API for creating agents and tasks

**Technology:** Gin web framework

**Key Files:**
- `server.go` (350 lines) - HTTP handlers
- `runnable.go` (80 lines) - Integration with controller-runtime manager

**Endpoints:**
```
POST   /api/v1/agents        - Create agent
GET    /api/v1/agents/:name  - Get agent
PATCH  /api/v1/agents/:name  - Update agent
DELETE /api/v1/agents/:name  - Delete agent

POST   /api/v1/tasks         - Create task
GET    /api/v1/tasks/:name   - Get task

POST   /webhook/v1beta3/event - GitHub webhook endpoint
```

**Why a REST API?**
- Easier integration for external systems
- Alternative to kubectl for creating resources
- Webhook handler for GitHub integration

**Port:** 8082 (configurable)

---

### 6. **validation** (`internal/validation/`)

**Purpose:** Validate resource specifications beyond basic schema validation

**Key Files:**
- `task_validation.go` - Task-specific validation logic

**Example Validations:**
- Ensure referenced agent exists
- Validate message format
- Check tool call structure

---

### 7. **otel** (`internal/otel/`)

**Purpose:** OpenTelemetry instrumentation setup

**Key Files:**
- `otel.go` (180 lines) - Tracer and meter provider initialization

**Features:**
- **Tracing**: Distributed traces across controllers and LLM calls
- **Metrics**: Custom metrics for task execution, tool calls, etc.
- **Exporters**: OTLP (Tempo, Prometheus)

**Usage:**
```go
// In controllers
ctx, span := otel.Tracer.Start(ctx, "task.reconcile")
defer span.End()

// Propagate trace context across resources
task.Spec.SpanContext = tracing.SerializeSpanContext(ctx)
```

---

## Testing Strategy

**Location:** `acp/test/`

### Test Types

#### 1. **Unit Tests** (Co-located with code)

**Convention:** `*_test.go` files next to implementation

**Framework:** Ginkgo + Gomega

**Example:**
```go
// internal/controller/task/task_controller_test.go
var _ = Describe("TaskController", func() {
    Context("when creating a task", func() {
        It("should initialize the task state", func() {
            // Test logic
        })
    })
})
```

**Run:** `make -C acp test-unit`

---

#### 2. **Integration Tests** (`test/e2e/`)

**Framework:** envtest (embedded etcd + kube-apiserver)

**Key Files:**
- `framework.go` - Test framework setup
- `getting_started/getting_started_test.go` - End-to-end workflow test

**What They Test:**
- Full operator workflow (Agent → Task → ToolCall)
- Controller interactions
- CRD validation
- State transitions

**Run:** `make -C acp test-e2e`

---

#### 3. **Test Utilities** (`test/utils/`)

**Purpose:** Helper functions for creating test resources

**Files:**
- `agent.go` - Create test agents
- `task.go` - Create test tasks
- `llm.go` - Create test LLMs
- `mcpserver.go` - Create test MCP servers
- `toolcall.go` - Create test tool calls
- `contactchannel.go` - Create test contact channels
- `secret.go` - Create test secrets

**Example:**
```go
import "github.com/TJ4519/agentcontrolplane/acp/test/utils"

// Create a test agent
agent := utils.NewAgent("test-agent", llm, mcpServers)
```

---

#### 4. **Mocks** (Generated by gomock)

**Generated Files:** `*_mock.go`

**Example:** `internal/humanlayer/mock_hlclient.go`

**Generate:** `make -C acp mocks`

**Usage in Tests:**
```go
mockHLClient := humanlayer.NewMockHLClient(ctrl)
mockHLClient.EXPECT().RequestApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return(response, nil)
```

---

### Test Coverage

**Run Coverage:**
```bash
make -C acp test-coverage
```

**View Coverage:**
```bash
go tool cover -html=coverage.out
```

---

## Build & Development Workflow

### Prerequisites

- **Go:** 1.23+
- **Docker:** For building images
- **kubectl:** For Kubernetes interaction
- **Kind:** For local Kubernetes cluster (or other K8s)
- **Make:** For build automation

### Common Commands

#### Development

```bash
# Format, vet, and lint code
make -C acp fmt vet lint

# Generate CRDs and deepcopy code (after changing types)
make -C acp manifests generate

# Generate mocks for testing
make -C acp mocks
```

#### Testing

```bash
# Run all tests
make -C acp test

# Run only unit tests
make -C acp test-unit

# Run only e2e tests
make -C acp test-e2e

# Run with coverage
make -C acp test-coverage
```

#### Building

```bash
# Build binary
make -C acp build

# Build Docker image
make -C acp docker-build

# Build and push image
make -C acp docker-buildx
```

#### Local Deployment

```bash
# One-command setup (creates kind cluster, deploys operator, deploys observability)
make setup

# Or step-by-step:
make -C acp deploy-local-kind  # Deploy operator to kind
make -C acp-example deploy     # Deploy observability stack

# Verify deployment
kubectl get deployments -n acp-system
kubectl get agent,task,toolcall,mcpserver,llm

# View logs
kubectl logs -n acp-system -l app.kubernetes.io/name=acp --tail 500 -f
```

#### Cleanup

```bash
# Delete kind cluster
make clean
```

### Development Cycle

**Typical workflow:**

1. **Make changes** to code (controllers, types, etc.)

2. **Update generated code** (if you changed types):
   ```bash
   make -C acp manifests generate
   ```

3. **Format and lint**:
   ```bash
   make -C acp fmt vet lint
   ```

4. **Run tests**:
   ```bash
   make -C acp test
   ```

5. **Build and deploy locally**:
   ```bash
   make -C acp docker-build
   make -C acp deploy-local-kind
   ```

6. **Test manually**:
   ```bash
   # Apply sample resources
   kubectl apply -f acp/config/samples/

   # Watch resources
   kubectl get agent,task,toolcall -w

   # Check logs
   kubectl logs -n acp-system -l app.kubernetes.io/name=acp -f
   ```

7. **Iterate** and repeat

---

## Key Architectural Patterns

### 1. **State Machine Pattern**

All controllers use explicit state machines for reconciliation.

**Example (Task Controller):**

```go
// state_machine.go
func (r *TaskReconciler) Process(ctx context.Context, task *acpv1alpha1.Task) (ctrl.Result, error) {
    switch task.Status.State {
    case "Initializing":
        return r.processInitializing(ctx, task)
    case "Pending":
        return r.processPending(ctx, task)
    case "ReadyForLLM":
        return r.processReadyForLLM(ctx, task)
    case "ToolCallsPending":
        return r.processToolCallsPending(ctx, task)
    case "FinalAnswer":
        return r.processFinalAnswer(ctx, task)
    default:
        return ctrl.Result{}, fmt.Errorf("unknown state: %s", task.Status.State)
    }
}
```

**Benefits:**
- Clear state transitions
- Easy to understand control flow
- Facilitates testing (test each state handler independently)
- Makes debugging easier (know exactly which state failed)

---

### 2. **Reconciliation Loop (Kubernetes Pattern)**

Controllers watch for resource changes and reconcile to desired state.

**Flow:**
1. **Event occurs** (create, update, delete, requeue)
2. **Reconcile() called** with resource name
3. **Fetch current state** from Kubernetes API
4. **Determine desired state** from spec
5. **Make changes** to reach desired state
6. **Update status** to reflect current state
7. **Requeue if needed** (for retries or waiting)

**Example:**
```go
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Fetch the task
    task := &acpv1alpha1.Task{}
    if err := r.Get(ctx, req.NamespacedName, task); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Process based on current state
    result, err := r.Process(ctx, task)

    // Update status
    if err := r.Status().Update(ctx, task); err != nil {
        return ctrl.Result{}, err
    }

    return result, err
}
```

**Requeue Strategies:**
- `ctrl.Result{Requeue: true}` - Requeue immediately
- `ctrl.Result{RequeueAfter: 10*time.Second}` - Requeue after delay
- `ctrl.Result{}` - Don't requeue (wait for next event)

---

### 3. **Dependency Injection**

Controllers receive dependencies via constructors, not globals.

**Example:**
```go
// cmd/main.go
mcpManager := mcpmanager.NewMCPManager(k8sClient, log)

taskReconciler := &controller.TaskReconciler{
    Client:      k8sClient,
    Scheme:      mgr.GetScheme(),
    Recorder:    mgr.GetEventRecorderFor("task-controller"),
    MCPManager:  mcpManager,  // Injected dependency
}
```

**Benefits:**
- Easier testing (inject mocks)
- Clear dependencies
- No hidden global state

---

### 4. **Factory Pattern**

Used for creating LLM clients and other complex objects.

**Example:**
```go
// internal/llmclient/factory.go
func NewLLMClient(ctx context.Context, k8sClient client.Client, llm *acpv1alpha1.LLM) (LLMClient, error) {
    switch llm.Spec.Provider {
    case "openai":
        return newOpenAIClient(ctx, k8sClient, llm)
    case "anthropic":
        return newAnthropicClient(ctx, k8sClient, llm)
    // ...
    default:
        return nil, fmt.Errorf("unsupported provider: %s", llm.Spec.Provider)
    }
}
```

---

### 5. **Distributed Locking**

For multi-pod deployments, prevent concurrent reconciliation of the same resource.

**Implementation:** Kubernetes Lease resources

**Location:** Task controller uses this for LLM calls

**Why Needed:**
- Prevent duplicate LLM calls (expensive, non-idempotent)
- Ensure only one pod processes a task at a time
- Allow safe horizontal scaling

**See:** `acp/docs/distributed-locking.md`

---

### 6. **Interface-Based Design**

Core functionality uses interfaces for flexibility.

**Examples:**

```go
// LLM client interface
type LLMClient interface {
    Chat(ctx context.Context, messages []schema.Message, tools []Tool) (*schema.Message, error)
}

// HumanLayer client interface
type HLClient interface {
    RequestApproval(ctx context.Context, functionCall FunctionCall, contactChannel ContactChannel) (ApprovalResponse, error)
    GetApprovalStatus(ctx context.Context, approvalID string) (ApprovalStatus, error)
}
```

**Benefits:**
- Easy to mock for testing
- Swap implementations without changing callers
- Clear contracts

---

## Essential Files for Deep Dives

### For Understanding the System

| File | Why Read It | Lines |
|------|-------------|-------|
| `acp/cmd/main.go` | Application entry point - see initialization sequence | 328 |
| `acp/api/v1alpha1/task_types.go` | Understand the core Task resource | 229 |
| `acp/internal/controller/task/state_machine.go` | Heart of the task orchestration | ~500 |
| `acp/internal/mcpmanager/mcpmanager.go` | How MCP tools are invoked | 450+ |
| `acp/internal/llmclient/llm_client.go` | LLM abstraction interface | 156 |

### For Understanding Architecture

| Document | What It Explains |
|----------|------------------|
| `acp/docs/kubebuilder-guide.md` | Kubebuilder patterns used |
| `acp/docs/distributed-locking.md` | Multi-pod coordination |
| `acp/docs/getting-started.md` | Quick start guide |
| `acp/docs/crd-reference.md` | Complete CRD field reference |

### For Examples

| Location | What You'll Find |
|----------|------------------|
| `acp/config/samples/` | Example YAML for all CRDs |
| `acp-example/agents/` | Real agent definitions |
| `acp/test/e2e/getting_started/` | End-to-end test (complete workflow) |

---

## Next Steps

### For New Developers

1. **Read this document** (you're here! ✓)

2. **Run locally:**
   ```bash
   make setup
   kubectl get agent,task,toolcall,mcpserver,llm
   ```

3. **Deploy a sample agent:**
   ```bash
   kubectl apply -f acp/config/samples/acp_v1alpha1_llm.yaml
   kubectl apply -f acp/config/samples/acp_v1alpha1_agent.yaml
   kubectl apply -f acp/config/samples/acp_v1alpha1_task.yaml
   ```

4. **Watch it execute:**
   ```bash
   kubectl get task -w
   kubectl logs -n acp-system -l app.kubernetes.io/name=acp -f
   ```

5. **Read the code:**
   - Start with `acp/cmd/main.go`
   - Then `acp/api/v1alpha1/` (understand the data model)
   - Then `acp/internal/controller/task/state_machine.go` (understand orchestration)

6. **Make a change:**
   - Add a log statement in a controller
   - Rebuild and redeploy
   - Verify your change

### For Deep Understanding

**Want to understand task orchestration?**
- Read `internal/controller/task/state_machine.go` in full
- Read `internal/controller/task/llm_helpers.go` (LLM interaction)
- Read `internal/controller/task/toolcall_helpers.go` (ToolCall management)

**Want to understand tool execution?**
- Read `internal/controller/toolcall/executor.go`
- Read `internal/mcpmanager/mcpmanager.go`
- Read `internal/adapters/mcp_adapter.go` (tool format conversion)

**Want to understand LLM integration?**
- Read `internal/llmclient/llm_client.go` (interface)
- Read `internal/llmclient/langchaingo_client.go` (implementation)
- Read `internal/llmclient/factory.go` (provider selection)

**Want to understand observability?**
- Read `internal/otel/otel.go` (OpenTelemetry setup)
- Check `acp-example/grafana/` for dashboards
- Check `acp-example/otel/` for collector config

---

## Glossary

- **ACP**: Agent Control Plane (this project)
- **CRD**: Custom Resource Definition (extends Kubernetes API)
- **MCP**: Model Control Protocol (tool provider standard)
- **LLM**: Large Language Model (GPT-4, Claude, etc.)
- **Reconciliation**: Kubernetes pattern for bringing actual state → desired state
- **Controller**: Component that watches resources and reconciles them
- **envtest**: Testing framework with embedded Kubernetes API server
- **Kubebuilder**: Framework for building Kubernetes operators
- **controller-runtime**: Library for building Kubernetes controllers
- **langchaingo**: Go library for LLM interactions
- **HumanLayer**: Service for human-in-the-loop workflows

---

## FAQ

**Q: Why Kubernetes for LLM orchestration?**
A: Kubernetes provides: declarative configuration, state management, reconciliation loops, RBAC, observability integration, and operational maturity.

**Q: Can I run this without Kubernetes?**
A: No, ACP is a Kubernetes operator. But you can use Kind for local development (single Docker container).

**Q: Which LLM providers are supported?**
A: OpenAI, Anthropic, Vertex AI, Mistral (via langchaingo). Easy to add more.

**Q: What is MCP?**
A: Model Control Protocol - a standard for exposing tools to LLMs. See https://modelcontextprotocol.io/

**Q: How do I add a new MCP server?**
A: Create an MCPServer resource pointing to your MCP server (stdio or HTTP). The operator will discover tools automatically.

**Q: Can I use custom LLM providers?**
A: Yes, extend `internal/llmclient/factory.go` to add new providers.

**Q: How does human approval work?**
A: Via HumanLayer API - submits approval requests to Slack/Email/SMS, polls for response.

**Q: How do I debug controller issues?**
A: Check logs (`kubectl logs -n acp-system ...`), check resource status (`kubectl describe task ...`), use OpenTelemetry traces in Grafana.

**Q: What happens if a pod crashes mid-conversation?**
A: Another pod picks up reconciliation. Conversation state is in Task resource. Distributed locking prevents duplicate work.

**Q: Can tasks call other agents?**
A: Yes! Agents can reference sub-agents. The tool call mechanism works for both MCP tools and sub-agent delegation.

---

## Contributing

See `README.md` and `CLAUDE.md` for contribution guidelines.

**Key principles:**
- Follow existing patterns
- Write tests for new features
- Update CRD documentation if changing types
- Run `make -C acp fmt vet lint test` before committing

---

**End of Codebase Overview**

*This document was generated on 2026-01-20 to provide comprehensive onboarding for developers and AI assistants working with the Agent Control Plane codebase.*
