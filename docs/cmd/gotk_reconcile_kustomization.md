## gotk reconcile kustomization

Reconcile a Kustomization resource

### Synopsis


The reconcile kustomization command triggers a reconciliation of a Kustomization resource and waits for it to finish.

```
gotk reconcile kustomization [name] [flags]
```

### Examples

```
  # Trigger a Kustomization apply outside of the reconciliation interval
  gotk reconcile kustomization podinfo

  # Trigger a sync of the Kustomization's source and apply changes
  gotk reconcile kustomization podinfo --with-source

```

### Options

```
  -h, --help          help for kustomization
      --with-source   reconcile kustomization source
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

