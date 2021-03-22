---
title: "flux get alert-providers command"
---
## flux get alert-providers

Get Provider statuses

### Synopsis

The get alert-provider command prints the statuses of the resources.

```
flux get alert-providers [flags]
```

### Examples

```
  # List all Providers and their status
  flux get alert-providers

```

### Options

```
  -h, --help   help for alert-providers
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

* [flux get](/cmd/flux_get/)	 - Get the resources and their status

