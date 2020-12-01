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
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux](flux.md)	 - Command line utility for assembling Kubernetes CD pipelines
* [flux delete alert](flux_delete_alert.md)	 - Delete a Alert resource
* [flux delete alert-provider](flux_delete_alert-provider.md)	 - Delete a Provider resource
* [flux delete auto](flux_delete_auto.md)	 - Delete automation objects
* [flux delete helmrelease](flux_delete_helmrelease.md)	 - Delete a HelmRelease resource
* [flux delete kustomization](flux_delete_kustomization.md)	 - Delete a Kustomization resource
* [flux delete receiver](flux_delete_receiver.md)	 - Delete a Receiver resource
* [flux delete source](flux_delete_source.md)	 - Delete sources

