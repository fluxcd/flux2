---
title: "flux get images update command"
---
## flux get images update

Get ImageUpdateAutomation status

### Synopsis

The get image update command prints the status of ImageUpdateAutomation objects.

```
flux get images update [flags]
```

### Examples

```
  # List all image update automation object and their status
  flux get image update

 # List image update automations from all namespaces
  flux get image update --all-namespaces
```

### Options

```
  -h, --help   help for update
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

