## flux get

Get sources and resources

### Synopsis

The get sub-commands print the statuses of sources and resources.

### Options

```
  -A, --all-namespaces   list the requested object(s) across all namespaces
  -h, --help             help for get
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
* [flux get alert-providers](flux_get_alert-providers.md)	 - Get Provider statuses
* [flux get alerts](flux_get_alerts.md)	 - Get Alert statuses
* [flux get auto](flux_get_auto.md)	 - Get automation statuses
* [flux get helmreleases](flux_get_helmreleases.md)	 - Get HelmRelease statuses
* [flux get kustomizations](flux_get_kustomizations.md)	 - Get Kustomization statuses
* [flux get receivers](flux_get_receivers.md)	 - Get Receiver statuses
* [flux get sources](flux_get_sources.md)	 - Get source statuses

