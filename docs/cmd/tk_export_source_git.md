## tk export source git

Export GitRepository sources in YAML format

### Synopsis

The export source git command exports on or all GitRepository sources in YAML format.

```
tk export source git [name] [flags]
```

### Examples

```
  # Export all GitRepository sources
  export source git --all > sources.yaml

  # Export a GitRepository source including the SSH key pair or basic auth credentials
  export source git my-private-repo --with-credentials > source.yaml

```

### Options

```
  -h, --help   help for git
```

### Options inherited from parent commands

```
      --all                  select all resources
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller])
      --kubeconfig string    path to the kubeconfig file (default "~/.kube/config")
      --namespace string     the namespace scope for this operation (default "gitops-system")
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
      --with-credentials     include credential secrets
```

### SEE ALSO

* [tk export source](tk_export_source.md)	 - Export sources

