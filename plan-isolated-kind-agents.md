# Isolated Kind Clusters for Developer Agents

## Objective
Update the developer agent infrastructure so each agent runs in its own isolated kind cluster, preventing conflicts and ensuring true parallel development.

## Current State
- All agents share the same kind cluster
- Potential for resource conflicts and namespace collisions
- Agents can interfere with each other's deployments
- Single point of failure if cluster has issues

## Proposed Solution
Each worktree gets its own kind cluster with isolated kubeconfig, allowing complete independence between agents.

## Implementation Plan

### 1. Update make setup target

Add cluster creation to the `setup` target in your Makefile (which is called by `create_worktree.sh`):

```makefile
# In your Makefile
setup:
	# Generate unique cluster name from current git branch
	CLUSTER_NAME=acp-$(shell git branch --show-current)
	# Find free ports
	HOST_PORT_8082=$$(bash hack/find_free_port.sh 10000 11000)
	HOST_PORT_9092=$$(bash hack/find_free_port.sh 10000 11000)
	HOST_PORT_13000=$$(bash hack/find_free_port.sh 10000 11000)
	# Generate kind-config.yaml from template
	npx envsubst < acp-example/kind/kind-config.template.yaml > acp-example/kind/kind-config.yaml
	# Create kind cluster with unique name
	kind create cluster --name $$CLUSTER_NAME --config acp-example/kind/kind-config.yaml
	# Export kubeconfig to worktree-local location
	mkdir -p .kube
	kind get kubeconfig --name $$CLUSTER_NAME > .kube/config
	# Create .envrc for direnv (optional but helpful)
	echo 'export KUBECONFIG="$(pwd)/.kube/config"' > .envrc
	echo 'export KIND_CLUSTER_NAME="$$CLUSTER_NAME"' >> .envrc
	# Continue with other setup steps as needed
```

### 2. Update Makefiles

Modify `acp/Makefile` to use local kubeconfig:
```makefile
# Use KUBECONFIG from environment, or local .kube/config if it exists, otherwise default to ~/.kube/config
KUBECONFIG ?= $(if $(wildcard $(PWD)/.kube/config),$(PWD)/.kube/config,$(HOME)/.kube/config)
export KUBECONFIG

# Add cluster name for kind operations
KIND_CLUSTER_NAME ?= $(shell basename $(PWD) | sed 's/agentcontrolplane_//')

# Update deploy-local-kind target
deploy-local-kind: manifests kustomize
	# Ensure cluster exists
	@if ! kind get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		echo "Cluster $(KIND_CLUSTER_NAME) not found. Please run setup first."; \
		exit 1; \
	fi
	# Continue with existing deploy logic...
```

### 3. Port Management

To avoid port conflicts, implement dynamic port allocation when generating the kind-config.yaml file.

Example bash function for dynamic port allocation:

```bash
# Find a free port in a given range
find_free_port() {
    local start_port=$1
    local end_port=$2
    for ((port=start_port; port<=end_port; port++)); do
        if ! lsof -i :$port >/dev/null 2>&1; then
            echo $port
            return 0
        fi
    done
    echo "No free port found in range $start_port-$end_port" >&2
    return 1
}

# Example usage:
HOST_PORT_8082=$(find_free_port 10000 11000)
HOST_PORT_9092=$(find_free_port 10000 11000)
HOST_PORT_13000=$(find_free_port 10000 11000)

# Then substitute these into your kind-config.yaml template
npx envsubst < kind-config.template.yaml > kind-config.yaml
```

### 4. Update hack/agent-developer.md

Add cluster management to developer workflow:
```markdown
### Step 0: Verify Your Cluster
```bash
# Check your cluster is running
kubectl cluster-info

# Verify you're using the right context
kubectl config current-context

# Should show: kind-acp-[your-branch-name]

# if it doesn't check with me and I'll confirm if you're in the right cluster

```

### 5. Cleanup Process

Update cleanup_coding_workers.sh:
```bash
# Function to cleanup kind cluster
cleanup_kind_cluster() {
    local branch_name=$1
    local cluster_name="acp-${branch_name}"
    
    if kind get clusters | grep -q "^${cluster_name}$"; then
        log "Deleting kind cluster: $cluster_name"
        kind delete cluster --name "$cluster_name"
    else
        info "Kind cluster not found: $cluster_name"
    fi
}
```

### 7. Resource Considerations

Each minimal kind cluster for the ACP controller uses approximately:
- 500-800 MB RAM (single control plane node, no heavy workloads)
- 0.5-1 CPU core (mostly idle except during deployments)
- 2-3 GB disk space (container images and etcd data)

For a machine running 7 agents, this means:
- 3.5-5.6 GB RAM total
- 3.5-7 CPU cores (but mostly idle)
- 14-21 GB disk space

The ACP controller itself is lightweight - it's just watching CRDs and managing resources. We're not running databases, heavy applications, or multiple replicas.

Consider adding resource limits or warnings.

## Benefits

1. **Complete Isolation**: No interference between agents
2. **Parallel Testing**: Each agent can run full integration tests
3. **Clean Failures**: If one cluster fails, others continue
4. **Easy Debugging**: Each agent has its own logs and resources
5. **True Parallel Development**: No resource contention

## Risks and Mitigations

1. **Resource Usage**: Multiple clusters still add up
   - Mitigation: Add resource checks before creating clusters
   - Consider single-node clusters with reduced resource requests
   - Option to share clusters for truly lightweight tasks

2. **Port Conflicts**: Multiple clusters need different host ports
   - Mitigation: Dynamic port allocation
   - Use cluster DNS instead of host ports where possible

3. **Complexity**: More moving parts to manage
   - Mitigation: Good automation and error handling
   - Clear documentation and troubleshooting guides

## Alternative Approaches

1. **Namespace Isolation**: Use single cluster with namespace per agent
   - Pros: Less resource usage
   - Cons: Less isolation, potential for conflicts

2. **Virtual Clusters**: Use vcluster for lightweight isolation
   - Pros: Better resource usage than full kind clusters
   - Cons: Additional complexity, less mature

3. **Remote Clusters**: Use cloud-based dev clusters
   - Pros: No local resource constraints
   - Cons: Network latency, cost, complexity

## Implementation Steps

9. add limits and requests to the manager pod and ensure it works - note it will need a decent ceiling to run mcp servers on stdio
1. Create proof of concept with single worktree
2. Update create_worktree.sh with cluster creation
3. Modify Makefiles for local kubeconfig
4. Update cleanup scripts
5. Test with multiple parallel agents
6. Document resource requirements
7. Add resource limit checks
8. Create troubleshooting guide

## Success Criteria

- Each agent can deploy without affecting others
- Clusters are automatically created and cleaned up
- Resource usage is reasonable (system doesn't crash)
- Existing workflows continue to work
- Clear error messages when resources are insufficient