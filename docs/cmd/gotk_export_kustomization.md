## gotk export kustomization

Export Kustomization resources in YAML format

### Synopsis

The export kustomization command exports one or all Kustomization resources in YAML format.

```
gotk export kustomization [name] [flags]
```

### Examples

```
  # Export all Kustomization resources
  gotk export kustomization --all > kustomizations.yaml

  # Export a Kustomization
  gotk export kustomization my-app > kustomization.yaml

```

### Options

```
  -h, --help   help for kustomization
```

### Options inherited from parent commands

```
      --all                 select all resources
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk export](gotk_export.md)	 - Export resources in YAML format

