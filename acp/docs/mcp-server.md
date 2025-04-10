# Model Control Protocol (MCP) Servers

## Overview

The Model Control Protocol (MCP) is a standard interface for connecting AI/LLM agents with external tools and capabilities. In ACP, MCP servers are defined using the `MCPServer` custom resource type.

## MCPServer Resource

The `MCPServer` resource defines how to connect to an MCP server, which can provide tools to LLM agents.

### Spec Fields

| Field | Description | Example |
|-------|-------------|---------|
| `transport` | Communication method (`stdio` or `http`) | `stdio` |
| `command` | Command to run (for stdio transport) | `"/usr/local/bin/mcp-server"` |
| `args` | Arguments for the command | `["--verbose"]` |
| `env` | Environment variables | See below |
| `url` | URL for HTTP transport | `"https://mcp-server.example.com"` |
| `resources` | CPU/memory requests and limits | See below |

### Environment Variables

The `env` field supports two ways to specify environment variables:

1. **Direct Values**:
   ```yaml
   env:
     - name: DEBUG
       value: "true"
   ```

2. **Secret References**:
   ```yaml
   env:
     - name: API_KEY
       valueFrom:
         secretKeyRef:
           name: mcp-credentials
           key: api-key
   ```

Secret references allow you to securely provide sensitive information like API keys and credentials to your MCP server without hardcoding them in the resource definition.

### Resource Requirements

You can specify resource requests and limits for the MCP server process:

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 200m
    memory: 256Mi
```

## Example: MCP Server with Secret Reference

```yaml
apiVersion: acp.humanlayer.dev/v1alpha1 
kind: MCPServer
metadata:
  name: fetch-mcp-server
  namespace: default
spec:
  transport: stdio
  command: "uvx"
  args: ["mcp-server-fetch"]
  env:
    - name: LOG_LEVEL
      value: "debug"
    - name: API_KEY
      valueFrom:
        secretKeyRef:
          name: fetch-api-credentials
          key: api-key
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi
```

You'll need to create the corresponding Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: fetch-api-credentials
  namespace: default
type: Opaque
data:
  api-key: <base64-encoded-api-key>
```

## Status Fields

| Field | Description |
|-------|-------------|
| `connected` | Whether the MCP server is connected |
| `status` | Current status (`Ready`, `Error`, or `Pending`) |
| `statusDetail` | Detailed status message |
| `tools` | List of tools provided by the MCP server |

## Using MCP-provided Tools

Tools discovered from an MCP server can be used in your Agents by referencing them by name. The controller manages making these tools available to the LLM.

See the `config/samples/` directory for complete examples.