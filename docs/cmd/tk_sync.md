## tk sync

Synchronize sources and resources

### Synopsis

The sync sub-commands trigger a reconciliation of sources and resources.

### Options

```
  -h, --help   help for sync
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
* [tk sync kustomization](tk_sync_kustomization.md)	 - Synchronize a Kustomization resource
* [tk sync source](tk_sync_source.md)	 - Synchronize sources

