## gotk get receivers

Get Receiver statuses

### Synopsis

The get receiver command prints the statuses of the resources.

```
gotk get receivers [flags]
```

### Examples

```
  # List all Receiver and their status
  gotk get receivers

```

### Options

```
  -h, --help   help for receivers
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

