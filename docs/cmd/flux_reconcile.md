## flux reconcile

Reconcile sources and resources

### Synopsis

The reconcile sub-commands trigger a reconciliation of sources and resources.

### Options

```
  -h, --help   help for reconcile
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
* [flux reconcile alert](flux_reconcile_alert.md)	 - Reconcile an Alert
* [flux reconcile alert-provider](flux_reconcile_alert-provider.md)	 - Reconcile a Provider
* [flux reconcile auto](flux_reconcile_auto.md)	 - Reconcile automation objects
* [flux reconcile helmrelease](flux_reconcile_helmrelease.md)	 - Reconcile a HelmRelease resource
* [flux reconcile kustomization](flux_reconcile_kustomization.md)	 - Reconcile a Kustomization resource
* [flux reconcile receiver](flux_reconcile_receiver.md)	 - Reconcile a Receiver
* [flux reconcile source](flux_reconcile_source.md)	 - Reconcile sources

