## tk export helmrelease

Export HelmRelease resources in YAML format

### Synopsis

The export helmrelease command exports one or all HelmRelease resources in YAML format.

```
tk export helmrelease [name] [flags]
```

### Examples

```
  # Export all HelmRelease resources
  tk export helmrelease --all > kustomizations.yaml

  # Export a HelmRelease
  tk export hr my-app > app-release.yaml

```

### Options

```
  -h, --help   help for helmrelease
```

### Options inherited from parent commands

```
      --all                 select all resources
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [tk export](tk_export.md)	 - Export resources in YAML format

