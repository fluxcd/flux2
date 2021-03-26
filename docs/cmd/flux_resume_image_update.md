---
title: "flux resume image update command"
---
## flux resume image update

Resume a suspended ImageUpdateAutomation

### Synopsis

The resume command marks a previously suspended ImageUpdateAutomation resource for reconciliation and waits for it to finish.

```
flux resume image update [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing ImageUpdateAutomation
  flux resume image update latest-images
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

* [flux resume image](../flux_resume_image/)	 - Resume image automation objects

