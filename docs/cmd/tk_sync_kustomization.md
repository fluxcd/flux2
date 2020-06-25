## tk sync kustomization

Synchronize a Kustomization resource

### Synopsis


The sync kustomization command triggers a reconciliation of a Kustomization resource and waits for it to finish.

```
tk sync kustomization [name] [flags]
```

### Examples

```
  # Trigger a Kustomization apply outside of the reconciliation interval
  sync kustomization podinfo

  # Trigger a sync of the Kustomization's source and apply changes
  sync kustomization podinfo --with-source

```

### Options

```
  -h, --help          help for kustomization
      --with-source   synchronize kustomization source
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

