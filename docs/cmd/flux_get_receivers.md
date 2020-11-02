## flux get receivers

Get Receiver statuses

### Synopsis

The get receiver command prints the statuses of the resources.

```
flux get receivers [flags]
```

### Examples

```
  # List all Receiver and their status
  flux get receivers

```

### Options

```
  -h, --help   help for receivers
```

### Options inherited from parent commands

```
  -A, --all-namespaces      list the requested object(s) across all namespaces
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux get](flux_get.md)	 - Get sources and resources

