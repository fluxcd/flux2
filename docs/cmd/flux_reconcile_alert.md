---
title: "flux reconcile alert command"
---
## flux reconcile alert

Reconcile an Alert

### Synopsis

The reconcile alert command triggers a reconciliation of an Alert resource and waits for it to finish.

```
flux reconcile alert [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing alert
  flux reconcile alert main
```

### Options

```
  -h, --help   help for alert
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

