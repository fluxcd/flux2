## tk delete source

Delete sources

### Synopsis

The delete source sub-commands delete sources.

### Options

```
  -h, --help   help for source
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

* [tk delete](tk_delete.md)	 - Delete sources and resources
* [tk delete source git](tk_delete_source_git.md)	 - Delete a GitRepository source

