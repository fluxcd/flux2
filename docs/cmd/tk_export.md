## tk export

Export resources in YAML format

### Synopsis

The export sub-commands export resources in YAML format.

### Options

```
      --all    select all resources
  -h, --help   help for export
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
* [tk export kustomization](tk_export_kustomization.md)	 - Export Kustomization resources in YAML format
* [tk export source](tk_export_source.md)	 - Export sources

