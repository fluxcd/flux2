## flux delete source bucket

Delete a Bucket source

### Synopsis

The delete source bucket command deletes the given Bucket from the cluster.

```
flux delete source bucket [name] [flags]
```

### Examples

```
  # Delete a Bucket source
  flux delete source bucket podinfo

```

### Options

```
  -h, --help   help for bucket
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
  -s, --silent              delete resource without asking for confirmation
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux delete source](flux_delete_source.md)	 - Delete sources

