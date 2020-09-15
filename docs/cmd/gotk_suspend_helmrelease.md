## gotk suspend helmrelease

Suspend reconciliation of HelmRelease

### Synopsis

The suspend command disables the reconciliation of a HelmRelease resource.

```
gotk suspend helmrelease [name] [flags]
```

### Examples

```
  # Suspend reconciliation for an existing Helm release
  gotk suspend hr podinfo

```

### Options

```
  -h, --help   help for helmrelease
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

