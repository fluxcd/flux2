## tk delete

Delete sources and resources

### Synopsis

The delete sub-commands delete sources and resources.

### Options

```
  -h, --help     help for delete
  -s, --silent   delete resource without asking for confirmation
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
* [tk delete kustomization](tk_delete_kustomization.md)	 - Delete a Kustomization resource
* [tk delete source](tk_delete_source.md)	 - Delete sources

