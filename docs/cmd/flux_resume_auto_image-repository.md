## flux resume auto image-repository

Resume a suspended ImageRepository

### Synopsis

The resume command marks a previously suspended ImageRepository resource for reconciliation and waits for it to finish.

```
flux resume auto image-repository [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing ImageRepository
  flux resume auto image-repository alpine

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

* [flux resume auto](flux_resume_auto.md)	 - Resume automation objects

