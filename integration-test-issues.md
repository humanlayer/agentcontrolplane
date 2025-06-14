# Integration Test Issues

## Issue 1: Human as Tool workflow - External Call ID not populated

**Description**: When testing the Human as Tool workflow, the ToolCall resource shows `External Call ID: ` (empty) and the human contact request does not appear in the pending human contacts list via the humanlayer client.

**Steps to reproduce**:
1. Create a ContactChannel with email type and valid HumanLayer API key
2. Create an Agent with humanContactChannels referencing the ContactChannel
3. Create a Task that triggers human contact (e.g., "Ask an expert what the fastest animal on the planet is")
4. The ToolCall gets created with phase "AwaitingHumanInput" but External Call ID is empty
5. Running `go run hack/humanlayer_client.go -o list-pending-human-contacts` times out or does not show the request

**Expected behavior**: 
- ToolCall should have a populated External Call ID
- The human contact request should appear in the pending list
- Should be able to respond to the request using the humanlayer client

**Actual behavior**:
- ToolCall External Call ID is empty
- Request does not appear in pending human contacts
- Cannot respond to the request
- HumanLayer API calls appear to timeout

**Resources involved**:
- ContactChannel: `human-expert` (Ready and validated)
- Agent: `agent-with-human-tool` (Ready)
- Task: `human-expert-task-test` (ToolCallsPending)
- ToolCall: `human-expert-task-test-r3i5dcg-tc-01` (AwaitingHumanInput)

**Controller logs**: No errors visible in controller logs, task keeps reconciling in ToolCallsPending phase. No toolcall-controller logs found for the human contact creation.

**Impact**: Prevents testing of the Human as Tool feature end-to-end

**Fix needed**: Investigation into why External Call ID is not being set and why the human contact request is not being created in HumanLayer API. May be related to ToolCall controller not processing HumanContact type tool calls properly.

---

## Issue 2: Human Approval workflow fails with invalid email addresses

**Description**: When testing human approval workflow with test email addresses (e.g., test@example.com), the approval request fails with "400 Bad Request".

**Steps to reproduce**:
1. Create a ContactChannel with email type using a test email address (test@example.com)
2. Create an MCPServer with approvalContactChannel referencing the ContactChannel
3. Create an Agent that uses the MCPServer
4. Create a Task that triggers a tool call requiring approval
5. The ToolCall fails with "ErrorRequestingHumanApproval" phase and error "failed to request approval: 400 Bad Request"

**Expected behavior**: 
- Should either succeed with test email or provide a clearer error message about invalid email

**Actual behavior**:
- ToolCall fails with 400 Bad Request
- No clear indication that the email address is invalid

**Resources involved**:
- ContactChannel: `approval-channel` 
- MCPServer: `fetch` (with approvalContactChannel)
- ToolCall: Shows `ErrorRequestingHumanApproval` phase

**Impact**: Prevents testing human approval workflow with test data. Requires valid email addresses for testing.

**Fix needed**: Either support test email addresses for development/testing, or provide clearer error messages when email validation fails.

---

## Issue 3: Documentation contains outdated API reference

**Description**: The getting-started.md documentation references swapi.dev API which is broken/unreliable.

**Steps to reproduce**:
1. Follow getting-started guide exactly as written
2. Try to fetch data from swapi.dev/api/people/2

**Expected behavior**: 
- API calls should work as documented

**Actual behavior**:
- swapi.dev API is unreliable/broken

**Fix applied**: Updated getting-started.md to use lotrapi.co instead of swapi.dev for more reliable testing.

**Impact**: Low - documentation issue only

**Status**: FIXED - Updated all references from swapi.dev to lotrapi.co API endpoints.

---

## Summary

### Working Features ✅
1. **Basic Agent and Task creation** - Works perfectly
2. **MCP Tools integration** - Works perfectly 
3. **Anthropic LLM integration** - Works perfectly
4. **Human Approval workflow** - Works when using valid email addresses
5. **Sub-Agent Delegation** - Works perfectly

### Issues Found ❌
1. **Human as Tool workflow** - External Call ID not populated, requests not created in HumanLayer API
2. **Human Approval with test emails** - Fails with 400 Bad Request for test email addresses

### Critical Issues for Development Team
- **Issue 1** is the most critical as it completely blocks the Human as Tool feature
- **Issue 2** impacts testing/development workflow but has a workaround (use valid emails)

The core ACP functionality works very well, with only the human interaction features having issues.