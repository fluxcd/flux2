---
title: "flux get sources chart command"
---
## flux get sources chart

Get HelmChart statuses

### Synopsis

The get sources chart command prints the status of the HelmCharts.

```
flux get sources chart [flags]
```

### Examples

```
  # List all Helm charts and their status
  flux get sources chart

 # List Helm charts from all namespaces
  flux get sources chart --all-namespaces
```

### Options

```
  -h, --help   help for chart
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

* [flux get sources](../flux_get_sources/)	 - Get source statuses

