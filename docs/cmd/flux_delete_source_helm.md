## flux delete source helm

Delete a HelmRepository source

### Synopsis

The delete source helm command deletes the given HelmRepository from the cluster.

```
flux delete source helm [name] [flags]
```

### Examples

```
  # Delete a Helm repository
  flux delete source helm podinfo

```

### Options

```
  -h, --help   help for helm
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
  -s, --silent              delete resource without asking for confirmation
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux delete source](flux_delete_source.md)	 - Delete sources

