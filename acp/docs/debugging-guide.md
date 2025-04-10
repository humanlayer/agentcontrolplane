# Debugging Guide

This guide explains how to debug the Kubechain operator locally using VS Code's debugger, allowing you to set breakpoints and step through the code execution.

## Prerequisites

- VS Code with the Go extension installed
- A running Kind cluster
- Basic familiarity with Go debugging in VS Code

## Setup VS Code Debugging Configuration

1. Create a `.vscode/launch.json` file in the kubechain directory with the following configuration:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Operator",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/main.go",
            "args": [],
            "env": {
                "KUBECONFIG": "${env:HOME}/.kube/config"
            }
        }
    ]
}
```

This configuration tells VS Code how to launch the operator in debug mode, using your standard kubeconfig file at `~/.kube/config`. Adjust the KUBECONFIG path if your configuration is located elsewhere.

## Debugging Workflow

### 1. Remove Currently Deployed Operator (if any)

If you already have the operator deployed in your cluster (e.g., via `make deploy-operator`), you'll need to remove it to avoid conflicts:

```bash
make undeploy-operator
```

This command removes both the operator deployment and the CRDs from your cluster.

### 2. Reinstall CRDs Only

To debug with existing resources, reinstall just the CRDs:

```bash
make kubechain-install
```

This command reinstalls the CRDs into your cluster without deploying the operator.

### 3. Set Breakpoints

In VS Code, set breakpoints in the controller code where you want to pause execution.

### 4. Start Debugging

Press F5 in VS Code or click the "Start Debugging" button in the Run & Debug panel to launch the operator in debug mode.

## Tips for Effective Debugging

1. **Use Conditional Breakpoints**: For controllers that handle many resources, use conditional breakpoints to break only on specific resources:
   ```
   resource.GetName() == "my-specific-resource"
   ```

2. **Debug Logs**: Add temporary debug log statements to track execution flow:
   ```go
   logger.Info("Processing resource", "status", resource.Status.Status, "phase", resource.Status.Phase)
   ```

3. **Focus on State Transitions**: Set breakpoints at the beginning and end of state transition code to understand how resources move through their lifecycles.

4. **Watch Resource Status**: Keep a terminal open with `kubectl get <resource> -w` to observe the effects of your code on the resource status.