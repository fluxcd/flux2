## tk sync source git

Synchronize a GitRepository source

### Synopsis

The sync source command triggers a reconciliation of a GitRepository resource and waits for it to finish.

```
tk sync source git [name] [flags]
```

### Examples

```
  # Trigger a git pull for an existing source
  sync source git podinfo

```

### Options

```
  -h, --help   help for git
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

* [tk sync source](tk_sync_source.md)	 - Synchronize sources

