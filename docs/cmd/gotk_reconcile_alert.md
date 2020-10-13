## gotk reconcile alert

Reconcile an Alert

### Synopsis

The reconcile alert command triggers a reconciliation of an Alert resource and waits for it to finish.

```
gotk reconcile alert [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing alert
  gotk reconcile alert main

```

### Options

```
  -h, --help   help for alert
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

