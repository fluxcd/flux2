---
title: "flux get all command"
---
## flux get all

Get all resources and statuses

### Synopsis

The get all command print the statuses of all resources.

```
flux get all [flags]
```

### Examples

```
  # List all resources in a namespace
  flux get all --namespace=flux-system

  # List all resources in all namespaces
  flux get all --all-namespaces
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

* [flux get](../flux_get/)	 - Get the resources and their status

