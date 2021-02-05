## flux get images policy

Get ImagePolicy status

### Synopsis

The get image policy command prints the status of ImagePolicy objects.

```
flux get images policy [flags]
```

### Examples

```
  # List all image policies and their status
  flux get image policy

 # List image policies from all namespaces
  flux get image policy --all-namespaces

```

### Options

```
  -h, --help   help for policy
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

