---
title: "flux delete command"
---
## flux delete

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
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux](/cmd/flux/)	 - Command line utility for assembling Kubernetes CD pipelines
* [flux delete alert](/cmd/flux_delete_alert/)	 - Delete a Alert resource
* [flux delete alert-provider](/cmd/flux_delete_alert-provider/)	 - Delete a Provider resource
* [flux delete helmrelease](/cmd/flux_delete_helmrelease/)	 - Delete a HelmRelease resource
* [flux delete image](/cmd/flux_delete_image/)	 - Delete image automation objects
* [flux delete kustomization](/cmd/flux_delete_kustomization/)	 - Delete a Kustomization resource
* [flux delete receiver](/cmd/flux_delete_receiver/)	 - Delete a Receiver resource
* [flux delete source](/cmd/flux_delete_source/)	 - Delete sources

