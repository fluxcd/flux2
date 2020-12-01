## flux suspend auto image-repository

Suspend reconciliation of an ImageRepository

### Synopsis

The suspend command disables the reconciliation of a ImageRepository resource.

```
flux suspend auto image-repository [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing ImageRepository
  flux suspend auto image-repository alpine

```

### Options

```
  -h, --help   help for image-repository
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux suspend auto](flux_suspend_auto.md)	 - Suspend automation objects

