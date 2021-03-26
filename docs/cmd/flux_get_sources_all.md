---
title: "flux get sources all command"
---
## flux get sources all

Get all source statuses

### Synopsis

The get sources all command print the statuses of all sources.

```
flux get sources all [flags]
```

### Examples

```
  # List all sources in a namespace
  flux get sources all --namespace=flux-system

  # List all sources in all namespaces
  flux get sources all --all-namespaces
```

### Options

```
  -h, --help   help for all
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

