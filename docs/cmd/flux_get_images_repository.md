## flux get images repository

Get ImageRepository status

### Synopsis

The get image repository command prints the status of ImageRepository objects.

```
flux get images repository [flags]
```

### Examples

```
  # List all image repositories and their status
  flux get image repository

 # List image repositories from all namespaces
  flux get image repository --all-namespaces

```

### Options

```
  -h, --help   help for repository
```

### Options inherited from parent commands

```
  -A, --all-namespaces      list the requested object(s) across all namespaces
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux get images](flux_get_images.md)	 - Get image automation object status

