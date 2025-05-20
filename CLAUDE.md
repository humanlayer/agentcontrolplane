# Agent Control Plane (ACP)

This document provides context and guidance for working with the Agent Control Plane codebase. When you're working on this project, refer to this guidance to ensure your work aligns with project conventions and patterns.

## Project Context

Agent Control Plane is a Kubernetes operator for managing Large Language Model (LLM) workflows. The project provides:

- Custom resources for LLM configurations and agent definitions
- A controller-based architecture for managing resources
- Integration with Model Control Protocol (MCP) servers using the `github.com/mark3labs/mcp-go` library
- LLM client implementations using `github.com/tmc/langchaingo`

Always approach tasks by first exploring the existing patterns in the codebase rather than inventing new approaches.

## Documentation

The codebase includes comprehensive documentation in the `acp/docs/` directory. **Always consult these docs first** when working on specific components:

- [MCP Server Guide](acp/docs/mcp-server.md) - For work involving Model Control Protocol servers
- [LLM Providers Guide](acp/docs/llm-providers.md) - For integrating with LLM providers
- [CRD Reference](acp/docs/crd-reference.md) - For understanding custom resource definitions
- [Kubebuilder Guide](acp/docs/kubebuilder-guide.md) - For work on the Kubernetes operators
- [Debugging Guide](acp/docs/debugging-guide.md) - For debugging the operator locally
- [Gin Servers Guide](acp/docs/gin-servers.md) - For work on API servers using Gin

## Building and Development

The project uses three Makefiles, each with a specific purpose:

- `Makefile` - Root-level commands for cross-component workflows (use this for high-level operations)
- `acp/Makefile` - Developer commands for the ACP operator (use this for day-to-day development)
- `acp-example/Makefile` - Commands for the example/development deployment environment

When you need to perform a build or development operation, read the appropriate Makefile to understand available commands:

```bash
# For ACP operator development
make -C acp

# For example environment setup 
make -C acp-example

# For high-level operations
make
```

## Code Organization

The project follows a Kubebuilder-based directory structure:

- `acp/api/v1alpha1/` - Custom Resource Definitions
- `acp/cmd/` - Application entry points
- `acp/config/` - Kubernetes manifests and configurations
- `acp/internal/` - Internal implementation code
  - `controller/` - Kubernetes controllers
  - `adapters/` - Integration adapters
  - `llmclient/` - LLM provider clients
  - `server/` - API server implementations

Always use the correct relative paths from the root when referencing files.

## Task-Specific Guidance

When tasked with modifying or extending the codebase, first determine what component you'll be working with, then refer to the appropriate guidance below.

### Kubernetes CRDs and Controllers

If working with Custom Resource Definitions or controllers:

1. **First read** the [Kubebuilder Guide](acp/docs/kubebuilder-guide.md) to understand the project's approach.
2. Look at existing CRD definitions in `acp/api/v1alpha1/` for patterns to follow.
3. Remember to regenerate manifests and code after changes by reading the acp Makefile to find the appropriate commands for generation.

Follow the Status/Phase pattern for resources with complex state machines, separating resource health (Status) from workflow progress (Phase). This pattern is consistent across the codebase.

### LLM Integration

When working on LLM provider integrations:

1. **First read** the [LLM Providers Guide](acp/docs/llm-providers.md) for current provider implementations.
2. Study the implementation in `acp/internal/llmclient/` before making changes.
3. Follow established credential management patterns using Kubernetes secrets.

### API Server Development

When working on API servers:

1. **First read** the [Gin Servers Guide](acp/docs/gin-servers.md) for best practices.
2. Keep HTTP handlers thin, with business logic moved to separate functions.
3. Use the context pattern consistently for propagating request context.

### Testing Approach

When writing tests, match the style of existing tests based on component type:

- **Controller Tests**: Follow state-based testing with clear state transition tests
- **Client Tests**: Use mock interfaces and dependency injection
- **End-to-End Tests**: For cross-component functionality in `acp/test/e2e/`

For controllers, organization is particularly important:
- Name contexts clearly with the state transition (e.g., `"Ready:Pending -> Ready:Running"`)
- Test each state transition independently
- Focus on behavior, not implementation details

## Code Style Guidelines

### Go

The project follows idiomatic Go patterns:

- Format code with `gofmt` (run `make -C acp fmt`)
- Use meaningful error handling with context propagation
- Implement dependency injection for testability
- Document public functions with godoc comments

### Kubernetes Patterns

When implementing Kubernetes controllers or resources:

- Separate controller logic from business logic
- Use Status and Phase fields consistently as described in controller docs
- Follow resource ownership patterns for garbage collection
- Add proper RBAC annotations to controllers before generating manifests

## When in Doubt

If you're unsure of the best approach when working on a specific component:

1. First check relevant documentation in `acp/docs/`
2. Look at similar, existing implementations in the codebase
3. Follow established patterns rather than inventing new ones
4. Ask the developer for feedback!

Remember that this is a Kubernetes operator project built with Kubebuilder. Stay consistent with Kubernetes API conventions and controller-runtime patterns.
