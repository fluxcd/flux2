## gotk export source bucket

Export Bucket sources in YAML format

### Synopsis

The export source git command exports on or all Bucket sources in YAML format.

```
gotk export source bucket [name] [flags]
```

### Examples

```
  # Export all Bucket sources
  gotk export source bucket --all > sources.yaml

  # Export a Bucket source including the static credentials
  gotk export source bucket my-bucket --with-credentials > source.yaml

```

### Options

```
  -h, --help   help for bucket
```

### Options inherited from parent commands

```
      --all                 select all resources
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
      --with-credentials    include credential secrets
```

### SEE ALSO

* [gotk export source](gotk_export_source.md)	 - Export sources

