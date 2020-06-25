## tk check

Check requirements and installation

### Synopsis

The check command will perform a series of checks to validate that
the local environment is configured correctly and if the installed components are healthy.

```
tk check [flags]
```

### Examples

```
  # Run pre-installation checks
  check --pre

  # Run installation checks
  check

```

### Options

```
  -h, --help   help for check
      --pre    only run pre-installation checks
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

* [tk](tk.md)	 - Command line utility for assembling Kubernetes CD pipelines

