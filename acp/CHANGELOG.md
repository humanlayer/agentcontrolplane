### v0.5.0 (April 11, 2025) 

This release simplifies tool management, enhances traceability, and strengthens human-in-the-loop execution patterns.

#### Breaking Changes

- **Removed the `Tool` CRD** and its controller. Agents no longer reference standalone `Tool` resources. Tool functionality is now:
  - Discovered dynamically via MCP servers, or
  - Handled via ContactChannels

- **Renamed `TaskRunToolCall` → `ToolCall`**
  - CRD `kind` changed: `TaskRunToolCall` → `ToolCall`
  - CRD `plural` changed: `taskruntoolcalls` → `toolcalls`
  - Labels, manifests, code references, and controller names updated accordingly.

- **Removed `tools` field from `AgentSpec`**
  - Agents no longer declare tool refs explicitly. Tools are resolved via MCPServers or as humanContacts.

#### Features

- **OpenTelemetry tracing for `ToolCall` execution**
  - Each `ToolCall` now has its own root span.
  - Child spans are created for:
    - MCP tool execution
    - Human approval workflows
  - Spans propagate through the `Task` context tree and are visible in Tempo.

- **Improved Human Approval Workflow**
  - Approval steps (via Slack/email) now generate structured OTel spans.
  - Final approval/rejection is captured and surfaced directly in `ToolCall.status.result`.

#### Other Changes

- Docs and examples updated to reflect:
  - Removal of `Tool` CRD
  - Transition from `TaskRunToolCall` to `ToolCall`
  - Refined `kubectl` usage for `toolcall` resources

---

> **Migration Notes**
>
> Before upgrading:
>
> - ❌ Delete all `Tool` resources (`kubectl delete tool --all`)
> - ✏️ Remove the `tools` array from `AgentSpec`
> - 🔄 Rename any `TaskRunToolCall` objects or manifests to use the `ToolCall` schema
> - 🔐 Ensure all `secretKeyRef` fields conform to the new `SecretKeyRef` structure

---





### v0.4.0

- Breaking Changes:
  - rename all CRDs, etc from kubechain to agentcontrolplane / ACP

be careful with this one y'all

### v0.2.0 (March 26, 2025)

Breaking Changes:
- `Task` and `TaskRun` have been combined into a single resource called `Task`. This greatly simplies the API and onboarding documentation.
- Removed experimental `externalAPI` and `builtin` tool types

Features:
- Added support for [multiple LLM providers](../README.md#using-other-language-models)
  - Anthropic Claude support
  - Google AI support
  - Mistral AI support
  - Vertex AI support
- Better handling when a [human rejects a proposed tool call](../README.md#incorporating-human-approval)


Fixes:
- Fixed a bug where a multi-turn tool-calling workflow could result in the wrong tool results being sent to the LLM
- Improved error handling for LLM API failures
  - Distinct handling of retriable vs non-retriable errors
  - Better error reporting in task status

### v0.1.13 (March 25, 2025)

Features:
- Added support for tool approval via [HumanLayer](https://humanlayer.dev) contact channels

Changes:
- Renamed ContactChannel CRD fields for better clarity
  - Changed `channelType` to `type`
  - Changed `slackConfig` to `slack`
  - Changed `emailConfig` to `email`
- Enhanced TaskRunToolCall status tracking
  - Added `externalCallID` field for tracking external service calls
  - Added new phases: `ErrorRequestingHumanApproval`, `ReadyToExecuteApprovedTool`, `ToolCallRejected`

### v0.1.12 (March 24, 2025)

Features:
- Added OpenTelemetry tracing support
  - Spans for LLM requests with context window and tool metrics
  - Parent spans for TaskRun lifecycle tracking
  - Completion spans for terminal states
  - Status and error propagation to spans

Changes:
- Refactored TaskRun phase transitions and improved phase transition logging
- Enhanced testing infrastructure
  - Improved TaskRun and TaskRunToolCall test suites
  - Added test utilities for common setup patterns

### v0.1.11 (March 24, 2025)

Features:
- Added support for contact channels with Slack and email integration
  - New ContactChannel CRD with validation fields, printer columns, and status tracking
  - Support for API key authentication
  - Email message customization options
  - Channel configuration validation

Fixes:
- Updated MCPServer CRD to support approval channels for tool execution

### v0.1.10 (March 24, 2025)

Features:
- Added MCP (Model Control Protocol) server support
  - New MCPServer CRD for tool execution
  - Support for stdio and http transport protocols
  - Tool discovery and validation
  - Resource configuration options
- Enhanced task run statuses and tracking
- Improved agent validation for MCP server access
- Added status details fields across CRDs for better observability

Infrastructure:
- Increased resource limits for controller
  - CPU: 1000m (up from 500m)
  - Memory: 512Mi (up from 128Mi)
- Updated base resource requests
  - CPU: 100m (up from 10m)
  - Memory: 256Mi (up from 64Mi)
