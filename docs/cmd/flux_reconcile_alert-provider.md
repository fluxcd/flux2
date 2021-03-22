---
title: "flux reconcile alert-provider command"
---
## flux reconcile alert-provider

Reconcile a Provider

### Synopsis

The reconcile alert-provider command triggers a reconciliation of a Provider resource and waits for it to finish.

```
flux reconcile alert-provider [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing provider
  flux reconcile alert-provider slack

```

### Options

```
  -h, --help   help for alert-provider
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

* [flux reconcile](/cmd/flux_reconcile/)	 - Reconcile sources and resources

