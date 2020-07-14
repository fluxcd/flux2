## tk suspend helmrelease

Suspend reconciliation of HelmRelease

### Synopsis

The suspend command disables the reconciliation of a HelmRelease resource.

```
tk suspend helmrelease [name] [flags]
```

### Options

```
  -h, --help   help for helmrelease
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [tk suspend](tk_suspend.md)	 - Suspend resources

