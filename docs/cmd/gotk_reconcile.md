## gotk reconcile

Reconcile sources and resources

### Synopsis

The reconcile sub-commands trigger a reconciliation of sources and resources.

### Options

```
  -h, --help   help for reconcile
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk](gotk.md)	 - Command line utility for assembling Kubernetes CD pipelines
* [gotk reconcile helmrelease](gotk_reconcile_helmrelease.md)	 - Reconcile a HelmRelease resource
* [gotk reconcile kustomization](gotk_reconcile_kustomization.md)	 - Reconcile a Kustomization resource
* [gotk reconcile source](gotk_reconcile_source.md)	 - Reconcile sources

