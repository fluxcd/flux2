## gotk suspend receiver

Suspend reconciliation of Receiver

### Synopsis

The suspend command disables the reconciliation of a Receiver resource.

```
gotk suspend receiver [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing Receiver
  gotk suspend receiver main

```

### Options

```
  -h, --help   help for receiver
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk suspend](gotk_suspend.md)	 - Suspend resources

