## gotk export source git

Export GitRepository sources in YAML format

### Synopsis

The export source git command exports on or all GitRepository sources in YAML format.

```
gotk export source git [name] [flags]
```

### Examples

```
  # Export all GitRepository sources
  gotk export source git --all > sources.yaml

  # Export a GitRepository source including the SSH key pair or basic auth credentials
  gotk export source git my-private-repo --with-credentials > source.yaml

```

### Options

```
  -h, --help   help for git
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

