## gotk reconcile alert-provider

Reconcile a Provider

### Synopsis

The reconcile alert-provider command triggers a reconciliation of a Provider resource and waits for it to finish.

```
gotk reconcile alert-provider [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing provider
  gotk reconcile alert-provider slack

```

### Options

```
  -h, --help   help for alert-provider
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

