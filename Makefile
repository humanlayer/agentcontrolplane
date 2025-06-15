# ACP Root Makefile
# Orchestrates commands from acp and acp-example directories

# Define directories
ACP_DIR = acp
EXAMPLE_DIR = acp-example

.PHONY: help build test cluster-up cluster-down build-operator deploy-operator undeploy-operator \
        deploy-samples undeploy-samples deploy-ui deploy-otel undeploy-otel \
        test-operator test-e2e setup-all clean-all \
        acp-% example-%

##@ General Commands

help: ## Display this help information
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Pattern Matching (Run child directory commands directly)

acp-%: ## Run any acp Makefile target: make acp-<target>
	$(MAKE) -C $(ACP_DIR) $*

example-%: ## Run any acp-example Makefile target: make example-<target>
	$(MAKE) -C $(EXAMPLE_DIR) $*

##@ Composite Commands

build: acp-build ## Build acp components


branchname := $(shell git branch --show-current)
dirname := $(shell basename ${PWD})
clustername := acp-$(branchname)

setup: ## Create isolated kind cluster for this branch and set up dependencies
	@echo "BRANCH: ${branchname}"
	@echo "DIRNAME: ${dirname}"
	@echo "CLUSTER: ${clustername}"
	
	# Generate dynamic ports and store in .ports.env
	@apiport=$$(./hack/find_free_port.sh 11000 11100); \
	acpport=$$(./hack/find_free_port.sh 11100 11200); \
	echo "KIND_APISERVER_PORT=$$apiport" > .ports.env; \
	echo "ACP_SERVER_PORT=$$acpport" >> .ports.env; \
	echo "Generated ports:"; \
	cat .ports.env
	
	# Create kind cluster with dynamic port configuration
	@if ! kind get clusters | grep -q "^${clustername}$$"; then \
		echo "Creating kind cluster: ${clustername}"; \
		. .ports.env && \
		mkdir -p acp/tmp && \
		export KIND_APISERVER_PORT && export ACP_SERVER_PORT && \
		npx envsubst < acp-example/kind/kind-config.template.yaml > acp/tmp/kind-config.yaml && \
		if grep -q "hostPort: *$$" acp/tmp/kind-config.yaml; then \
			echo "ERROR: Empty hostPort found in generated config. Variables not substituted properly."; \
			echo "Generated config:"; \
			cat acp/tmp/kind-config.yaml; \
			echo "Environment variables:"; \
			echo "KIND_APISERVER_PORT=$$KIND_APISERVER_PORT"; \
			echo "ACP_SERVER_PORT=$$ACP_SERVER_PORT"; \
			exit 1; \
		fi && \
		kind create cluster --name ${clustername} --config acp/tmp/kind-config.yaml; \
	else \
		echo "Kind cluster already exists: ${clustername}"; \
	fi
	
	# Export kubeconfig to worktree-local location
	@mkdir -p .kube
	@kind export kubeconfig --name ${clustername} --kubeconfig .kube/config
	@echo "Kubeconfig exported to .kube/config"
	
	
	# Create secrets with API keys
	@if [ -n "${OPENAI_API_KEY:-}" ]; then \
		KUBECONFIG=.kube/config kubectl create secret generic openai --from-literal=OPENAI_API_KEY=${OPENAI_API_KEY} --dry-run=client -o yaml | KUBECONFIG=.kube/config kubectl apply -f -; \
	fi
	@if [ -n "${ANTHROPIC_API_KEY:-}" ]; then \
		KUBECONFIG=.kube/config kubectl create secret generic anthropic --from-literal=ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY} --dry-run=client -o yaml | KUBECONFIG=.kube/config kubectl apply -f -; \
	fi
	@if [ -n "${HUMANLAYER_API_KEY:-}" ]; then \
		KUBECONFIG=.kube/config kubectl create secret generic humanlayer --from-literal=HUMANLAYER_API_KEY=${HUMANLAYER_API_KEY} --dry-run=client -o yaml | KUBECONFIG=.kube/config kubectl apply -f -; \
	fi
	
	# Set up acp dependencies
	$(MAKE) -C $(ACP_DIR) mocks deps
	
	# Deploy ACP controller
	@echo "Deploying ACP controller..."
	$(MAKE) -C $(ACP_DIR) deploy-local-kind
	
	# Wait for controller to be ready
	@echo "Waiting for ACP controller to be ready..."
	@KUBECONFIG=.kube/config timeout 120 bash -c 'until kubectl get deployment acp-controller-manager -n default >/dev/null 2>&1; do echo "Waiting for deployment to be created..."; sleep 2; done'
	@KUBECONFIG=.kube/config kubectl wait --for=condition=available --timeout=120s deployment/acp-controller-manager -n default
	@echo "✅ ACP controller is ready!"
	
	@echo ""
	@echo "✅ Setup complete! To use the isolated cluster:"
	@echo "   source .envrc    # or use direnv for automatic loading"
	@echo "   kubectl get nodes"
	@echo "   kubectl get pods -n default  # Check ACP controller status" 


teardown: ## Teardown the isolated kind cluster and clean up
	@echo "BRANCH: ${branchname}"
	@echo "CLUSTER: ${clustername}"
	
	# Delete kind cluster
	@if kind get clusters | grep -q "^${clustername}$$"; then \
		echo "Deleting kind cluster: ${clustername}"; \
		kind delete cluster --name ${clustername}; \
	else \
		echo "Kind cluster '${clustername}' not found"; \
	fi
	
	# Clean up local files
	@if [ -f .kube/config ]; then \
		echo "Removing local kubeconfig"; \
		rm -f .kube/config; \
		rmdir .kube 2>/dev/null || true; \
	fi
	
	@echo "✅ Teardown complete!"

check: 
	# $(MAKE) -C $(ACP_DIR) fmt vet lint test generate 

test: acp-test ## Run tests for acp components

check-keys-set: acp-check-keys-set

##@ Cluster Management

cluster-up: ## Create the Kind cluster
	$(MAKE) -C $(EXAMPLE_DIR) kind-up

cluster-down: ## Delete the Kind cluster
	$(MAKE) -C $(EXAMPLE_DIR) kind-down

##@ Operator Management

build-operator: ## Build the ACP operator binary
	$(MAKE) -C $(ACP_DIR) build

deploy-operator: ## Deploy the ACP operator to the local Kind cluster
	$(MAKE) -C $(ACP_DIR) deploy-local-kind

undeploy-operator: ## Undeploy the operator and remove CRDs
	$(MAKE) -C $(ACP_DIR) undeploy
	$(MAKE) -C $(ACP_DIR) uninstall

##@ Resource Management

deploy-samples: ## Deploy sample resources to the cluster
	$(MAKE) -C $(ACP_DIR) deploy-samples

undeploy-samples: ## Remove sample resources
	$(MAKE) -C $(ACP_DIR) undeploy-samples

show-samples: ## Show status of sample resources
	$(MAKE) -C $(ACP_DIR) show-samples

watch-samples: ## Watch status of sample resources with continuous updates
	$(MAKE) -C $(ACP_DIR) watch-samples

##@ UI and Observability

deploy-ui: ## Deploy the ACP UI
	$(MAKE) -C $(EXAMPLE_DIR) ui-deploy

deploy-otel: ## Deploy the observability stack (Prometheus, OpenTelemetry, Grafana, Tempo, Loki)
	$(MAKE) -C $(EXAMPLE_DIR) otel-stack

undeploy-otel: ## Remove the observability stack
	$(MAKE) -C $(EXAMPLE_DIR) otel-stack-down

otel-access: ## Display access instructions for monitoring stack
	$(MAKE) -C $(EXAMPLE_DIR) otel-access

##@ Testing

test-operator: ## Run unit tests for the operator
	$(MAKE) -C $(ACP_DIR) test

test-e2e: ## Run end-to-end tests (requires a running cluster)
	$(MAKE) -C $(ACP_DIR) test-e2e

##@ All-in-One Commands

setup-all: cluster-up deploy-operator deploy-samples deploy-ui deploy-otel ## Set up the entire environment
	@echo "Complete environment setup finished successfully"

clean-all: undeploy-samples undeploy-operator undeploy-otel cluster-down ## Clean up everything
	@echo "Complete environment cleanup finished successfully"

.PHONY: githooks
githooks:
	ln -s ${PWD}/hack/git_pre_push.sh ${PWD}/.git/hooks/pre-push