## tk reconcile helmrelease

Reconcile a HelmRelease resource

### Synopsis


The reconcile kustomization command triggers a reconciliation of a HelmRelease resource and waits for it to finish.

```
tk reconcile helmrelease [name] [flags]
```

### Examples

```
  # Trigger a HelmRelease apply outside of the reconciliation interval
  tk reconcile hr podinfo

  # Trigger a reconciliation of the HelmRelease's source and apply changes
  tk reconcile hr podinfo --with-source

```

### Options

```
  -h, --help          help for helmrelease
      --with-source   reconcile HelmRelease source
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [tk reconcile](tk_reconcile.md)	 - Reconcile sources and resources

