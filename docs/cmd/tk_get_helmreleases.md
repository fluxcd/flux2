## tk get helmreleases

Get HelmRelease statuses

### Synopsis

The get helmreleases command prints the statuses of the resources.

```
tk get helmreleases [flags]
```

### Examples

```
  # List all Helm releases and their status
  tk get helmreleases

```

### Options

```
  -h, --help   help for helmreleases
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [tk get](tk_get.md)	 - Get sources and resources

