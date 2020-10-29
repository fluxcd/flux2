## gotk get helmreleases

Get HelmRelease statuses

### Synopsis

The get helmreleases command prints the statuses of the resources.

```
gotk get helmreleases [flags]
```

### Examples

```
  # List all Helm releases and their status
  gotk get helmreleases

```

### Options

```
  -h, --help   help for helmreleases
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

* [gotk get](gotk_get.md)	 - Get sources and resources

