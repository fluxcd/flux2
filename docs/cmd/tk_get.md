## tk get

Get sources and resources

### Synopsis

The get sub-commands print the statuses of sources and resources.

### Options

```
  -h, --help   help for get
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
* [tk get kustomizations](tk_get_kustomizations.md)	 - Get Kustomization source statuses
* [tk get sources](tk_get_sources.md)	 - Get source statuses

