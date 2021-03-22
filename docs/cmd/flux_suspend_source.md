---
title: "flux suspend source command"
---
## flux suspend source

Suspend sources

### Synopsis

The suspend sub-commands suspend the reconciliation of a source.

### Options

```
  -h, --help   help for source
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux suspend](/cmd/flux_suspend/)	 - Suspend resources
* [flux suspend source bucket](/cmd/flux_suspend_source_bucket/)	 - Suspend reconciliation of a Bucket
* [flux suspend source chart](/cmd/flux_suspend_source_chart/)	 - Suspend reconciliation of a HelmChart
* [flux suspend source git](/cmd/flux_suspend_source_git/)	 - Suspend reconciliation of a GitRepository
* [flux suspend source helm](/cmd/flux_suspend_source_helm/)	 - Suspend reconciliation of a HelmRepository

