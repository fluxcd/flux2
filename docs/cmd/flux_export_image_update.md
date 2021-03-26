---
title: "flux export image update command"
---
## flux export image update

Export ImageUpdateAutomation resources in YAML format

### Synopsis

The export image update command exports one or all ImageUpdateAutomation resources in YAML format.

```
flux export image update [name] [flags]
```

### Examples

```
  # Export all ImageUpdateAutomation resources
  flux export image update --all > updates.yaml

  # Export a specific automation
  flux export image update latest-images > latest.yaml
```

### Options

```
  -h, --help   help for update
```

### Options inherited from parent commands

```
      --all                 select all resources
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux export image](../flux_export_image/)	 - Export image automation objects

