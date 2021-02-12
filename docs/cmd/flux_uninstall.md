## flux uninstall

Uninstall Flux and its custom resource definitions

### Synopsis

The uninstall command removes the Flux components and the toolkit.fluxcd.io resources from the cluster.

```
flux uninstall [flags]
```

### Examples

```
  # Uninstall Flux components, its custom resources and namespace
  flux uninstall --namespace=flux-system

  # Uninstall Flux but keep the namespace
  flux uninstall --namespace=infra --keep-namespace=true

```

### Options

```
      --dry-run          only print the objects that would be deleted
  -h, --help             help for uninstall
      --keep-namespace   skip namespace deletion
  -s, --silent           delete components without asking for confirmation
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux](flux.md)	 - Command line utility for assembling Kubernetes CD pipelines

