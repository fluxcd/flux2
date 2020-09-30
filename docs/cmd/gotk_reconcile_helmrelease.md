## gotk reconcile helmrelease

Reconcile a HelmRelease resource

### Synopsis


The reconcile kustomization command triggers a reconciliation of a HelmRelease resource and waits for it to finish.

```
gotk reconcile helmrelease [name] [flags]
```

### Examples

```
  # Trigger a HelmRelease apply outside of the reconciliation interval
  gotk reconcile hr podinfo

  # Trigger a reconciliation of the HelmRelease's source and apply changes
  gotk reconcile hr podinfo --with-source

```

### Options

```
  -h, --help          help for helmrelease
      --with-source   reconcile HelmRelease source
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk reconcile](gotk_reconcile.md)	 - Reconcile sources and resources

