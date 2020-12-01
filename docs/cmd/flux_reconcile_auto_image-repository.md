## flux reconcile auto image-repository

Reconcile an ImageRepository

### Synopsis

The reconcile auto image-repository command triggers a reconciliation of an ImageRepository resource and waits for it to finish.

```
flux reconcile auto image-repository [name] [flags]
```

### Examples

```
  # Trigger an scan for an existing image repository
  flux reconcile auto image-repository alpine

```

### Options

```
  -h, --help   help for image-repository
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

* [flux reconcile auto](flux_reconcile_auto.md)	 - Reconcile automation objects

