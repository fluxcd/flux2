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

* [flux](../flux/)	 - Command line utility for assembling Kubernetes CD pipelines
* [flux delete alert](../flux_delete_alert/)	 - Delete a Alert resource
* [flux delete alert-provider](../flux_delete_alert-provider/)	 - Delete a Provider resource
* [flux delete helmrelease](../flux_delete_helmrelease/)	 - Delete a HelmRelease resource
* [flux delete image](../flux_delete_image/)	 - Delete image automation objects
* [flux delete kustomization](../flux_delete_kustomization/)	 - Delete a Kustomization resource
* [flux delete receiver](../flux_delete_receiver/)	 - Delete a Receiver resource
* [flux delete source](../flux_delete_source/)	 - Delete sources

