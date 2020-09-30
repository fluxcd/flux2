## gotk

Command line utility for assembling Kubernetes CD pipelines

### Synopsis

Command line utility for assembling Kubernetes CD pipelines the GitOps way.

### Examples

```
  # Check prerequisites
  gotk check --pre

  # Install the latest version of the toolkit
  gotk install --version=master

  # Create a source from a public Git repository
  gotk create source git webapp-latest \
    --url=https://github.com/stefanprodan/podinfo \
    --branch=master \
    --interval=3m

  # List GitRepository sources and their status
  gotk get sources git

  # Trigger a GitRepository source reconciliation
  gotk reconcile source git gotk-system

  # Export GitRepository sources in YAML format
  gotk export source git --all > sources.yaml

  # Create a Kustomization for deploying a series of microservices
  gotk create kustomization webapp-dev \
    --source=webapp-latest \
    --path="./deploy/webapp/" \
    --prune=true \
    --interval=5m \
    --validation=client \
    --health-check="Deployment/backend.webapp" \
    --health-check="Deployment/frontend.webapp" \
    --health-check-timeout=2m

  # Trigger a git sync of the Kustomization's source and apply changes
  gotk reconcile kustomization webapp-dev --with-source

  # Suspend a Kustomization reconciliation
  gotk suspend kustomization webapp-dev

  # Export Kustomizations in YAML format
  gotk export kustomization --all > kustomizations.yaml

  # Resume a Kustomization reconciliation
  gotk resume kustomization webapp-dev

  # Delete a Kustomization
  gotk delete kustomization webapp-dev

  # Delete a GitRepository source
  gotk delete source git webapp-latest

  # Uninstall the toolkit and delete CRDs
  gotk uninstall --crds

```

### Options

```
  -h, --help                help for gotk
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk bootstrap](gotk_bootstrap.md)	 - Bootstrap toolkit components
* [gotk check](gotk_check.md)	 - Check requirements and installation
* [gotk completion](gotk_completion.md)	 - Generates completion scripts for various shells
* [gotk create](gotk_create.md)	 - Create or update sources and resources
* [gotk delete](gotk_delete.md)	 - Delete sources and resources
* [gotk export](gotk_export.md)	 - Export resources in YAML format
* [gotk get](gotk_get.md)	 - Get sources and resources
* [gotk install](gotk_install.md)	 - Install the toolkit components
* [gotk reconcile](gotk_reconcile.md)	 - Reconcile sources and resources
* [gotk resume](gotk_resume.md)	 - Resume suspended resources
* [gotk suspend](gotk_suspend.md)	 - Suspend resources
* [gotk uninstall](gotk_uninstall.md)	 - Uninstall the toolkit components

