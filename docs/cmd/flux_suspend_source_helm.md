## flux suspend source helm

Suspend reconciliation of a HelmRepository

### Synopsis

The suspend command disables the reconciliation of a HelmRepository resource.

```
flux suspend source helm [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing HelmRepository
  flux suspend source helm bitnami

```

### Options

```
  -h, --help   help for helm
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

* [flux suspend source](flux_suspend_source.md)	 - Suspend sources

