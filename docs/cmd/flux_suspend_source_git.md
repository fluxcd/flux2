---
title: "flux suspend source git command"
---
## flux suspend source git

Suspend reconciliation of a GitRepository

### Synopsis

The suspend command disables the reconciliation of a GitRepository resource.

```
flux suspend source git [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing GitRepository
  flux suspend source git podinfo
```

### Options

```
  -h, --help   help for git
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

