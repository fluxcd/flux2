---
title: "flux get images all command"
---
## flux get images all

Get all image statuses

### Synopsis

The get image sub-commands print the statuses of all image objects.

```
flux get images all [flags]
```

### Examples

```
  # List all image objects in a namespace
  flux get images all --namespace=flux-system

  # List all image objects in all namespaces
  flux get images all --all-namespaces
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

* [flux get images](/cmd/flux_get_images/)	 - Get image automation object status

