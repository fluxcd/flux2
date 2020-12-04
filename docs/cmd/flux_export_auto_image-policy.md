## flux export auto image-policy

Export ImagePolicy resources in YAML format

### Synopsis

The export image-policy command exports one or all ImagePolicy resources in YAML format.

```
flux export auto image-policy [name] [flags]
```

### Examples

```
  # Export all ImagePolicy resources
  flux export auto image-policy --all > image-policies.yaml

  # Export a Provider
  flux export auto image-policy alpine1x > alpine1x.yaml

```

### Options

```
  -h, --help   help for image-policy
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

