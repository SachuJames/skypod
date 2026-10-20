# Deploying skypod with ArgoCD

The Application in this folder installs the skypod chart from the
published Helm repo, and ArgoCD keeps the cluster matching it from
there on: new chart versions are picked up by syncing, not by running
helm by hand.

## Install ArgoCD

Follow the current official instructions (they amount to applying the
install manifest and waiting for the pods):

```
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

## Apply the Application

```
kubectl apply -f deploy/argocd/application.yaml
```

ArgoCD creates the skypod namespace and installs the chart with
autoscaling on and the ServiceMonitor enabled.

## Updating

When a new chart version is published, bump `targetRevision` in
`application.yaml` to it and apply the file again. ArgoCD syncs the
change with prune and self-heal, so removed resources are cleaned up
and manual edits drift back to the chart.

## Checking status

```
kubectl -n argocd get applications
argocd app get skypod        # if the CLI is installed
```

Healthy means Synced and the resources showing in the desired state.
