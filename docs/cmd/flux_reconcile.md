---
title: "flux reconcile command"
---
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
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux](/cmd/flux/)	 - Command line utility for assembling Kubernetes CD pipelines
* [flux reconcile alert](/cmd/flux_reconcile_alert/)	 - Reconcile an Alert
* [flux reconcile alert-provider](/cmd/flux_reconcile_alert-provider/)	 - Reconcile a Provider
* [flux reconcile helmrelease](/cmd/flux_reconcile_helmrelease/)	 - Reconcile a HelmRelease resource
* [flux reconcile image](/cmd/flux_reconcile_image/)	 - Reconcile image automation objects
* [flux reconcile kustomization](/cmd/flux_reconcile_kustomization/)	 - Reconcile a Kustomization resource
* [flux reconcile receiver](/cmd/flux_reconcile_receiver/)	 - Reconcile a Receiver
* [flux reconcile source](/cmd/flux_reconcile_source/)	 - Reconcile sources

