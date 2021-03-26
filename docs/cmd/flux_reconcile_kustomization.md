---
title: "flux reconcile kustomization command"
---
## flux reconcile kustomization

Reconcile a Kustomization resource

### Synopsis


The reconcile kustomization command triggers a reconciliation of a Kustomization resource and waits for it to finish.

```
flux reconcile kustomization [name] [flags]
```

### Examples

```
  # Trigger a Kustomization apply outside of the reconciliation interval
  flux reconcile kustomization podinfo

  # Trigger a sync of the Kustomization's source and apply changes
  flux reconcile kustomization podinfo --with-source
```

### Options

```
  -h, --help          help for kustomization
      --with-source   reconcile Kustomization source
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

* [flux reconcile](../flux_reconcile/)	 - Reconcile sources and resources

