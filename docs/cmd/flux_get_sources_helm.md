---
title: "flux get sources helm command"
---
## flux get sources helm

Get HelmRepository source statuses

### Synopsis

The get sources helm command prints the status of the HelmRepository sources.

```
flux get sources helm [flags]
```

### Examples

```
  # List all Helm repositories and their status
  flux get sources helm

 # List Helm repositories from all namespaces
  flux get sources helm --all-namespaces

```

### Options

```
  -h, --help   help for helm
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

