---
title: "flux export image policy command"
---
## flux export image policy

Export ImagePolicy resources in YAML format

### Synopsis

The export image policy command exports one or all ImagePolicy resources in YAML format.

```
flux export image policy [name] [flags]
```

### Examples

```
  # Export all ImagePolicy resources
  flux export image policy --all > image-policies.yaml

  # Export a specific policy
  flux export image policy alpine1x > alpine1x.yaml
```

### Options

```
  -h, --help   help for policy
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

