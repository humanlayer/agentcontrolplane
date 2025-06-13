package task

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"

	"github.com/humanlayer/agentcontrolplane/acp/internal/adapters"
	"github.com/humanlayer/agentcontrolplane/acp/internal/humanlayer"
	"github.com/humanlayer/agentcontrolplane/acp/internal/llmclient"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
	"go.opentelemetry.io/otel/trace"
)

const (
	DefaultRequeueDelay  = 5 * time.Second
	HumanLayerAPITimeout = 10 * time.Second
	LLMRequestTimeout    = 30 * time.Second
)

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=tasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=toolcalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=llms,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete

// MCPManager defines the interface for managing MCP servers and tools
type MCPManager interface {
	GetTools(serverName string) ([]acp.MCPTool, bool)
}

// LLMClientFactory defines the interface for creating LLM clients
type LLMClientFactory interface {
	CreateClient(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error)
}

// HumanLayerClientFactory defines the interface for creating HumanLayer clients
type HumanLayerClientFactory interface {
	NewClient(baseURL string) (humanlayer.HumanLayerClientWrapper, error)
}

// ToolAdapter defines the interface for converting tools between different formats
type ToolAdapter interface {
	ConvertMCPTools(tools []acp.MCPTool, serverName string) []llmclient.Tool
	ConvertContactChannels(channels []acp.ContactChannel) []llmclient.Tool
	ConvertSubAgents(agents []acp.Agent) []llmclient.Tool
}

// defaultLLMClientFactory provides the default implementation of LLMClientFactory
type defaultLLMClientFactory struct{}

func (f *defaultLLMClientFactory) CreateClient(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
	return llmclient.NewLLMClient(ctx, llm, apiKey)
}

// defaultHumanLayerClientFactory provides the default implementation of HumanLayerClientFactory
type defaultHumanLayerClientFactory struct{}

func (f *defaultHumanLayerClientFactory) NewClient(baseURL string) (humanlayer.HumanLayerClientWrapper, error) {
	clientFactory, err := humanlayer.NewHumanLayerClientFactory(baseURL)
	if err != nil {
		return nil, err
	}
	return clientFactory.NewHumanLayerClient(), nil
}

// defaultToolAdapter provides the default implementation of ToolAdapter
type defaultToolAdapter struct{}

func (a *defaultToolAdapter) ConvertMCPTools(tools []acp.MCPTool, serverName string) []llmclient.Tool {
	return adapters.ConvertMCPToolsToLLMClientTools(tools, serverName)
}

func (a *defaultToolAdapter) ConvertContactChannels(channels []acp.ContactChannel) []llmclient.Tool {
	tools := make([]llmclient.Tool, 0, len(channels))
	for _, channel := range channels {
		tool := llmclient.ToolFromContactChannel(channel)
		if tool != nil {
			tools = append(tools, *tool)
		}
	}
	return tools
}

func (a *defaultToolAdapter) ConvertSubAgents(agents []acp.Agent) []llmclient.Tool {
	tools := make([]llmclient.Tool, 0, len(agents))
	for _, agent := range agents {
		delegateTool := llmclient.Tool{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        "delegate_to_agent__" + agent.Name,
				Description: agent.Spec.Description,
				Parameters: llmclient.ToolFunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"message": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"message"},
				},
			},
			ACPToolType: acp.ToolTypeDelegateToAgent,
		}
		tools = append(tools, delegateTool)
	}
	return tools
}

// TaskStatusUpdate defines the parameters for updating task status
type TaskStatusUpdate struct {
	Ready        bool
	Status       acp.TaskStatusType
	Phase        acp.TaskPhase
	StatusDetail string
	Error        string
	EventType    string
	EventReason  string
	EventMessage string
}

// mockLLMClientFactory provides a mock implementation for testing
type mockLLMClientFactory struct {
	createFunc func(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error)
}

func (f *mockLLMClientFactory) CreateClient(ctx context.Context, llm acp.LLM, apiKey string) (llmclient.LLMClient, error) {
	return f.createFunc(ctx, llm, apiKey)
}

// TaskReconciler reconciles a Task object
type TaskReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	recorder                record.EventRecorder
	llmClientFactory        LLMClientFactory
	MCPManager              MCPManager
	humanLayerClientFactory HumanLayerClientFactory
	toolAdapter             ToolAdapter
	Tracer                  trace.Tracer
	stateMachine            *StateMachine
}

// validateTaskAndAgent checks if the agent exists and is ready
func (r *TaskReconciler) contextWithTaskSpan(ctx context.Context, task *acp.Task) context.Context {
	if task.Status.SpanContext == nil || task.Status.SpanContext.TraceID == "" || task.Status.SpanContext.SpanID == "" {
		return ctx // no root yet or invalid context
	}

	sc, err := reconstructSpanContext(task.Status.SpanContext.TraceID, task.Status.SpanContext.SpanID)
	if err != nil {
		log.FromContext(ctx).V(1).Info("Failed to reconstruct span context", "error", err)
		return ctx
	}

	return trace.ContextWithSpanContext(ctx, sc)
}

// collectTools collects all tools from the agent's MCP servers
func (r *TaskReconciler) collectTools(ctx context.Context, agent *acp.Agent) []llmclient.Tool {
	logger := log.FromContext(ctx)
	tools := make([]llmclient.Tool, 0)

	// Iterate through each MCP server directly to maintain server-tool association
	for _, serverRef := range agent.Spec.MCPServers {
		mcpTools, found := r.MCPManager.GetTools(serverRef.Name)
		if !found {
			logger.Info("Server not found or has no tools", "server", serverRef.Name)
			continue
		}
		// Use the injected tool adapter to convert tools
		tools = append(tools, r.toolAdapter.ConvertMCPTools(mcpTools, serverRef.Name)...)
		logger.Info("Added MCP server tools", "server", serverRef.Name, "toolCount", len(mcpTools))
	}

	// Collect and convert HumanContactChannel tools
	contactChannels := make([]acp.ContactChannel, 0, len(agent.Status.ValidHumanContactChannels))
	for _, validChannel := range agent.Status.ValidHumanContactChannels {
		channel := &acp.ContactChannel{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: validChannel.Name}, channel); err != nil {
			logger.Error(err, "Failed to get ContactChannel", "name", validChannel.Name)
			continue
		}
		contactChannels = append(contactChannels, *channel)
	}
	tools = append(tools, r.toolAdapter.ConvertContactChannels(contactChannels)...)
	logger.Info("Added contact channel tools", "count", len(contactChannels))

	// Collect and convert sub-agent tools
	subAgents := make([]acp.Agent, 0, len(agent.Spec.SubAgents))
	for _, subAgentRef := range agent.Spec.SubAgents {
		subAgent := &acp.Agent{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: subAgentRef.Name}, subAgent); err != nil {
			logger.Error(err, "Failed to get sub-agent", "name", subAgentRef.Name)
			continue
		}
		subAgents = append(subAgents, *subAgent)
	}
	tools = append(tools, r.toolAdapter.ConvertSubAgents(subAgents)...)
	logger.Info("Added sub-agent delegate tools", "count", len(subAgents))

	return tools
}

// Reconcile validates the task's agent reference and sends the prompt to the LLM.
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	task, err := r.getTask(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.FromContext(ctx).V(1).Info("Starting reconciliation", "name", task.Name)

	// Ensure StateMachine is initialized
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}

	// Attach task span context for tracing (except for initialization)
	if task.Status.Phase != "" && task.Status.SpanContext != nil {
		ctx = r.contextWithTaskSpan(ctx, task)
	}

	// Delegate to StateMachine
	return r.stateMachine.Process(ctx, task)
}

// getTask retrieves a task by namespaced name
func (r *TaskReconciler) getTask(ctx context.Context, namespacedName client.ObjectKey) (*acp.Task, error) {
	var task acp.Task
	if err := r.Get(ctx, namespacedName, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// generateRandomString returns a securely generated random string
func generateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret), nil
}

func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("task-controller")
	if r.llmClientFactory == nil {
		r.llmClientFactory = &defaultLLMClientFactory{}
	}

	// Initialize MCPManager if not already set
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManager()
	}

	// Initialize HumanLayerClientFactory if not already set
	if r.humanLayerClientFactory == nil {
		r.humanLayerClientFactory = &defaultHumanLayerClientFactory{}
	}

	// Initialize ToolAdapter if not already set
	if r.toolAdapter == nil {
		r.toolAdapter = &defaultToolAdapter{}
	}

	// Initialize StateMachine
	r.ensureStateMachine()

	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.Task{}).
		Complete(r)
}

// ensureStateMachine initializes the state machine if not already initialized
func (r *TaskReconciler) ensureStateMachine() {
	if r.stateMachine != nil {
		return
	}

	r.stateMachine = NewStateMachine(
		r.Client,
		r.recorder,
		r.llmClientFactory,
		r.MCPManager,
		r.humanLayerClientFactory,
		r.toolAdapter,
		r.Tracer,
	)
}
