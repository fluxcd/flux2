## flux get alerts

Get Alert statuses

### Synopsis

The get alert command prints the statuses of the resources.

```
flux get alerts [flags]
```

### Examples

```
  # List all Alerts and their status
  flux get alerts

```

### Options

```
  -h, --help   help for alerts
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

* [flux get](flux_get.md)	 - Get sources and resources

