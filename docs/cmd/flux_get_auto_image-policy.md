## flux get auto image-policy

Get ImagePolicy statuses

### Synopsis

The get auto image-policy command prints the status of ImagePolicy objects.

```
flux get auto image-policy [flags]
```

### Examples

```
  # List all image policies and their status
  flux get auto image-policy

 # List image policies from all namespaces
  flux get auto image-policy --all-namespaces

```

### Options

```
  -h, --help   help for image-policy
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

* [flux get auto](flux_get_auto.md)	 - Get automation statuses

