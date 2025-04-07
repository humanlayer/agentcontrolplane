# Custom Resource Definition (CRD) Reference

This document provides reference information for the Custom Resource Definitions (CRDs) used in Kubechain.

## MCPServer

The MCPServer CRD represents a Model Control Protocol server instance.

### Spec Fields

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `transport` | string | Connection type: "stdio" or "http" | Yes |
| `command` | string | Command to run (for stdio transport) | No |
| `args` | []string | Arguments for the command | No |
| `env` | []EnvVar | Environment variables | No |
| `url` | string | URL (for http transport) | No |
| `resources` | ResourceRequirements | CPU/memory resource requests/limits | No |

#### EnvVar

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `name` | string | Environment variable name | Yes |
| `value` | string | Direct value for the environment variable | No* |
| `valueFrom` | EnvVarSource | Source for the environment variable value | No* |

*Either `value` or `valueFrom` must be specified.

#### EnvVarSource

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `secretKeyRef` | SecretKeySelector | Reference to a secret | No |

#### SecretKeySelector

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `name` | string | Name of the secret | Yes |
| `key` | string | Key within the secret | Yes |

#### ResourceRequirements

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `limits` | ResourceList | Maximum resource limits | No |
| `requests` | ResourceList | Minimum resource requests | No |

ResourceList is a map of ResourceName to resource.Quantity (e.g., `cpu: 100m`).

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `connected` | boolean | Whether the MCP server is connected |
| `status` | string | Current status: "Ready", "Error", or "Pending" |
| `statusDetail` | string | Detailed status message |
| `tools` | []MCPTool | List of tools provided by the MCP server |

#### MCPTool

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `name` | string | Name of the tool | Yes |
| `description` | string | Description of the tool | No |
| `inputSchema` | runtime.RawExtension | JSON schema for the tool's input parameters | No |

## LLM

The LLM CRD represents a Large Language Model configuration.

### Spec Fields

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `provider` | string | LLM provider (one of: "openai", "anthropic", "mistral", "google", "vertex") | Yes |
| `apiKeyFrom` | SecretKeySelector | Secret containing the API key | Yes |
| `baseConfig` | object | Common configuration options across providers (model, temperature, etc.) | No |
| `providerConfig` | object | Provider-specific configuration (openaiConfig, anthropicConfig, vertexConfig, etc.) | No |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `ready` | boolean | Whether the LLM is ready to use |
| `status` | string | Current status: "Ready", "Error", or "Pending" |
| `statusDetail` | string | Detailed status message |

## Agent

The Agent CRD represents an LLM agent with specific tools and capabilities.

### Spec Fields

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `llmRef` | NameRef | Reference to an LLM resource | Yes |
| `systemPrompt` | string | System prompt for the agent | No |
| `tools` | []ToolRef | Tools available to the agent | No |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `ready` | boolean | Whether the agent is ready to use |
| `status` | string | Current status: "Ready", "Error", or "Pending" |
| `statusDetail` | string | Detailed status message |

## Tool

The Tool CRD represents a capability that can be used by an agent.

### Spec Fields

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `toolType` | string | Type of tool | Yes |
| `name` | string | Name of the tool | Yes |
| `description` | string | Description of the tool | No |
| `arguments` | object | JSON schema for tool arguments | No |
| `execute` | object | Execution configuration | Yes |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `ready` | boolean | Whether the tool is ready to use |
| `status` | string | Current status: "Ready", "Error", or "Pending" |
| `statusDetail` | string | Detailed status message |

## Task

The Task CRD represents a task instance.

### Spec Fields

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `agentRef` | LocalObjectReference | Reference to the agent to execute the task | Yes |
| `userMessage` | string | Message to send to the agent | Yes |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase of execution |
| `phaseHistory` | []PhaseTransition | History of phase transitions |
| `contextWindow` | []Message | The conversation context |