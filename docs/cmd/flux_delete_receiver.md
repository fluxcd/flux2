---
title: "flux delete receiver command"
---
## flux delete receiver

Delete a Receiver resource

### Synopsis

The delete receiver command removes the given Receiver from the cluster.

```
flux delete receiver [name] [flags]
```

### Examples

```
  # Delete an Receiver and the Kubernetes resources created by it
  flux delete receiver main

```

### Options

```
  -h, --help   help for receiver
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

