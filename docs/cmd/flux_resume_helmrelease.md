---
title: "flux resume helmrelease command"
---
## flux resume helmrelease

Resume a suspended HelmRelease

### Synopsis

The resume command marks a previously suspended HelmRelease resource for reconciliation and waits for it to
finish the apply.

```
flux resume helmrelease [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing Helm release
  flux resume hr podinfo
```

### Options

```
  -h, --help   help for helmrelease
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

* [flux resume](../flux_resume/)	 - Resume suspended resources

