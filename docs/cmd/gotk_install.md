## gotk install

Install the toolkit components

### Synopsis

The install command deploys the toolkit components in the specified namespace.
If a previous version is installed, then an in-place upgrade will be performed.

```
gotk install [flags]
```

### Examples

```
  # Install the latest version in the gotk-system namespace
  gotk install --version=latest --namespace=gotk-system

  # Dry-run install for a specific version and a series of components
  gotk install --dry-run --version=v0.0.7 --components="source-controller,kustomize-controller"

  # Dry-run install with manifests preview
  gotk install --dry-run --verbose

  # Write install manifests to file
  gotk install --export > gotk-system.yaml

```

### Options

```
      --arch arch                  cluster architecture, available options are: (amd64, arm, arm64) (default amd64)
      --components strings         list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
      --dry-run                    only print the object that would be applied
      --export                     write the install manifests to stdout and exit
  -h, --help                       help for install
      --image-pull-secret string   Kubernetes secret name used for pulling the toolkit images from a private registry
      --log-level logLevel         log level, available options are: (debug, info, error) (default info)
      --network-policy             deny ingress access to the toolkit controllers from other namespaces using network policies (default true)
      --registry string            container registry where the toolkit images are published (default "ghcr.io/fluxcd")
  -v, --version string             toolkit version (default "latest")
      --watch-all-namespaces       watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed (default true)
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk](gotk.md)	 - Command line utility for assembling Kubernetes CD pipelines

