---
title: "flux suspend source bucket command"
---
## flux suspend source bucket

Suspend reconciliation of a Bucket

### Synopsis

The suspend command disables the reconciliation of a Bucket resource.

```
flux suspend source bucket [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing Bucket
  flux suspend source bucket podinfo
```

### Options

```
  -h, --help   help for bucket
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

* [flux suspend source](../flux_suspend_source/)	 - Suspend sources

