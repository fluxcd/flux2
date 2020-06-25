## tk suspend kustomization

Suspend reconciliation of Kustomization

### Synopsis

The suspend command disables the reconciliation of a Kustomization resource.

```
tk suspend kustomization [name] [flags]
```

### Options

```
  -h, --help   help for kustomization
```

### Options inherited from parent commands

```
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller])
      --kubeconfig string    path to the kubeconfig file (default "~/.kube/config")
      --namespace string     the namespace scope for this operation (default "gitops-system")
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
```

### SEE ALSO

* [tk suspend](tk_suspend.md)	 - Suspend resources

