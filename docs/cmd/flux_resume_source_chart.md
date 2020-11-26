## flux resume source chart

Resume a suspended HelmChart

### Synopsis

The resume command marks a previously suspended HelmChart resource for reconciliation and waits for it to finish.

```
flux resume source chart [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing HelmChart
  flux resume source chart podinfo

```

### Options

```
  -h, --help   help for chart
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

* [flux resume source](flux_resume_source.md)	 - Resume sources

