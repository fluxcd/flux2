## gotk export alert-provider

Export Provider resources in YAML format

### Synopsis

The export alert-provider command exports one or all Provider resources in YAML format.

```
gotk export alert-provider [name] [flags]
```

### Examples

```
  # Export all Provider resources
  gotk export ap --all > alert-providers.yaml

  # Export a Provider
  gotk export ap slack > slack.yaml

```

### Options

```
  -h, --help   help for alert-provider
```

### Options inherited from parent commands

```
      --all                 select all resources
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk export](gotk_export.md)	 - Export resources in YAML format

