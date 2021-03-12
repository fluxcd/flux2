## flux export source bucket

Export Bucket sources in YAML format

### Synopsis

The export source git command exports one or all Bucket sources in YAML format.

```
flux export source bucket [name] [flags]
```

### Examples

```
  # Export all Bucket sources
  flux export source bucket --all > sources.yaml

  # Export a Bucket source including the static credentials
  flux export source bucket my-bucket --with-credentials > source.yaml

```

### Options

```
  -h, --help   help for bucket
```

### Options inherited from parent commands

```
      --all                 select all resources
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
      --with-credentials    include credential secrets
```

### SEE ALSO

* [flux export source](flux_export_source.md)	 - Export sources

