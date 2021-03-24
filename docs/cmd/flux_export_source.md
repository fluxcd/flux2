---
title: "flux export source command"
---
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
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux export](/cmd/flux_export/)	 - Export resources in YAML format
* [flux export source bucket](/cmd/flux_export_source_bucket/)	 - Export Bucket sources in YAML format
* [flux export source git](/cmd/flux_export_source_git/)	 - Export GitRepository sources in YAML format
* [flux export source helm](/cmd/flux_export_source_helm/)	 - Export HelmRepository sources in YAML format

