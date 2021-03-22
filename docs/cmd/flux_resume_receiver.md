---
title: "flux resume receiver command"
---
## flux resume receiver

Resume a suspended Receiver

### Synopsis

The resume command marks a previously suspended Receiver resource for reconciliation and waits for it to
finish the apply.

```
flux resume receiver [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing Receiver
  flux resume receiver main

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

* [flux resume](/cmd/flux_resume/)	 - Resume suspended resources

