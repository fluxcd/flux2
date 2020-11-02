## flux resume helmrelease

Resume a suspended HelmRelease

### Synopsis

The resume command marks a previously suspended HelmRelease resource for reconciliation and waits for it to
finish the apply.

```
flux resume helmrelease [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing Helm release
  flux resume hr podinfo

```

### Options

```
  -h, --help   help for helmrelease
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

* [flux resume](flux_resume.md)	 - Resume suspended resources

