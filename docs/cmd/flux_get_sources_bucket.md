## flux get sources bucket

Get Bucket source statuses

### Synopsis

The get sources bucket command prints the status of the Bucket sources.

```
flux get sources bucket [flags]
```

### Examples

```
  # List all Buckets and their status
  flux get sources bucket

```

### Options

```
  -h, --help   help for bucket
```

### Options inherited from parent commands

```
  -A, --all-namespaces      list the requested object(s) across all namespaces
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux get sources](flux_get_sources.md)	 - Get source statuses

