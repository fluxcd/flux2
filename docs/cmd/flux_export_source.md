## flux export source

Export sources

### Synopsis

The export source sub-commands export sources in YAML format.

### Options

```
  -h, --help               help for source
      --with-credentials   include credential secrets
```

### Options inherited from parent commands

```
      --all                 select all resources
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux export](flux_export.md)	 - Export resources in YAML format
* [flux export source bucket](flux_export_source_bucket.md)	 - Export Bucket sources in YAML format
* [flux export source git](flux_export_source_git.md)	 - Export GitRepository sources in YAML format
* [flux export source helm](flux_export_source_helm.md)	 - Export HelmRepository sources in YAML format

