## gotk get kustomizations

Get Kustomization statuses

### Synopsis

The get kustomizations command prints the statuses of the resources.

```
gotk get kustomizations [flags]
```

### Examples

```
  # List all kustomizations and their status
  gotk get kustomizations

```

### Options

```
  -h, --help   help for kustomizations
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk get](gotk_get.md)	 - Get sources and resources

