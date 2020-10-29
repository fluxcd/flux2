## gotk reconcile source bucket

Reconcile a Bucket source

### Synopsis

The reconcile source command triggers a reconciliation of a Bucket resource and waits for it to finish.

```
gotk reconcile source bucket [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing source
  gotk reconcile source bucket podinfo

```

### Options

```
  -h, --help   help for bucket
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk reconcile source](gotk_reconcile_source.md)	 - Reconcile sources

