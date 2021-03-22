---
title: "flux create secret command"
---
## flux create secret

Create or update Kubernetes secrets

### Synopsis

The create source sub-commands generate Kubernetes secrets specific to Flux.

### Options

```
  -h, --help   help for secret
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   absolute path to the kubeconfig file
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux create](/cmd/flux_create/)	 - Create or update sources and resources
* [flux create secret git](/cmd/flux_create_secret_git/)	 - Create or update a Kubernetes secret for Git authentication
* [flux create secret helm](/cmd/flux_create_secret_helm/)	 - Create or update a Kubernetes secret for Helm repository authentication
* [flux create secret tls](/cmd/flux_create_secret_tls/)	 - Create or update a Kubernetes secret with TLS certificates

