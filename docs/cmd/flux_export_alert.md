## flux export alert

Export Alert resources in YAML format

### Synopsis

The export alert command exports one or all Alert resources in YAML format.

```
flux export alert [name] [flags]
```

### Examples

```
  # Export all Alert resources
  flux export alert --all > alerts.yaml

  # Export a Alert
  flux export alert main > main.yaml

```

### Options

```
  -h, --help   help for alert
```

### Options inherited from parent commands

```
      --all                 select all resources
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux export](flux_export.md)	 - Export resources in YAML format

