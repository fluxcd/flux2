## tk suspend

Suspend resources

### Synopsis

The suspend sub-commands suspend the reconciliation of a resource.

### Options

```
  -h, --help   help for suspend
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

* [tk](tk.md)	 - Command line utility for assembling Kubernetes CD pipelines
* [tk suspend kustomization](tk_suspend_kustomization.md)	 - Suspend reconciliation of Kustomization

