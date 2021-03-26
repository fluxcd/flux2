---
title: "flux get sources git command"
---
## flux get sources git

Get GitRepository source statuses

### Synopsis

The get sources git command prints the status of the GitRepository sources.

```
flux get sources git [flags]
```

### Examples

```
  # List all Git repositories and their status
  flux get sources git

 # List Git repositories from all namespaces
  flux get sources git --all-namespaces
```

### Options

```
  -h, --help   help for git
```

### Options inherited from parent commands

```
  -A, --all-namespaces      list the requested object(s) across all namespaces
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux get sources](/cmd/flux_get_sources/)	 - Get source statuses

