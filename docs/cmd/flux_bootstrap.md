## flux bootstrap

Bootstrap toolkit components

### Synopsis

The bootstrap sub-commands bootstrap the toolkit components on the targeted Git provider.

### Options

```
      --arch arch                  cluster architecture, available options are: (amd64, arm, arm64) (default amd64)
      --branch string              default branch (for GitHub this must match the default branch setting for the organization) (default "main")
      --components strings         list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
      --components-extra strings   list of components in addition to those supplied or defaulted, accepts comma-separated values
  -h, --help                       help for bootstrap
      --image-pull-secret string   Kubernetes secret name used for pulling the toolkit images from a private registry
      --log-level logLevel         log level, available options are: (debug, info, error) (default info)
      --network-policy             deny ingress access to the toolkit controllers from other namespaces using network policies (default true)
      --registry string            container registry where the toolkit images are published (default "ghcr.io/fluxcd")
      --token-auth                 when enabled, the personal access token will be used instead of SSH deploy key
  -v, --version string             toolkit version (default "latest")
      --watch-all-namespaces       watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed (default true)
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux](flux.md)	 - Command line utility for assembling Kubernetes CD pipelines
* [flux bootstrap github](flux_bootstrap_github.md)	 - Bootstrap toolkit components in a GitHub repository
* [flux bootstrap gitlab](flux_bootstrap_gitlab.md)	 - Bootstrap toolkit components in a GitLab repository

