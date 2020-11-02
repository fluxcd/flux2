## flux suspend alert

Suspend reconciliation of Alert

### Synopsis

The suspend command disables the reconciliation of a Alert resource.

```
flux suspend alert [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing Alert
  flux suspend alert main

```

### Options

```
  -h, --help   help for alert
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

* [flux suspend](flux_suspend.md)	 - Suspend resources

