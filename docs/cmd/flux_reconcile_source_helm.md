## flux reconcile source helm

Reconcile a HelmRepository source

### Synopsis

The reconcile source command triggers a reconciliation of a HelmRepository resource and waits for it to finish.

```
flux reconcile source helm [name] [flags]
```

### Examples

```
  # Trigger a reconciliation for an existing source
  flux reconcile source helm podinfo

```

### Options

```
  -h, --help   help for helm
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux reconcile source](flux_reconcile_source.md)	 - Reconcile sources

