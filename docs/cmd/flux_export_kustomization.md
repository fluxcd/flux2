## flux export kustomization

Export Kustomization resources in YAML format

### Synopsis

The export kustomization command exports one or all Kustomization resources in YAML format.

```
flux export kustomization [name] [flags]
```

### Examples

```
  # Export all Kustomization resources
  flux export kustomization --all > kustomizations.yaml

  # Export a Kustomization
  flux export kustomization my-app > kustomization.yaml

```

### Options

```
  -h, --help   help for kustomization
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

* [flux export](flux_export.md)	 - Export resources in YAML format

