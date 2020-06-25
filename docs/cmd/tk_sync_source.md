## tk sync source

Synchronize sources

### Synopsis

The sync source sub-commands trigger a reconciliation of sources.

### Options

```
  -h, --help   help for source
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

* [tk sync](tk_sync.md)	 - Synchronize sources and resources
* [tk sync source git](tk_sync_source_git.md)	 - Synchronize a GitRepository source

