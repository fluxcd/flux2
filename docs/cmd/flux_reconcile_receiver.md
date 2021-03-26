---
title: "flux reconcile receiver command"
---
## flux reconcile receiver

Reconcile a Receiver

### Synopsis

The reconcile receiver command triggers a reconciliation of a Receiver resource and waits for it to finish.

```
flux reconcile receiver [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing receiver
  flux reconcile receiver main
```

### Options

```
  -h, --help   help for receiver
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

