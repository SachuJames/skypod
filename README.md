<div align="center">

<img src="docs/assets/banner.svg" alt="skypod" width="100%"/>

[![CI](https://github.com/SachuJames/skypod/actions/workflows/ci.yml/badge.svg)](https://github.com/SachuJames/skypod/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/SachuJames/skypod)](https://github.com/SachuJames/skypod/releases)
![Go](https://img.shields.io/badge/go-1.22%2B-00ADD8)
![Kubernetes](https://img.shields.io/badge/kubernetes-1.30-326CE5)
![Terraform](https://img.shields.io/badge/terraform-1.5%2B-7B42BC)
![Helm](https://img.shields.io/badge/helm-v3-0F1689)
![License](https://img.shields.io/badge/license-MIT-green)

**[Release v0.1.0](https://github.com/SachuJames/skypod/releases/tag/v0.1.0)** · **[Image on GHCR](https://github.com/SachuJames/skypod/pkgs/container/skypod)** · **[Helm chart repo](https://sachujames.github.io/skypod/)**

<img src="https://skillicons.dev/icons?i=go,kubernetes,terraform,docker,helm,prometheus,grafana,githubactions,aws" alt="Go, Kubernetes, Terraform, Docker, Helm, Prometheus, Grafana, GitHub Actions, AWS" height="44"/>

One repo, two halves: an application worth deploying, and everything
needed to deploy it properly, locally on kind or on AWS EKS.

</div>

## ☁️ What this is

The app is a tiny Go HTTP service on purpose. It answers, it burns CPU on
request (`/work?ms=300`) so autoscaling has something real to react to,
it exposes its own Prometheus metrics, and it ships as a small static
binary in a scratch image, with no shell and no userland to speak of.
The interesting part is the platform around it:

- a **Helm chart** with probes, a hardened security context, NetworkPolicy
  and Pod Security Standards, an HPA and a PodDisruptionBudget
- **Terraform** that builds the AWS side from nothing: VPC, subnets,
  NAT, IAM, an EKS cluster and a node group in private subnets
- **Prometheus and Grafana** wiring: a ServiceMonitor for the app, a
  dashboard for its metrics, an SLO dashboard, recording rules, and
  alerts including burn-rate ones against a 99.9 percent target
- **GitOps**: an ArgoCD Application installs the chart from the
  published Helm repo and keeps the cluster matching it
- **CI** that vets and tests the Go code, lints and renders the chart,
  checks and validates the Terraform, and builds the image, plus a
  **release workflow** that publishes to GHCR

```mermaid
flowchart LR
    user(["client / load script"]) --> svc["Service :80"]
    svc --> pods["skypod pods"]
    hpa["HPA<br/>65 percent CPU"] -.->|scales| pods
    prom["Prometheus"] -.->|scrapes /metrics| pods
    graf["Grafana"] --> prom
    tf["Terraform"] -->|provisions| eks["AWS EKS"]
    eks -.->|runs| pods
    kind["kind, local"] -.->|mirrors| pods
```

## 🚀 Quickstart: run it locally

You need Docker, plus `kubectl`, `helm` and `kind` (versions print with
`make tools`). Then:

```
make demo
kubectl -n skypod port-forward svc/skypod 8080:80
curl localhost:8080/
```

> [!TIP]
> `make demo` is the whole show: it creates a three node kind cluster,
> builds the image, loads it in, and installs the chart with autoscaling on.

The pod security profile is already baked into the chart; if you want the
namespace itself labelled for the restricted standard, run `make secure-ns`
once after the deploy.

Now give the HPA something to do:

```
make load
kubectl -n skypod get hpa -w
```

The load script hammers `/work?ms=200` from a dozen workers for two
minutes. CPU climbs past the 65 percent target and the replica count
follows it up, then settles back down when the load stops.

## ⎈ Install the chart from the Helm repo

Already have a cluster? The chart is published as a Helm repository on
GitHub Pages:

```
helm repo add skypod https://sachujames.github.io/skypod/
helm repo update
helm install skypod skypod/skypod
```

## 🔄 Deploy with GitOps

If you run ArgoCD, this is a first-class path, not an afterthought: the
Application in `deploy/argocd/` installs the chart straight from the
Helm repo above, with automated sync, prune and self-heal. To move to a
new chart version, bump its targetRevision and apply again.

```
kubectl apply -f deploy/argocd/application.yaml
```

The full flow, updating included, is in `deploy/argocd/README.md`.

## 🌩️ The AWS path

```
cd terraform
terraform init
terraform plan
terraform apply
```

> [!WARNING]
> An EKS cluster bills real money by the hour, control plane, nodes and
> NAT included. Read the cost notes in `terraform/README.md` first, and
> run `terraform destroy` when you are done looking at it.

This creates the EKS cluster described in `terraform/README.md`. Connect
and install the chart:

```
aws eks update-kubeconfig --name skypod --region ap-south-1
helm upgrade --install skypod charts/skypod --namespace skypod --create-namespace
```

## 📦 What is inside

| Directory | Contents |
| --- | --- |
| `app/` | The Go service, its tests, and a multi-stage Dockerfile |
| `charts/skypod/` | The Helm chart: Deployment, Service, HPA, PDB, ServiceMonitor and an optional Ingress |
| `terraform/` | VPC, EKS cluster, managed node group, IAM. Plain resources, no external modules |
| `deploy/local/` | The kind cluster definition used by the Makefile |
| `deploy/monitoring/` | kube-prometheus-stack values, an app dashboard, an SLO dashboard, recording rules and alert rules |
| `deploy/argocd/` | The GitOps path: an ArgoCD Application installing the chart from the published Helm repo |
| `.github/workflows/` | CI on every push, and the release publisher |

The app itself:

| Route | What it does |
| --- | --- |
| `GET /` | JSON with service name, version and pod hostname |
| `GET /healthz`, `GET /readyz` | Plain 200 ok, used as the probes |
| `GET /metrics` | Prometheus text, hand-written, no client library |
| `GET /work?ms=N` | Burns about N ms of CPU (capped at 2000) |
| `GET /echo` | Reflects method, path and headers back as JSON |

## 📈 Observability

`make monitoring` installs kube-prometheus-stack into its own namespace.
The chart's ServiceMonitor points Prometheus at the app's `/metrics`, the
dashboard in `deploy/monitoring/dashboards/` covers request rate, error
rate, latency, pod CPU and HPA replicas, and `alerts.yaml` adds rules for
a high error rate and for pods restarting too often.

## 🔁 CI and releases

Every push runs four checks: Go vet and tests, Helm lint and template,
Terraform format and validate, and a Docker build. Tagging a version
(`git tag v0.1.0`) triggers the release workflow, which pushes
`ghcr.io/sachujames/skypod` and attaches the packaged Helm chart to the
GitHub release. The same chart is served from the
[Helm repo on GitHub Pages](https://sachujames.github.io/skypod/).

## ⚠️ Honest limits

The Terraform is checked with `fmt` and `validate` in CI and locally, and
the chart is linted, rendered and schema-checked, but the EKS stack in
this repo has not been applied to a live AWS account yet. The same goes
for the NetworkPolicy, the Pod Security labels and the recording rules:
rendered and schema-checked, not yet run against a live cluster. The
kind
quickstart was written and verified at the template level in an
environment without a Docker daemon, so the full load test loop is
documented from the mechanism (HPA at 65 percent of the CPU request,
`/work` burning real CPU) rather than from a run I can show you numbers
for. If you run `make demo` on a machine with Docker, that is the demo.

The service is stateless and tiny by design. The point of the repo is
the platform, and the platform makes no claim to be finished either:
ingress is off by default, there is no TLS story, and the Terraform uses
one NAT gateway where a serious setup would run one per AZ.

## 📄 License

MIT. See [LICENSE](LICENSE).

<div align="center">
<img src="docs/assets/footer.svg" alt="" width="100%"/>
</div>
