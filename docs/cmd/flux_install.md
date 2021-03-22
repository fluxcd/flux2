---
title: "flux install command"
---
## flux install

Install or upgrade Flux

### Synopsis

The install command deploys Flux in the specified namespace.
If a previous version is installed, then an in-place upgrade will be performed.

```
flux install [flags]
```

### Examples

```
  # Install the latest version in the flux-system namespace
  flux install --version=latest --namespace=flux-system

  # Install a specific version and a series of components
  flux install --dry-run --version=v0.0.7 --components="source-controller,kustomize-controller"

  # Install Flux onto tainted Kubernetes nodes
  flux install --toleration-keys=node.kubernetes.io/dedicated-to-flux

  # Dry-run install with manifests preview
  flux install --dry-run --verbose

  # Write install manifests to file
  flux install --export > flux-system.yaml

```

### Options

```
      --cluster-domain string      internal cluster domain (default "cluster.local")
      --components strings         list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
      --components-extra strings   list of components in addition to those supplied or defaulted, accepts comma-separated values
      --dry-run                    only print the object that would be applied
      --export                     write the install manifests to stdout and exit
  -h, --help                       help for install
      --image-pull-secret string   Kubernetes secret name used for pulling the toolkit images from a private registry
      --log-level logLevel         log level, available options are: (debug, info, error) (default info)
      --network-policy             deny ingress access to the toolkit controllers from other namespaces using network policies (default true)
      --registry string            container registry where the toolkit images are published (default "ghcr.io/fluxcd")
      --toleration-keys strings    list of toleration keys used to schedule the components pods onto nodes with matching taints
  -v, --version string             toolkit version, when specified the manifests are downloaded from https://github.com/fluxcd/flux2/releases
      --watch-all-namespaces       watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed (default true)
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux](/cmd/flux/)	 - Command line utility for assembling Kubernetes CD pipelines

