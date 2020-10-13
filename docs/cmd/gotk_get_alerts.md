## gotk get alerts

Get Alert statuses

### Synopsis

The get alert command prints the statuses of the resources.

```
gotk get alerts [flags]
```

### Examples

```
  # List all Alerts and their status
  gotk get alerts

```

### Options

```
  -h, --help   help for alerts
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

