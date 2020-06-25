## tk delete source git

Delete a GitRepository source

### Synopsis

The delete source git command deletes the given GitRepository from the cluster.

```
tk delete source git [name] [flags]
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
  -s, --silent               delete resource without asking for confirmation
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
```

### SEE ALSO

* [tk delete source](tk_delete_source.md)	 - Delete sources

