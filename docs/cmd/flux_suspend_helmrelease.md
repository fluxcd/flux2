---
title: "flux suspend helmrelease command"
---
## flux suspend helmrelease

Suspend reconciliation of HelmRelease

### Synopsis

The suspend command disables the reconciliation of a HelmRelease resource.

```
flux suspend helmrelease [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing Helm release
  flux suspend hr podinfo

```

### Options

```
  -h, --help   help for helmrelease
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux suspend](/cmd/flux_suspend/)	 - Suspend resources

