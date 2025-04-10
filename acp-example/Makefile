# Makefile for kubechain-example

# Cluster name and configuration
CLUSTER_NAME = kind
KIND_CONFIG = kind/kind-config.yaml
OTEL_CONFIG = otel/otel-config.yaml

# Environment variable to customize NodePort range (if supported by your setup)
export KIND_NODE_PORT_RANGE = 30000-35000

.PHONY: kind-up kind-down operator-deploy-local operator-deploy ui-deploy otel-deploy prometheus-deploy grafana-deploy tempo-deploy loki-up loki-down otel-stack otel-stack-down all

kind-up:
	@echo "Creating kind cluster '$(CLUSTER_NAME)'..."
	kind create cluster --name $(CLUSTER_NAME) --config $(KIND_CONFIG)

kind-down:
	@echo "Deleting kind cluster '$(CLUSTER_NAME)'..."
	kind delete cluster --name $(CLUSTER_NAME)

operator-deploy-local:
	@$(MAKE) -C ../kubechain deploy-local-kind

operator-deploy:
	kubectl apply -f https://raw.githubusercontent.com/humanlayer/smallchain/refs/heads/main/kubechain/config/release/latest.yaml

ui-deploy:
	@echo "Deploying kubechain-ui to cluster..."
	kubectl apply -f deploy/kubechain-ui.yaml

otel-deploy-base:
	@echo "Deploying OpenTelemetry collector to cluster..."
	helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
	helm repo update
	helm upgrade --install otel-collector open-telemetry/opentelemetry-collector \
		--values otel/values.yaml 

otel-patch-service:
	@echo "Patching OTEL collector service to use NodePorts..."
	kubectl patch svc otel-collector-opentelemetry-collector -p '{"spec": {"type": "NodePort", "ports": [ \
		{"name": "jaeger-compact", "port": 6831, "protocol": "UDP", "targetPort": 6831, "nodePort": 6831}, \
		{"name": "jaeger-grpc", "port": 14250, "protocol": "TCP", "targetPort": 14250, "nodePort": 14250}, \
		{"name": "jaeger-thrift", "port": 14268, "protocol": "TCP", "targetPort": 14268, "nodePort": 14268}, \
		{"name": "otlp", "port": 4317, "protocol": "TCP", "targetPort": 4317, "nodePort": 4317}, \
		{"name": "otlp-http", "port": 4318, "protocol": "TCP", "targetPort": 4318, "nodePort": 4318}, \
		{"name": "zipkin", "port": 9411, "protocol": "TCP", "targetPort": 9411, "nodePort": 9411}, \
		{"name": "prometheus", "port": 8889, "protocol": "TCP", "targetPort": 8889, "nodePort": 8889} \
	]}}'

otel-up: otel-deploy-base otel-patch-service 

otel-test:
	@echo "Waiting for OpenTelemetry collector deployment to be ready..."
	kubectl wait --for=condition=Available deployment/otel-collector-opentelemetry-collector --timeout=300s
	@echo "Waiting for Grafana deployment to be ready..."
	kubectl wait --for=condition=Available deployment/grafana --timeout=300s
	@echo "Waiting for Prometheus deployments to be ready..."
	kubectl wait --for=condition=Available deployment/prometheus-server deployment/prometheus-kube-state-metrics deployment/prometheus-prometheus-pushgateway --timeout=300s
	@echo "Waiting for Loki deployments and statefulsets to be ready..."
	kubectl wait --for=condition=Available deployment/loki-gateway --timeout=300s
	@echo "Waiting for Loki statefulset..."
	kubectl rollout status --watch --timeout=600s statefulset/loki
	@echo "Waiting for Loki chunks cache statefulset..."
	kubectl rollout status --watch --timeout=600s statefulset/loki-chunks-cache
	@echo "Waiting for Loki results cache statefulset..."
	kubectl rollout status --watch --timeout=600s statefulset/loki-results-cache
	@echo "Waiting for Tempo statefulset to be ready..."
	kubectl rollout status --watch --timeout=600s statefulset/tempo
	@echo "Waiting for Prometheus alertmanager statefulset to be ready..."
	kubectl rollout status --watch --timeout=600s statefulset/prometheus-alertmanager
	uv run otel/otel-test-traces.py
	:
	: Open Telemetry data has been sent to the collector.
	: you can view the data in grafana at http://localhost:13000/d/test-data/test-data

otel-down:
	@echo "Deleting OpenTelemetry collector from cluster..."
	helm delete otel-collector

prometheus-up:
	@echo "Deploying Prometheus to cluster..."
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	helm upgrade --install prometheus prometheus-community/prometheus \
		--values prometheus/values.yaml

prometheus-down:
	@echo "Deleting Prometheus from cluster..."
	helm delete prometheus

grafana-up:
	@echo "Deploying Grafana to cluster..."
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update
	helm upgrade --install grafana grafana/grafana \
		--values grafana/values.yaml

grafana-down:
	@echo "Deleting Grafana from cluster..."
	helm delete grafana

tempo-up:
	@echo "Deploying Tempo to cluster..."
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update
	helm upgrade --install tempo grafana/tempo \
		--set tempo.receivers.otlp.protocols.grpc.endpoint=0.0.0.0:4317 \
		--set tempo.receivers.otlp.protocols.http.endpoint=0.0.0.0:4318 \
		--set service.type=NodePort \
		--set service.ports.otlp-grpc.nodePort=4317 \
		--set service.ports.otlp-http.nodePort=4318

tempo-down:
	@echo "Deleting Tempo from cluster..."
	helm delete tempo

loki-up:
	@echo "Deploying Loki to cluster..."
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update
	helm upgrade --install loki grafana/loki \
		--values loki/values.yaml

loki-down:
	@echo "Deleting Loki from cluster..."
	helm delete loki

otel-stack: prometheus-up otel-up grafana-up tempo-up loki-up otel-test
otel-stack-down: otel-down prometheus-down grafana-down tempo-down loki-down

otel-access:
	@echo "Access instructions for monitoring stack:"
	@echo "\nGrafana:"
	@echo "Password: admin"
	@echo "Access: http://localhost:13000"
	@echo "\nPrometheus:"
	@echo "Access: http://localhost:9090"

all: kind-up operator-build operator-deploy ui-deploy otel-deploy prometheus-deploy grafana-deploy tempo-deploy loki-deploy
