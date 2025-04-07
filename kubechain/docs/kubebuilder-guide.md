# Kubebuilder Development Guide

## Overview

Kubechain is built using [Kubebuilder](https://book.kubebuilder.io/), a framework for building Kubernetes APIs using custom resource definitions (CRDs). This guide explains how to use kubebuilder in this project, particularly for adding new resources and maintaining existing ones.

## Current Project Structure

The project uses Kubebuilder v4 with a domain of `humanlayer.dev` and an API group of `kubechain`. All resources are in the `v1alpha1` version and are namespaced.

Current resources include:
- `LLM` - Configuration for large language models
- `Agent` - Defines an agent using an LLM and tools
- `Tool` - Defines tools that can be used by agents
- `Task` - Represents a run of a task
- `TaskRunToolCall` - Represents a tool call during a task run
- `MCPServer` - Defines a Model Control Protocol server for tool integration

## Adding a New Resource

To add a new resource to the project, follow these steps:

1. Create the new resource using kubebuilder:

```bash
kubebuilder create api --group kubechain --version v1alpha1 --kind YourNewResource --namespaced true --resource true --controller true
```

This will:
- Create a new file in `api/v1alpha1/yournewresource_types.go`
- Create a new controller in `internal/controller/yournewresource/`
- Update the PROJECT file with the new resource

2. Define your resource fields in the `*Spec` and `*Status` structs in the generated `_types.go` file.

3. Add RBAC annotations to the controller in `internal/controller/yournewresource/yournewresource_controller.go`:

```go
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=yournewresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=yournewresources/status,verbs=get;update;patch
```

Add additional RBAC annotations for any other resources your controller needs to access:

```go
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=someotherresource,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
```

4. Add kubebuilder printing column annotations to your resource's struct:

```go
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
```

5. Generate manifests to create the CRD and update RBAC:

```bash
make manifests
```

### Example: Adding a ContactChannel Resource

Here's an example of creating a ContactChannel resource:

```bash
kubebuilder create api --group kubechain --version v1alpha1 --kind ContactChannel --namespaced true --resource true --controller true
```

Then, edit the generated `api/v1alpha1/contactchannel_types.go` file:

```go
type ContactChannelSpec struct {
	// Type specifies the type of contact channel (e.g., "email", "slack", "teams")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=email;slack;teams
	Type string `json:"type"`

	// Address is the destination for notifications (email address, channel ID, etc.)
	// +kubebuilder:validation:Required
	Address string `json:"address"`

	// SecretRef contains credentials needed to access the channel
	// +optional
	SecretRef *SecretKeyRef `json:"secretRef,omitempty"`
}

type ContactChannelStatus struct {
	// Ready indicates if the channel is ready to receive notifications
	Ready bool `json:"ready,omitempty"`

	// Status indicates the current status of the channel
	// +kubebuilder:validation:Enum=Ready;Error;Pending
	Status string `json:"status,omitempty"`

	// StatusDetail provides additional details about the current status
	StatusDetail string `json:"statusDetail,omitempty"`
}
```

Edit the controller to add RBAC annotations:

```go
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=contactchannels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=contactchannels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// ContactChannelReconciler reconciles a ContactChannel object
type ContactChannelReconciler struct {
    // ...
}
```

Add printing columns:

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Address",type="string",JSONPath=".spec.address"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:resource:scope=Namespaced
```

After making these changes, run:

```bash
make manifests
```

## Common Pitfalls and Solutions

### 1. Missing or Incorrect RBAC Annotations

**Problem**: The controller can't access resources it needs because RBAC permissions are missing.

**Solution**: Make sure to add proper RBAC annotations to your controller before running `make manifests`. Remember to include permissions for any resources your controller accesses, not just the one it's primarily responsible for.

### 2. PROJECT File Out of Sync

**Problem**: The PROJECT file doesn't contain all resources or has incorrect information.

**Solution**: If kubebuilder doesn't update the PROJECT file correctly, you can manually edit it to ensure all resources are properly listed. After manual edits, run `make manifests` to regenerate manifests.

### 3. DeepCopy Methods Missing

**Problem**: After adding new fields, you get compilation errors about missing DeepCopy methods.

**Solution**: Run `make generate` to regenerate DeepCopy methods:

```bash
make generate
```

### 4. CRD Validation Issues

**Problem**: The CRD validation schema doesn't match your expectations.

**Solution**: Check the kubebuilder validation annotations in your type definitions. Most validation is done using annotations like:

```go
// +kubebuilder:validation:Minimum=0
// +kubebuilder:validation:Maximum=100
// +kubebuilder:validation:Enum=option1;option2;option3
```

### 5. Forgetting to Update the Controller Implementation

**Problem**: The new resource is created but doesn't do anything.

**Solution**: Remember to implement the Reconcile function in your controller to handle the resource's logic.

## Kubebuilder Commands Reference

### Initialize a New Project

```bash
kubebuilder init --domain example.com --repo github.com/example/project
```

### Create a New API/Resource

```bash
kubebuilder create api --group groupname --version v1 --kind KindName
```

### Generate Code and Manifests

```bash
# Update generated code
make generate

# Generate CRDs and RBAC
make manifests
```

### Install CRDs into a Cluster

```bash
make install
```

### Package for Release

```bash
make release-local tag=v0.1.0
```

## Best Practices

1. **Use proper validation annotations**: Add validation to your resource fields with kubebuilder annotations to ensure users provide valid data.

2. **Keep RBAC minimal**: Only request the permissions your controller actually needs. This follows the principle of least privilege.

3. **Include status conditions**: Status conditions provide a standardized way to report resource state.

4. **Add printer columns**: Printer columns make the resource more user-friendly when viewed with `kubectl get`.

5. **Document your resource fields**: Use comments to document what each field does and any constraints.

6. **Write unit tests**: Test your controller's reconciliation logic with proper unit tests.

7. **Follow Kubernetes API conventions**: Follow the [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md) for naming and field organization.

## Additional Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)