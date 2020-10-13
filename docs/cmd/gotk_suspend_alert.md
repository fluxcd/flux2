## gotk suspend alert

Suspend reconciliation of Alert

### Synopsis

The suspend command disables the reconciliation of a Alert resource.

```
gotk suspend alert [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing Alert
  gotk suspend alert main

```

### Options

```
  -h, --help   help for alert
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk suspend](gotk_suspend.md)	 - Suspend resources

