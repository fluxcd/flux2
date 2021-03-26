---
title: "flux delete alert command"
---
## flux delete alert

Delete a Alert resource

### Synopsis

The delete alert command removes the given Alert from the cluster.

```
flux delete alert [name] [flags]
```

### Examples

```
  # Delete an Alert and the Kubernetes resources created by it
  flux delete alert main
```

### Options

```
  -h, --help   help for alert
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

* [flux delete](/cmd/flux_delete/)	 - Delete sources and resources

