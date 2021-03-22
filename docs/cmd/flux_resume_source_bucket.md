---
title: "flux resume source bucket command"
---
## flux resume source bucket

Resume a suspended Bucket

### Synopsis

The resume command marks a previously suspended Bucket resource for reconciliation and waits for it to finish.

```
flux resume source bucket [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing Bucket
  flux resume source bucket podinfo

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

* [flux resume source](/cmd/flux_resume_source/)	 - Resume sources

