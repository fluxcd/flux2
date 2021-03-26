---
title: "flux delete source git command"
---
## flux delete source git

Delete a GitRepository source

### Synopsis

The delete source git command deletes the given GitRepository from the cluster.

```
flux delete source git [name] [flags]
```

### Examples

```
  # Delete a Git repository
  flux delete source git podinfo
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
  -s, --silent              delete resource without asking for confirmation
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux delete source](/cmd/flux_delete_source/)	 - Delete sources

