## flux export source git

Export GitRepository sources in YAML format

### Synopsis

The export source git command exports on or all GitRepository sources in YAML format.

```
flux export source git [name] [flags]
```

### Examples

```
  # Export all GitRepository sources
  flux export source git --all > sources.yaml

  # Export a GitRepository source including the SSH key pair or basic auth credentials
  flux export source git my-private-repo --with-credentials > source.yaml

```

### Options

```
  -h, --help   help for git
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

