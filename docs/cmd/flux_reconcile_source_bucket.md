## flux reconcile source bucket

Reconcile a Bucket source

### Synopsis

The reconcile source command triggers a reconciliation of a Bucket resource and waits for it to finish.

```
flux reconcile source bucket [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing source
  flux reconcile source bucket podinfo

```

### Options

```
  -h, --help   help for bucket
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux reconcile source](flux_reconcile_source.md)	 - Reconcile sources

