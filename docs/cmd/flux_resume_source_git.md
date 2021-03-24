---
title: "flux resume source git command"
---
## flux resume source git

Resume a suspended GitRepository

### Synopsis

The resume command marks a previously suspended GitRepository resource for reconciliation and waits for it to finish.

```
flux resume source git [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing GitRepository
  flux resume source git podinfo

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

* [flux resume source](/cmd/flux_resume_source/)	 - Resume sources

