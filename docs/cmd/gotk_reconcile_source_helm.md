## gotk reconcile source helm

Reconcile a HelmRepository source

### Synopsis

The reconcile source command triggers a reconciliation of a HelmRepository resource and waits for it to finish.

```
gotk reconcile source helm [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing source
  gotk reconcile source helm podinfo

```

### Options

```
  -h, --help   help for helm
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk reconcile source](gotk_reconcile_source.md)	 - Reconcile sources

