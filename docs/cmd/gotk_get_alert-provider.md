## gotk get alert-provider

Get Provider statuses

### Synopsis

The get alert-provider command prints the statuses of the resources.

```
gotk get alert-provider [flags]
```

### Examples

```
  # List all Providers and their status
  gotk get alert-provider

```

### Options

```
  -h, --help   help for alert-provider
```

### Options inherited from parent commands

```
  -A, --all-namespaces      list the requested object(s) across all namespaces
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk get](gotk_get.md)	 - Get sources and resources

