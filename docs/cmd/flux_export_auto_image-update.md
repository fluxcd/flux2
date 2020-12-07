## flux export auto image-update

Export ImageUpdateAutomation resources in YAML format

### Synopsis

The export image-update command exports one or all ImageUpdateAutomation resources in YAML format.

```
flux export auto image-update [name] [flags]
```

### Examples

```
  # Export all ImageUpdateAutomation resources
  flux export auto image-update --all > updates.yaml

  # Export a specific automation
  flux export auto image-update latest-images > latest.yaml

```

### Options

```
  -h, --help   help for image-update
```

### Options inherited from parent commands

```
      --all                 select all resources
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux export auto](flux_export_auto.md)	 - Export automation objects

