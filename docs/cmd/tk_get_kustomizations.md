## tk get kustomizations

Get Kustomization source statuses

### Synopsis

The get kustomizations command prints the statuses of the resources.

```
tk get kustomizations [flags]
```

### Options

```
  -h, --help   help for kustomizations
```

### Options inherited from parent commands

```
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller])
      --kubeconfig string    path to the kubeconfig file (default "~/.kube/config")
      --namespace string     the namespace scope for this operation (default "gitops-system")
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
```

### SEE ALSO

* [tk get](tk_get.md)	 - Get sources and resources

