## gotk export source helm

Export HelmRepository sources in YAML format

### Synopsis

The export source git command exports on or all HelmRepository sources in YAML format.

```
gotk export source helm [name] [flags]
```

### Examples

```
  # Export all HelmRepository sources
  gotk export source helm --all > sources.yaml

  # Export a HelmRepository source including the basic auth credentials
  gotk export source helm my-private-repo --with-credentials > source.yaml

```

### Options

```
  -h, --help   help for helm
```

### Options inherited from parent commands

```
      --all                 select all resources
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
      --with-credentials    include credential secrets
```

### SEE ALSO

* [gotk export source](gotk_export_source.md)	 - Export sources

