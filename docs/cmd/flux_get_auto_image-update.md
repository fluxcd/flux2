## flux get auto image-update

Get ImageUpdateAutomation statuses

### Synopsis

The get auto image-update command prints the status of ImageUpdateAutomation objects.

```
flux get auto image-update [flags]
```

### Examples

```
  # List all image update automation object and their status
  flux get auto image-update

 # List image update automations from all namespaces
  flux get auto image-update --all-namespaces

```

### Options

```
  -h, --help   help for image-update
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

* [flux get auto](flux_get_auto.md)	 - Get automation statuses

