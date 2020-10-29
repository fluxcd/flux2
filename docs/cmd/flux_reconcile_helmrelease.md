## flux reconcile helmrelease

Reconcile a HelmRelease resource

### Synopsis


The reconcile kustomization command triggers a reconciliation of a HelmRelease resource and waits for it to finish.

```
flux reconcile helmrelease [name] [flags]
```

### Examples

```
  # Trigger a HelmRelease apply outside of the reconciliation interval
  flux reconcile hr podinfo

  # Trigger a reconciliation of the HelmRelease's source and apply changes
  flux reconcile hr podinfo --with-source

```

### Options

```
  -h, --help          help for helmrelease
      --with-source   reconcile HelmRelease source
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux reconcile](flux_reconcile.md)	 - Reconcile sources and resources

