## tk install

Install the toolkit components

### Synopsis

The install command deploys the toolkit components in the specified namespace.
If a previous version is installed, then an in-place upgrade will be performed.

```
tk install [flags]
```

### Examples

```
  # Install the latest version in the gitops-systems namespace
  install --version=master --namespace=gitops-systems

  # Dry-run install for a specific version and a series of components
  install --dry-run --version=0.0.1 --components="source-controller,kustomize-controller"

  # Dry-run install with manifests preview 
  install --dry-run --verbose

```

### Options

```
      --dry-run            only print the object that would be applied
  -h, --help               help for install
      --manifests string   path to the manifest directory, dev only
  -v, --version string     toolkit tag or branch (default "master")
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

