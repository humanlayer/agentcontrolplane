package mcpserver

import (
	"context"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/internal/mcpmanager"
)

const (
	StatusError   = "Error"
	StatusPending = "Pending"
	StatusReady   = "Ready"
)

// MCPClientFactory defines the interface for creating MCP clients
type MCPClientFactory interface {
	CreateStdioClient(ctx context.Context, command string, env []string, args ...string) (mcpclient.MCPClient, error)
	CreateHTTPClient(ctx context.Context, url string) (mcpclient.MCPClient, error)
}

// EnvVarProcessor defines the interface for processing environment variables
type EnvVarProcessor interface {
	ProcessEnvVars(ctx context.Context, envVars []acp.EnvVar, namespace string) ([]string, error)
}

// MCPServerManagerInterface defines the interface for MCP server management
type MCPServerManagerInterface interface {
	ConnectServer(ctx context.Context, mcpServer *acp.MCPServer) error
	GetTools(serverName string) ([]acp.MCPTool, bool)
	GetConnection(serverName string) (*mcpmanager.MCPConnection, bool)
	DisconnectServer(serverName string)
	CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (string, error)
	FindServerForTool(fullToolName string) (serverName string, toolName string, found bool)
	Close()
}

// MCPServerStatusUpdate defines consistent status update parameters
type MCPServerStatusUpdate struct {
	Connected    bool
	Status       string
	StatusDetail string
	Tools        []acp.MCPTool
	Error        string
	EventType    string
	EventReason  string
	EventMessage string
}

// Default factory implementations
type defaultMCPClientFactory struct{}

func (f *defaultMCPClientFactory) CreateStdioClient(ctx context.Context, command string, env []string, args ...string) (mcpclient.MCPClient, error) {
	return mcpclient.NewStdioMCPClient(command, env, args...)
}

func (f *defaultMCPClientFactory) CreateHTTPClient(ctx context.Context, url string) (mcpclient.MCPClient, error) {
	return mcpclient.NewSSEMCPClient(url)
}

type defaultEnvVarProcessor struct {
	client client.Client
}

func (p *defaultEnvVarProcessor) ProcessEnvVars(ctx context.Context, envVars []acp.EnvVar, namespace string) ([]string, error) {
	return processEnvVars(ctx, p.client, envVars, namespace)
}

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=mcpservers/status,verbs=get;update;patch

// MCPServerReconciler reconciles a MCPServer object
type MCPServerReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	MCPManager      MCPServerManagerInterface
	clientFactory   MCPClientFactory
	envVarProcessor EnvVarProcessor
	stateMachine    *StateMachine
}

// getMCPServer retrieves an MCPServer by namespaced name
func (r *MCPServerReconciler) getMCPServer(ctx context.Context, namespacedName client.ObjectKey) (*acp.MCPServer, error) {
	var mcpServer acp.MCPServer
	if err := r.Get(ctx, namespacedName, &mcpServer); err != nil {
		return nil, err
	}
	return &mcpServer, nil
}

// Reconcile processes the MCPServer resource using StateMachine
func (r *MCPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	mcpServer, err := r.getMCPServer(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.FromContext(ctx).V(1).Info("Starting reconciliation", "name", mcpServer.Name)

	// Ensure StateMachine is initialized
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}

	// Delegate to StateMachine
	return r.stateMachine.Process(ctx, mcpServer)
}

// ensureStateMachine initializes the state machine if not already initialized
func (r *MCPServerReconciler) ensureStateMachine() {
	if r.stateMachine != nil {
		return
	}

	// Initialize dependencies if not provided
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManagerWithClient(r.Client)
	}
	if r.clientFactory == nil {
		r.clientFactory = &defaultMCPClientFactory{}
	}
	if r.envVarProcessor == nil {
		r.envVarProcessor = &defaultEnvVarProcessor{client: r.Client}
	}

	// Create StateMachine
	r.stateMachine = NewStateMachine(r.Client, r.recorder, r.MCPManager, r.clientFactory, r.envVarProcessor)
}

// SetupWithManager sets up the controller with the Manager using factory defaults
func (r *MCPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("mcpserver-controller")

	// Initialize factories with defaults if not already set
	if r.clientFactory == nil {
		r.clientFactory = &defaultMCPClientFactory{}
	}

	if r.envVarProcessor == nil {
		r.envVarProcessor = &defaultEnvVarProcessor{client: r.Client}
	}

	// Initialize the MCP manager if not already set
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManagerWithClient(r.Client)
	}

	// Initialize StateMachine
	r.stateMachine = NewStateMachine(r.Client, r.recorder, r.MCPManager, r.clientFactory, r.envVarProcessor)

	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.MCPServer{}).
		Complete(r)
}
