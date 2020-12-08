## flux resume auto image-update

Resume a suspended ImageUpdateAutomation

### Synopsis

The resume command marks a previously suspended ImageUpdateAutomation resource for reconciliation and waits for it to finish.

```
flux resume auto image-update [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing ImageUpdateAutomation
  flux resume auto image-update latest-images

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

* [flux resume auto](flux_resume_auto.md)	 - Resume automation objects

