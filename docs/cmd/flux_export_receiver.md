## flux export receiver

Export Receiver resources in YAML format

### Synopsis

The export receiver command exports one or all Receiver resources in YAML format.

```
flux export receiver [name] [flags]
```

### Examples

```
  # Export all Receiver resources
  flux export receiver --all > receivers.yaml

  # Export a Receiver
  flux export receiver main > main.yaml

```

### Options

```
  -h, --help   help for receiver
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

