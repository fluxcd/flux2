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
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux bootstrap](flux_bootstrap.md)	 - Bootstrap toolkit components
* [flux check](flux_check.md)	 - Check requirements and installation
* [flux completion](flux_completion.md)	 - Generates completion scripts for various shells
* [flux create](flux_create.md)	 - Create or update sources and resources
* [flux delete](flux_delete.md)	 - Delete sources and resources
* [flux export](flux_export.md)	 - Export resources in YAML format
* [flux get](flux_get.md)	 - Get sources and resources
* [flux install](flux_install.md)	 - Install or upgrade Flux
* [flux logs](flux_logs.md)	 - Display formatted logs for Flux components
* [flux reconcile](flux_reconcile.md)	 - Reconcile sources and resources
* [flux resume](flux_resume.md)	 - Resume suspended resources
* [flux suspend](flux_suspend.md)	 - Suspend resources
* [flux uninstall](flux_uninstall.md)	 - Uninstall Flux and its custom resource definitions

