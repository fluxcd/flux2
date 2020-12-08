## flux reconcile auto image-update

Reconcile an ImageUpdateAutomation

### Synopsis

The reconcile auto image-update command triggers a reconciliation of an ImageUpdateAutomation resource and waits for it to finish.

```
flux reconcile auto image-update [name] [flags]
```

### Examples

```
  # Trigger an automation run for an existing image update automation
  flux reconcile auto image-update latest-images

```

### Options

```
  -h, --help   help for image-update
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

