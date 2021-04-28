---
title: "flux suspend source chart command"
---
## flux suspend source chart

Suspend reconciliation of a HelmChart

### Synopsis

The suspend command disables the reconciliation of a HelmChart resource.

```
flux suspend source chart [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing HelmChart
  flux suspend source chart podinfo
```

### Options

```
  -h, --help   help for chart
```

### Options inherited from parent commands

```
      --all                 suspend all resources in that namespace
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux suspend source](../flux_suspend_source/)	 - Suspend sources

