## flux export auto image-repository

Export ImageRepository resources in YAML format

### Synopsis

The export image-repository command exports one or all ImageRepository resources in YAML format.

```
flux export auto image-repository [name] [flags]
```

### Examples

```
  # Export all ImageRepository resources
  flux export auto image-repository --all > image-repositories.yaml

  # Export a Provider
  flux export auto image-repository alpine > alpine.yaml

```

### Options

```
  -h, --help   help for image-repository
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

