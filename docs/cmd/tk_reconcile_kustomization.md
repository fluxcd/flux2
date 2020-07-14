## tk reconcile kustomization

Reconcile a Kustomization resource

### Synopsis


The reconcile kustomization command triggers a reconciliation of a Kustomization resource and waits for it to finish.

```
tk reconcile kustomization [name] [flags]
```

### Examples

```
  # Trigger a Kustomization apply outside of the reconciliation interval
  tk reconcile kustomization podinfo

  # Trigger a sync of the Kustomization's source and apply changes
  tk reconcile kustomization podinfo --with-source

```

### Options

```
  -h, --help          help for kustomization
      --with-source   reconcile kustomization source
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [tk reconcile](tk_reconcile.md)	 - Reconcile sources and resources

