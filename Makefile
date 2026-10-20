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

cluster:
	kind create cluster --config deploy/local/kind-config.yaml

image:
	docker build --build-arg VERSION=$(TAG) -t $(IMAGE):$(TAG) ./app
	kind load docker-image $(IMAGE):$(TAG) --name skypod

deploy:
	helm upgrade --install $(RELEASE) charts/skypod \
		--namespace $(NAMESPACE) --create-namespace \
		--set image.repository=$(IMAGE) --set image.tag=$(TAG) \
		--set serviceMonitor.enabled=true

# Label the namespace for the restricted Pod Security profile.
secure-ns:
	kubectl label namespace $(NAMESPACE) \
		pod-security.kubernetes.io/enforce=restricted \
		pod-security.kubernetes.io/audit=restricted \
		pod-security.kubernetes.io/warn=restricted --overwrite

monitoring:
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	helm upgrade --install monitoring prometheus-community/kube-prometheus-stack \
		--namespace monitoring --create-namespace \
		--values deploy/monitoring/values.yaml
	kubectl -n skypod apply -f deploy/monitoring/alerts.yaml

load:
	bash scripts/load.sh

# Everything, end to end. Then: kubectl port-forward -n skypod svc/skypod 8080:80
demo: cluster image deploy
	@echo "Deployed. Try: kubectl -n skypod port-forward svc/skypod 8080:80"

clean:
	-kind delete cluster --name skypod
