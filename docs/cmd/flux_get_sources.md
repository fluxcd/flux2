---
title: "flux get sources command"
---
## flux get sources

Get source statuses

### Synopsis

The get source sub-commands print the statuses of the sources.

### Options

```
  -h, --help   help for sources
```

### Options inherited from parent commands

```
  -A, --all-namespaces      list the requested object(s) across all namespaces
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux get](/cmd/flux_get/)	 - Get the resources and their status
* [flux get sources all](/cmd/flux_get_sources_all/)	 - Get all source statuses
* [flux get sources bucket](/cmd/flux_get_sources_bucket/)	 - Get Bucket source statuses
* [flux get sources chart](/cmd/flux_get_sources_chart/)	 - Get HelmChart statuses
* [flux get sources git](/cmd/flux_get_sources_git/)	 - Get GitRepository source statuses
* [flux get sources helm](/cmd/flux_get_sources_helm/)	 - Get HelmRepository source statuses

