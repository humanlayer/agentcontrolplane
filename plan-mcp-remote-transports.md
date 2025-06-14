# MCP Remote Transport Support Plan

## Objective
Add support for remote MCP servers using SSE (Server-Sent Events) and Streamable HTTP (sHTTP) transports to enable connecting to cloud-hosted MCP servers.

**Note:**
- **SSE support must be maintained** even though it is deprecated as of 2024-11-05, because it is still widely used in the field.
- **Streamable HTTP (sHTTP) is being added as the new standard** for remote MCP server communication (as of 2025-03-26).

## Background
Currently, the ACP only supports:
- **stdio transport**: For local command execution
- **http transport**: Using SSE (NewSSEMCPClient) - but this is the legacy transport

The MCP specification has evolved to include:
- **SSE transport** (deprecated as of 2024-11-05, but still widely used; support must be maintained)
- **Streamable HTTP (sHTTP) transport** (current standard as of 2025-03-26; support must be added)

## Current Implementation Analysis

### Transport Support
- `mcpmanager.go`: Currently only creates `NewStdioMCPClient` or `NewSSEMCPClient`
- `mcpserver_types.go`: Transport enum only allows "stdio" or "http"
- The "http" transport currently uses the legacy SSE client

### Key Components
1. **MCPServerManager** (`internal/mcpmanager/mcpmanager.go`)
   - Line 134-154: Transport selection logic
   - Line 148: Uses `NewSSEMCPClient` for http transport

2. **MCPServer CRD** (`api/v1alpha1/mcpserver_types.go`)
   - Line 12-14: Transport validation enum

## Implementation Plan

### Phase 1: Update Transport Enum and Types

1. **Update MCPServerSpec Transport Field**
   ```go
   // Transport specifies the transport type for the MCP server
   // +kubebuilder:validation:Enum=stdio;http;sse;streamable-http
   Transport string `json:"transport"`
   ```
   **Clarification:**
   - `http` and `sse` both map to the SSE client for backward compatibility.
   - `streamable-http` (or `shttp`) maps to the new Streamable HTTP client.

2. **Add Optional Headers Field**
   ```go
   // Headers are optional HTTP headers for remote transports
   // +optional
   Headers map[string]string `json:"headers,omitempty"`
   
   // SessionID for streamable-http transport session resumption
   // +optional
   SessionID string `json:"sessionId,omitempty"`
   ```

### Phase 2: Update MCPManager

1. **Check mcp-go Library Version**
   - Verify if `NewStreamableHttpClient` exists in current version
   - If not, check for updates or implement wrapper

2. **Update ConnectServer Method**
   ```go
   switch mcpServer.Spec.Transport {
   case "stdio":
       // Existing stdio logic
   case "http", "sse":
       // Use SSE client (legacy but widely supported)
       mcpClient, err = mcpclient.NewSSEMCPClient(mcpServer.Spec.URL)
   case "streamable-http":
       // Check if available in mcp-go
       if streamableClientExists {
           mcpClient, err = mcpclient.NewStreamableHttpClient(mcpServer.Spec.URL)
       } else {
           // Fallback or error
       }
   }
   ```
   **Clarification:**
   - Backward compatibility is maintained by mapping both `http` and `sse` to the SSE client.
   - New deployments should use `streamable-http` for the new standard.

3. **Handle Session Management**
   - Store session IDs in status for resumption
   - Add reconnection logic with session ID headers

### Phase 3: Add Security and Configuration

1. **TLS/HTTPS Enforcement**
   - Validate URLs use HTTPS for production
   - Add option to allow HTTP for development

2. **Authentication Support**
   - Support Bearer tokens in headers
   - Support API keys in headers

3. **Timeout Configuration**
   - Add configurable timeouts for remote connections
   - Handle long-running operations with SSE streams

### Phase 4: Update Controller Logic

1. **Connection Health Checks**
   - Implement periodic health checks for remote servers
   - Handle network interruptions gracefully

2. **Status Updates**
   - Add connection type to status
   - Show session information for streamable-http

### Phase 5: Testing

1. **Unit Tests**
   - Mock remote MCP servers
   - Test all transport types

2. **Integration Tests**
   - Test with real SSE servers
   - Test with streamable-http servers
   - Test failover scenarios

3. **Example Configurations**
   ```yaml
   # SSE Transport Example
   apiVersion: acp.humanlayer.dev/v1alpha1
   kind: MCPServer
   metadata:
     name: remote-sse-server
   spec:
     transport: sse
     url: https://mcp.example.com/sse
     headers:
       - name: Pragma
         value: "no-cache"
       - name: Authorization
         valueFrom:
           secretKeyRef:
             name: SSEServer
             key: SSE_AUTHORIZATION_HEADER
   
   # Streamable HTTP Example  
   apiVersion: acp.humanlayer.dev/v1alpha1
   kind: MCPServer
   metadata:
     name: remote-streamable-server
   spec:
     transport: streamable-http
     url: https://mcp.example.com/mcp
     headers:
       - name: Authorization
         valueFrom:
           secretKeyRef:
             name: StreamableServer
             key: STREAMABLE_AUTHORIZATION_HEADER
   ```

## Implementation Steps

1. **Research mcp-go Library**
   - Check go.mod for current version
   - Look for streamable HTTP support
   - Determine if library update needed

2. **Update CRD Schema**
   - Add new transport types
   - Add headers and session fields
   - Regenerate CRDs

3. **Implement Transport Logic**
   - Update mcpmanager.go
   - Add proper error handling
   - Implement reconnection logic

4. **Add Examples and Documentation**
   - Document new transport types
   - Provide configuration examples
   - Update README

## Risks and Mitigations

1. **Library Support**
   - Risk: mcp-go may not support streamable-http yet
   - Mitigation: Implement wrapper or contribute upstream

2. **Backward Compatibility**
   - Risk: Breaking existing "http" transport users
   - Mitigation: Keep "http" as alias for "sse"; maintain SSE support as long as needed

3. **Security**
   - Risk: Exposing credentials in CRDs
   - Mitigation: Use secretKeyRef for sensitive headers

## Success Criteria

- Can connect to remote MCP servers using SSE transport (even though deprecated)
- Can connect to remote MCP servers using streamable-http transport (new standard)
- examples added to the acp/docs/getting-started.md guide
- IF AVAILABLE - examples tested in a new acp/tests/e2e/ package
- Graceful handling of connection failures and reconnection
- Proper session management for streamable-http
- All existing stdio and http transports continue working

## Commit Strategy
- Commit after updating CRD schemas
- Commit after implementing each transport type
- Commit after adding tests
- Commit after documentation updates

Remember to adopt the hack/agent-developer.md persona and follow the Dan Abramov methodology throughout this implementation.

## References and Further Reading

- **mcp-proxy Example Configs**  
  [sparfenyuk/mcp-proxy GitHub Repository](https://github.com/sparfenyuk/mcp-proxy)  
  [config_example.json](https://github.com/sparfenyuk/mcp-proxy/blob/main/config_example.json)  
  Example JSON structure for multiple MCP servers, each with a transport type, command/URL, headers, and arguments.

- **Level Up Coding: MCP Server and Client with SSE & The New Streamable HTTP**  
  [Level Up Coding - MCP Server and Client with SSE & The New Streamable HTTP](https://levelup.gitconnected.com/mcp-server-and-client-with-sse-the-new-streamable-http-d860850d9d9d)  
  Code and configuration examples for both SSE and Streamable HTTP transports, including endpoint and client/server logic.

- **Official MCP Specification**  
  [Model Context Protocol Specification (2025-03-26)](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports)  
  Details on transport types, protocol requirements, and best practices for MCP clients and servers.