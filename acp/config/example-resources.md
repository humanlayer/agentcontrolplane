# Example Application Resources

This document describes the sample resources provided in the `config/samples` directory and explains how they match our implemented CRDs for the example application.

Deploy them with 

```
kustomize build samples | kubectl apply -f -
```

---

## MCPServer Resource with Secret References

[./samples/acp_mcpserver_with_secrets.yaml](./samples/acp_mcpserver_with_secrets.yaml)

**Resource:** `MCPServer`  
**API Version:** `acp.humanlayer.dev/v1alpha1`  
**Kind:** `MCPServer`

**Sample File:** `config/samples/acp_mcpserver_with_secrets.yaml`

**Key Fields:**

- **transport:** The connection type (e.g., `"stdio"`)
- **command:** The command to run for stdio MCP servers
- **args:** Arguments to pass to the command
- **env:** Environment variables to set for the server
  - Can include direct values:
    ```yaml
    - name: DIRECT_VALUE
      value: "some-direct-value"
    ```
  - Can reference secrets:
    ```yaml
    - name: SECRET_VALUE
      valueFrom:
        secretKeyRef:
          name: mcp-credentials
          key: api-key
    ```
- **resources:** Resource requests and limits (optional)
  ```yaml
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi
  ```

**Required Secret:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mcp-credentials
  namespace: default
type: Opaque
data:
  api-key: c2VjcmV0LWFwaS1rZXktdmFsdWU=  # base64 encoded value of "secret-api-key-value"
```

**Benefits of Secret References:**
- Keeps sensitive information out of resource definitions
- Follows Kubernetes patterns for secret management
- Allows for centralized management of credentials

---

## LLM

[./samples/acp_llm.yaml](./samples/acp_llm.yaml)

**Resource:** `LLM`  
**API Version:** `acp.humanlayer.dev/v1alpha1`  
**Kind:** `LLM`

**Sample File:** `config/samples/acp_llm.yaml`

**Key Fields:**

- **provider:** One of: `"openai"`, `"anthropic"`, `"mistral"`, `"google"`, or `"vertex"`
- **apiKeyFrom:**
  - References a secret (e.g. secret name: `openai`)
  - Key: e.g. `OPENAI_API_KEY`
- **maxTokens:** e.g. `1000`

_Note:_ Ensure that the referenced secret exists for your chosen provider:
- OpenAI: Create a secret named `openai` with the key `OPENAI_API_KEY`
- Anthropic: Create a secret named `anthropic` with the key `ANTHROPIC_API_KEY`
- Mistral: Create a secret named `mistral` with the key `MISTRAL_API_KEY` 
- Google: Create a secret named `google` with the key `GOOGLE_API_KEY`
- Vertex: Create a secret named `vertex` with the key `service-account-json` containing the service account credentials

See the samples file for examples of all supported providers.

---

## Agent

[./samples/acp_agent.yaml](./samples/acp_agent.yaml)

**Resource:** `Agent`  
**API Version:** `acp.humanlayer.dev/v1alpha1`  
**Kind:** `Agent`

**Sample File:** `config/samples/acp_agent.yaml`

**Key Fields:**

- **llmRef:**
  - Must refer to the LLM resource (e.g. `gpt-4o`)
- **tools:**
  - A list of tool references (e.g. one tool with name `"add"`)
- **system:**
  - A system prompt (e.g. instructions for a calculator agent)

---

## Tool

[./samples/acp_tool.yaml](./samples/acp_tool.yaml)

**Resource:** `Tool`  
**API Version:** `acp.humanlayer.dev/v1alpha1`  
**Kind:** `Tool`

**Sample File:** `config/samples/acp_tool.yaml`

**Key Fields:**

- **toolType:** e.g. `"function"`
- **name:** e.g. `"add"`
- **description:** A short description (e.g. "Add two numbers")
- **arguments:**
  - A JSON schema defining the expected input arguments. For instance, properties "a" and "b" of type number.
- **execute:**
  - Configuration for how the tool is executed (e.g. use a builtin function called `"add"`)

---

## Task

[./samples/acp_task.yaml](./samples/acp_task.yaml)

**Resource:** `Task`  
**API Version:** `acp.humanlayer.dev/v1alpha1`  
**Kind:** `Task`

**Sample File:** `config/samples/acp_task.yaml`

**Key Fields:**

- **agentRef:**
  - References an existing Agent (e.g. `"calculator-agent"`)
- **userMessage:**
  - The task prompt or request (e.g. `"What is 2 + 2?"`)
  - Used for single-turn queries
  - Cannot be used with `contextWindow`

**OR**

- **contextWindow:**
  - Array of messages representing an initial conversation
  - Used for multi-turn conversations or continuing previous conversations
  - Must contain at least one user message (can be at any position)
  - Cannot be used with `userMessage`
  - If no system message is included, the agent's system message will be prepended
  - Tool calls or tool messages in the contextWindow are NOT executed - they're treated purely as conversation history
  - The last message can be any valid role (user, assistant, tool) - the controller will send the entire context to the LLM
  - The last user message (not necessarily the last message) will be used for the UserMsgPreview field

**Sample with contextWindow:** `config/samples/acp_v1alpha1_context_window_task.yaml`

---

## Additional Notes

- **Secrets:** Make sure all required secrets are created in your cluster:
  - For LLMs: create the secret referenced by `apiKeyFrom.secretKeyRef` (e.g., secret `openai` with key `OPENAI_API_KEY`)
  - For MCPServers: create any secrets referenced in `env[].valueFrom.secretKeyRef` (e.g., secret `mcp-credentials` with key `api-key`)

- **CRDs & Controllers:** Before applying these sample files, ensure that the CRDs are installed (use `make manifests install`) and that the controllers are deployed (`make deploy`).


- **Secret Permissions:** The ACP controller needs permission to read secrets in the namespaces where your resources are deployed. The default RBAC rules in `config/rbac/role.yaml` include these permissions.

These sample files now match our example application design, where:

- An LLM (`gpt-4o`) is defined with the required API key reference.
- A Calculator Agent (`calculator-agent`) uses that LLM and has a system prompt suited for mathematical operations.
- A Tool (`add`) is implemented to perform addition.
- A Task (`calculate-sum`) uses the Agent to process an arithmetic question.
- MCPServers can be configured with environment variables from both direct values and secret references.

Ensure that your cluster includes the necessary prerequisites (such as all required secrets) so that the status fields eventually show "ready" once the controllers have reconciled the objects.

Happy deploying\!
