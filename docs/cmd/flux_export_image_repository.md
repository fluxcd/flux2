---
title: "flux export image repository command"
---
## flux export image repository

Export ImageRepository resources in YAML format

### Synopsis

The export image repository command exports one or all ImageRepository resources in YAML format.

```
flux export image repository [name] [flags]
```

### Examples

```
  # Export all ImageRepository resources
  flux export image repository --all > image-repositories.yaml

  # Export a specific ImageRepository resource
  flux export image repository alpine > alpine.yaml

```

### Options

```
  -h, --help   help for repository
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

* [flux export image](/cmd/flux_export_image/)	 - Export image automation objects

