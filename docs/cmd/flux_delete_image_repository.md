---
title: "flux delete image repository command"
---
## flux delete image repository

Delete an ImageRepository object

### Synopsis

The delete image repository command deletes the given ImageRepository from the cluster.

```
flux delete image repository [name] [flags]
```

### Examples

```
  # Delete an image repository
  flux delete image repository alpine
```

### Options

```
  -h, --help   help for repository
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

* [flux delete image](/cmd/flux_delete_image/)	 - Delete image automation objects

