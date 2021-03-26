---
title: "flux resume source helm command"
---
## flux resume source helm

Resume a suspended HelmRepository

### Synopsis

The resume command marks a previously suspended HelmRepository resource for reconciliation and waits for it to finish.

```
flux resume source helm [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing HelmRepository
  flux resume source helm bitnami
```

### Options

```
  -h, --help   help for helm
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

* [flux resume source](../flux_resume_source/)	 - Resume sources

