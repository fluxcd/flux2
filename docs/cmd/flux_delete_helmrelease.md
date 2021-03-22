---
title: "flux delete helmrelease command"
---
## flux delete helmrelease

Delete a HelmRelease resource

### Synopsis

The delete helmrelease command removes the given HelmRelease from the cluster.

```
flux delete helmrelease [name] [flags]
```

### Examples

```
  # Delete a Helm release and the Kubernetes resources created by it
  flux delete hr podinfo

```

### Options

```
  -h, --help   help for helmrelease
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

