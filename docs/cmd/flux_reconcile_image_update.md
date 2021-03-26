---
title: "flux reconcile image update command"
---
## flux reconcile image update

Reconcile an ImageUpdateAutomation

### Synopsis

The reconcile image update command triggers a reconciliation of an ImageUpdateAutomation resource and waits for it to finish.

```
flux reconcile image update [name] [flags]
```

### Examples

```
  # Trigger an automation run for an existing image update automation
  flux reconcile image update latest-images
```

### Options

```
  -h, --help   help for update
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

* [flux reconcile image](../flux_reconcile_image/)	 - Reconcile image automation objects

