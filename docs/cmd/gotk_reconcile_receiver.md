## gotk reconcile receiver

Reconcile a Receiver

### Synopsis

The reconcile receiver command triggers a reconciliation of a Receiver resource and waits for it to finish.

```
gotk reconcile receiver [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing receiver
  gotk reconcile receiver main

```

### Options

```
  -h, --help   help for receiver
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

