---
title: "flux get sources bucket command"
---
## flux get sources bucket

Get Bucket source statuses

### Synopsis

The get sources bucket command prints the status of the Bucket sources.

```
flux get sources bucket [flags]
```

### Examples

```
  # List all Buckets and their status
  flux get sources bucket

 # List buckets from all namespaces
  flux get sources helm --all-namespaces
```

### Options

```
  -h, --help   help for bucket
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

