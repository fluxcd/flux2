---
title: "flux resume image repository command"
---
## flux resume image repository

Resume a suspended ImageRepository

### Synopsis

The resume command marks a previously suspended ImageRepository resource for reconciliation and waits for it to finish.

```
flux resume image repository [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing ImageRepository
  flux resume image repository alpine
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

* [flux resume image](../flux_resume_image/)	 - Resume image automation objects

