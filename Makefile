NAMESPACE := skypod
RELEASE   := skypod
IMAGE     := ghcr.io/sachujames/skypod
TAG       ?= local

.PHONY: tools cluster image deploy secure-ns monitoring load demo clean

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

# Label the app namespace so Pod Security Standards enforce the
# restricted profile in it. Run once per cluster, after deploy.
secure-ns:
	kubectl label namespace $(NAMESPACE) \
		pod-security.kubernetes.io/enforce=restricted \
		pod-security.kubernetes.io/audit=restricted \
		pod-security.kubernetes.io/warn=restricted --overwrite

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
