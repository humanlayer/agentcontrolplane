[Makefile](./Makefile) - commands to deploy the stack - use `make otel-stack otel-test` to deploy the stack and test it
[agents](./agents) - custom resources to deploy a simple calculator agent
[otel](./otel) - values for the otel collector, and script to push data in


# What is happening here?

This is a demo environment for Kubechain that also deploys a complete observability stack (Prometheus, Grafana, Tempo, Loki) for monitoring LLM operations. You can use this demo to test and interact with LLM-powered agents running in a K8s cluster (behind `kind`).

# Prerequisites

- Docker
- Kind (Kubernetes in Docker)
- Helm
- kubectl
- uv (Python package manager)

## Getting Started

1. Create a local Kubernetes cluster:
   ```
   make kind-up
   ```

2. Deploy the complete observability stack:
   ```
   make otel-stack
   ```

3. Build and deploy the Kubechain operator:
   ```
   make operator-build operator-deploy
   ```

4. Deploy example agents:
   ```
   kubectl apply -f agents/crds.yaml
   ```

5. Run a test to generate telemetry data:
   ```
   make otel-test
   ```

6. Access the monitoring dashboards:
   ```
   make otel-access
   ```
   
   - Grafana: http://localhost:13000 (password: admin)
   - Prometheus: http://localhost:9090

## Cleanup

To tear down the environment:

```
make otel-stack-down
make kind-down
```
