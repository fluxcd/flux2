## tk export kustomization

Export Kustomization resources in YAML format

### Synopsis

The export kustomization command exports one or all Kustomization resources in YAML format.

```
tk export kustomization [name] [flags]
```

### Examples

```
  # Export all Kustomization resources
  export kustomization --all > kustomizations.yaml

  # Export a Kustomization
  export kustomization my-app > kustomization.yaml

```

### Options

```
  -h, --help   help for kustomization
```

### Options inherited from parent commands

```
      --all                  select all resources
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller])
      --kubeconfig string    path to the kubeconfig file (default "~/.kube/config")
      --namespace string     the namespace scope for this operation (default "gitops-system")
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
```

### SEE ALSO

* [tk export](tk_export.md)	 - Export resources in YAML format

