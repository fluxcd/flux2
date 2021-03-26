---
title: "flux reconcile image repository command"
---
## flux reconcile image repository

Reconcile an ImageRepository

### Synopsis

The reconcile image repository command triggers a reconciliation of an ImageRepository resource and waits for it to finish.

```
flux reconcile image repository [name] [flags]
```

### Examples

```
  # Trigger an scan for an existing image repository
  flux reconcile image repository alpine
```

### Options

```
  -h, --help   help for repository
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

* [flux reconcile image](/cmd/flux_reconcile_image/)	 - Reconcile image automation objects

