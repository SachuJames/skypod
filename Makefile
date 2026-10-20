NAMESPACE := skypod
RELEASE   := skypod
IMAGE     := ghcr.io/sachujames/skypod
TAG       ?= local

.PHONY: tools cluster image deploy monitoring load demo clean

tools:
	@go version
	@kubectl version --client
	@helm version --short
	@kind version
	@terraform version

# Create the local cluster.
cluster:
	kind create cluster --config deploy/local/kind-config.yaml

# Build the image and load it into kind.
image:
	docker build --build-arg VERSION=$(TAG) -t $(IMAGE):$(TAG) ./app
	kind load docker-image $(IMAGE):$(TAG) --name skypod

# Install the chart into the cluster.
deploy:
	helm upgrade --install $(RELEASE) charts/skypod \
		--namespace $(NAMESPACE) --create-namespace \
		--set image.repository=$(IMAGE) --set image.tag=$(TAG) \
		--set serviceMonitor.enabled=true

# Install Prometheus and Grafana next to the app.
monitoring:
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	helm upgrade --install monitoring prometheus-community/kube-prometheus-stack \
		--namespace monitoring --create-namespace \
		--values deploy/monitoring/values.yaml
	kubectl -n skypod apply -f deploy/monitoring/alerts.yaml

# Burn CPU until the autoscaler adds pods.
load:
	bash scripts/load.sh

# Everything, end to end. Then: kubectl port-forward -n skypod svc/skypod 8080:80
demo: cluster image deploy
	@echo "Deployed. Try: kubectl -n skypod port-forward svc/skypod 8080:80"

clean:
	-kind delete cluster --name skypod
