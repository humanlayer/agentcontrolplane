# Agent Control Plane

The Agent Control Plane (ACP) is a Kubernetes operator for managing Large Language Model (LLM) workflows.

## Description

ACP provides Custom Resource Definitions (CRDs) for defining and managing LLM-based agents, tools, and tasks within a Kubernetes cluster. It enables you to define reusable components for AI/LLM workflows, including the Model Control Protocol (MCP) integration for tool extensibility.

## Getting Started

### Prerequisites
- go version v1.23.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/acp:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/acp:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/acp:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/acp/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
kubebuilder edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

## Contributing

### Development Workflow

The project uses [Kubebuilder](https://book.kubebuilder.io/) for scaffolding Kubernetes resources and controllers. If you're extending the API or adding new resource types, please refer to our [Kubebuilder Guide](./docs/kubebuilder-guide.md) for detailed instructions on:

- Adding new custom resources
- Updating existing resources
- Working with controllers
- Generating RBAC permissions
- Following best practices

### Make Targets

Run `make help` for more information on all potential `make` targets. Common targets include:

- `make build` - Build the manager binary
- `make manifests` - Generate WebhookConfiguration, ClusterRole, and CustomResourceDefinition objects
- `make generate` - Generate code (DeepCopy methods)
- `make test` - Run tests
- `make docker-build` - Build the Docker image

### Resources

- [Kubebuilder Book](https://book.kubebuilder.io/introduction.html) - Official Kubebuilder documentation
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime) - Library for building controllers
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md) - Standards for Kubernetes API design

## Documentation

- [MCP Server Guide](./docs/mcp-server.md) - Detailed guide for working with MCP servers
- [CRD Reference](./docs/crd-reference.md) - Complete reference for all Custom Resource Definitions
- [Kubebuilder Guide](./docs/kubebuilder-guide.md) - How to develop with Kubebuilder in this project

## Resource Types

### MCPServer

Model Control Protocol (MCP) servers provide a way to extend the functionality of LLMs with custom tools. The MCPServer resource supports:

- **Transport:** 
  - `stdio`: Communicate with an MCP server via standard I/O
  - `http`: Communicate with an MCP server via HTTP

- **Environment Variables:** MCPServer resources support environment variables with:
  - Direct values: `value: "some-value"`
  - Secret references: `valueFrom.secretKeyRef` pointing to a Kubernetes Secret

Example with secret reference:
```yaml
apiVersion: acp.humanlayer.dev/acp
kind: MCPServer
metadata:
  name: mcp-server-with-secret
spec:
  transport: stdio
  command: "/usr/local/bin/mcp-server"
  env:
    - name: API_KEY
      valueFrom:
        secretKeyRef:
          name: my-secret
          key: api-key
```

For full examples, see the `config/samples/` directory.

### LLM

The LLM resource defines a language model configuration, including:
- Provider information (e.g., OpenAI)
- API key references (using Kubernetes Secrets)
- Model configurations

### Agent

The Agent resource defines an LLM agent with:
- A reference to an LLM
- System prompt
- Available tools

### Tool

The Tool resource defines a capability that can be used by an Agent, such as:
- Function-based tools
- MCP-provided tools
- Human approval tools

### Task

The Task resource represents a request to an Agent, which starts a conversation.

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

