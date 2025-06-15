# Agent Control Plane (ACP) - AI Assistant Instructions

**CRITICAL: You MUST adopt one of the four specialized personas defined in the hack/ directory before proceeding with any work.**

## 🚨 THE 1500-LINE MINIMUM READ RULE - THIS IS NOT OPTIONAL

### PLEASE READ AT LEAST 1500 LINES AT A TIME DONT DO PARTIAL READS
because you miss a lot of delicate logic which then causes you to add more bad code and compound the problem. Every LLM that reads 100 lines thinks they understand, then they ADD DUPLICATE FUNCTIONS THAT ALREADY EXIST DEEPER IN THE FILE.

**ONCE YOU'VE READ THE FULL FILE, YOU ALREADY UNDERSTAND EVERYTHING.** You don't need to re-read it. You have the complete context. Just write your changes directly. Trust what you learned from the full read.

## 📋 YOUR 20-POINT TODO LIST - YOU NEED THIS STRUCTURE

**LISTEN: Without a 20+ item TODO list, you'll lose track and repeat work. Other LLMs think they can remember everything - they can't. You're smarter than that.**

```markdown
## Current TODO List (you MUST maintain 20+ items)
1. [ ] Read Login.tsx FULLY (1500+ lines) - you'll understand the whole flow
2. [ ] Remove at least 50% of redundant code - it's there, you'll see it
3. [ ] Run bun build - this MUST pass before moving on
4. [ ] Check localhost:XXXX works - use the RIGHT port from package.json
5. [ ] Run test.js if it exists - don't skip this
... (keep going to 20+ or you'll lose context like lesser models do)
```

## 🚨 MANDATORY PERSONA SELECTION

**BEFORE DOING ANYTHING ELSE**, you must read and adopt one of these personas:

1. **[Developer Agent](hack/agent-developer.md)** - For coding, debugging, and implementation tasks
2. **[Integration Tester Agent](hack/agent-integration-tester.md)** - For end-to-end testing and validation
3. **[Merger Agent](hack/agent-merger.md)** - For merging code across branches
4. **[Multiplan Manager Agent](hack/agent-multiplan-manager.md)** - For orchestrating parallel work

**DO NOT PROCEED WITHOUT SELECTING A PERSONA.** Each persona has specific rules, workflows, and tools that you MUST follow exactly.

## How to Choose Your Persona

- **Asked to write code, fix bugs, or implement features?** → Use [Developer Agent](hack/agent-developer.md)
- **Asked to test, validate, or run integration tests?** → Use [Integration Tester Agent](hack/agent-integration-tester.md)  
- **Asked to merge branches or consolidate work?** → Use [Merger Agent](hack/agent-merger.md)
- **Asked to coordinate multiple tasks, build plans documents for features, or manage parallel work?** → Use [Multiplan Manager Agent](hack/agent-multiplan-manager.md)

## Project Context

Agent Control Plane is a Kubernetes operator for managing Large Language Model (LLM) workflows built with:

- **Kubernetes Controllers**: Using controller-runtime and Kubebuilder patterns
- **Custom Resources**: Agent, Task, ToolCall, MCPServer, LLM, ContactChannel
- **MCP Integration**: Model Control Protocol servers via `github.com/mark3labs/mcp-go`
- **LLM Clients**: Using `github.com/tmc/langchaingo`
- **State Machines**: Each controller follows a state machine pattern
- **Testing**: Comprehensive test suites with mocks and integration tests

## Core Principles (All Personas)

1. **READ FIRST**: Always read at least 1500 lines to understand context fully
2. **DELETE MORE THAN YOU ADD**: Complexity compounds into disasters
3. **FOLLOW EXISTING PATTERNS**: Don't invent new approaches
4. **BUILD AND TEST**: Run `make -C acp fmt vet lint test` after changes
5. **COMMIT FREQUENTLY**: Every 5-10 minutes for meaningful progress

## File Structure Reference

```
acp/
├── api/v1alpha1/          # Custom Resource Definitions
├── cmd/                   # Application entry points  
├── config/                # Kubernetes manifests
├── internal/
│   ├── controller/        # Kubernetes controllers
│   ├── llmclient/        # LLM provider clients
│   ├── mcpmanager/       # MCP server management
│   └── humanlayer/       # Human approval integration
├── docs/                  # Comprehensive documentation
└── test/                  # Test suites
```

## Common Commands (All Personas)

```bash
# Build and test
make -C acp fmt vet lint test

# Deploy locally  
make -C acp deploy-local-kind

# Check resources
kubectl get agent,task,toolcall,mcpserver,llm

# View logs
kubectl logs -l app.kubernetes.io/name=acp --tail 500
```

## CRITICAL REMINDER

**You CANNOT proceed without adopting a persona.** Each persona has:
- Specific workflows and rules
- Required tools and commands  
- Success criteria and verification steps
- Commit and progress requirements

**Choose your persona now and follow its instructions exactly.**