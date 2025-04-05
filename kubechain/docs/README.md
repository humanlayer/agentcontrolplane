# Kubechain Documentation

## Overview

Kubechain is a Kubernetes operator for managing Large Language Model (LLM) workflows. It provides custom resources for:

- LLM configurations
- Agent definitions
- Tools and capabilities
- Task execution
- MCP servers for tool integration

## Guides

- [MCP Server Guide](./mcp-server.md) - Working with Model Control Protocol servers
- [LLM Providers Guide](./llm-providers.md) - Configuring different LLM providers (OpenAI, Anthropic, Mistral, Google, Vertex)
- [CRD Reference](./crd-reference.md) - Complete reference for all Custom Resource Definitions
- [Kubebuilder Guide](./kubebuilder-guide.md) - How to develop with Kubebuilder in this project
- [Debugging Guide](./debugging-guide.md) - How to debug the operator locally with VS Code

## Example Resources

See the [Example Resources](../config/example-resources.md) document for details on the sample resources provided in the `config/samples` directory.

## Sample Files

For concrete examples, check the sample YAML files in the [`config/samples/`](../config/samples/) directory:

- [`kubechain_v1alpha1_mcpserver.yaml`](../config/samples/kubechain_v1alpha1_mcpserver.yaml) - Basic MCP server
- [`kubechain_v1alpha1_mcpserver_with_secrets.yaml`](../config/samples/kubechain_v1alpha1_mcpserver_with_secrets.yaml) - MCP server with secret references
- [`kubechain_v1alpha1_llm.yaml`](../config/samples/kubechain_v1alpha1_llm.yaml) - LLM configuration
- [`kubechain_v1alpha1_agent.yaml`](../config/samples/kubechain_v1alpha1_agent.yaml) - Agent definition
- [`kubechain_v1alpha1_tool.yaml`](../config/samples/kubechain_v1alpha1_tool.yaml) - Tool definition
- [`kubechain_v1alpha1_task.yaml`](../config/samples/kubechain_v1alpha1_task.yaml) - Task execution

## Development

For general development documentation, see the [CONTRIBUTING](../CONTRIBUTING.md) guide.

For instructions on working with Kubebuilder to extend the Kubernetes API (adding new CRDs, controllers, etc.), refer to the [Kubebuilder Guide](./kubebuilder-guide.md).