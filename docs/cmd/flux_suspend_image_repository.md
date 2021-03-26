---
title: "flux suspend image repository command"
---
## flux suspend image repository

Suspend reconciliation of an ImageRepository

### Synopsis

The suspend image repository command disables the reconciliation of a ImageRepository resource.

```
flux suspend image repository [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing ImageRepository
  flux suspend image repository alpine
```

### Options

```
  -h, --help   help for repository
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

* [flux suspend image](/cmd/flux_suspend_image/)	 - Suspend image automation objects

