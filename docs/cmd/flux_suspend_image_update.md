## flux suspend image update

Suspend reconciliation of an ImageUpdateAutomation

### Synopsis

The suspend image update command disables the reconciliation of a ImageUpdateAutomation resource.

```
flux suspend image update [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing ImageUpdateAutomation
  flux suspend image update latest-images

```

### Options

```
  -h, --help   help for update
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

* [flux suspend image](flux_suspend_image.md)	 - Suspend image automation objects

