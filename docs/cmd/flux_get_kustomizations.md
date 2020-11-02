## flux get kustomizations

Get Kustomization statuses

### Synopsis

The get kustomizations command prints the statuses of the resources.

```
flux get kustomizations [flags]
```

### Examples

```
  # List all kustomizations and their status
  flux get kustomizations

```

### Options

```
  -h, --help   help for kustomizations
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

* [flux get](flux_get.md)	 - Get sources and resources

