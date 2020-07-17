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
  tk install --version=latest --namespace=gitops-systems

  # Dry-run install for a specific version and a series of components
  tk install --dry-run --version=v0.0.7 --components="source-controller,kustomize-controller"

  # Dry-run install with manifests preview 
  tk install --dry-run --verbose

  # Write install manifests to file 
  tk install --export > gitops-system.yaml

```

### Options

```
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
      --dry-run              only print the object that would be applied
      --export               write the install manifests to stdout and exit
  -h, --help                 help for install
      --manifests string     path to the manifest directory, dev only
  -v, --version string       toolkit version (default "latest")
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [tk](tk.md)	 - Command line utility for assembling Kubernetes CD pipelines

