## flux export

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
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux](flux.md)	 - Command line utility for assembling Kubernetes CD pipelines
* [flux export alert](flux_export_alert.md)	 - Export Alert resources in YAML format
* [flux export alert-provider](flux_export_alert-provider.md)	 - Export Provider resources in YAML format
* [flux export auto](flux_export_auto.md)	 - Export automation objects
* [flux export helmrelease](flux_export_helmrelease.md)	 - Export HelmRelease resources in YAML format
* [flux export kustomization](flux_export_kustomization.md)	 - Export Kustomization resources in YAML format
* [flux export receiver](flux_export_receiver.md)	 - Export Receiver resources in YAML format
* [flux export source](flux_export_source.md)	 - Export sources

