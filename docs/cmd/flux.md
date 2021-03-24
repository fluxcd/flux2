---
title: "flux command"
---
## flux

Command line utility for assembling Kubernetes CD pipelines

### Synopsis

Command line utility for assembling Kubernetes CD pipelines the GitOps way.

### Examples

```
  # Check prerequisites
  flux check --pre

  # Install the latest version of Flux
  flux install --version=master

  # Create a source from a public Git repository
  flux create source git webapp-latest \
    --url=https://github.com/stefanprodan/podinfo \
    --branch=master \
    --interval=3m

  # List GitRepository sources and their status
  flux get sources git

  # Trigger a GitRepository source reconciliation
  flux reconcile source git flux-system

  # Export GitRepository sources in YAML format
  flux export source git --all > sources.yaml

  # Create a Kustomization for deploying a series of microservices
  flux create kustomization webapp-dev \
    --source=webapp-latest \
    --path="./deploy/webapp/" \
    --prune=true \
    --interval=5m \
    --validation=client \
    --health-check="Deployment/backend.webapp" \
    --health-check="Deployment/frontend.webapp" \
    --health-check-timeout=2m

  # Trigger a git sync of the Kustomization's source and apply changes
  flux reconcile kustomization webapp-dev --with-source

  # Suspend a Kustomization reconciliation
  flux suspend kustomization webapp-dev

  # Export Kustomizations in YAML format
  flux export kustomization --all > kustomizations.yaml

  # Resume a Kustomization reconciliation
  flux resume kustomization webapp-dev

  # Delete a Kustomization
  flux delete kustomization webapp-dev

  # Delete a GitRepository source
  flux delete source git webapp-latest

  # Uninstall Flux and delete CRDs
  flux uninstall

```

### Options

```
      --context string      kubernetes context to use
  -h, --help                help for flux
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux bootstrap](/cmd/flux_bootstrap/)	 - Bootstrap toolkit components
* [flux check](/cmd/flux_check/)	 - Check requirements and installation
* [flux completion](/cmd/flux_completion/)	 - Generates completion scripts for various shells
* [flux create](/cmd/flux_create/)	 - Create or update sources and resources
* [flux delete](/cmd/flux_delete/)	 - Delete sources and resources
* [flux export](/cmd/flux_export/)	 - Export resources in YAML format
* [flux get](/cmd/flux_get/)	 - Get the resources and their status
* [flux install](/cmd/flux_install/)	 - Install or upgrade Flux
* [flux logs](/cmd/flux_logs/)	 - Display formatted logs for Flux components
* [flux reconcile](/cmd/flux_reconcile/)	 - Reconcile sources and resources
* [flux resume](/cmd/flux_resume/)	 - Resume suspended resources
* [flux suspend](/cmd/flux_suspend/)	 - Suspend resources
* [flux uninstall](/cmd/flux_uninstall/)	 - Uninstall Flux and its custom resource definitions

