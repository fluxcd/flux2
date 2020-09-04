## gotk bootstrap

Bootstrap toolkit components

### Synopsis

The bootstrap sub-commands bootstrap the toolkit components on the targeted Git provider.

### Options

```
      --arch string                arch can be amd64 or arm64 (default "amd64")
      --components strings         list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
  -h, --help                       help for bootstrap
      --image-pull-secret string   Kubernetes secret name used for pulling the toolkit images from a private registry
      --registry string            container registry where the toolkit images are published (default "ghcr.io/fluxcd")
  -v, --version string             toolkit version (default "latest")
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk](gotk.md)	 - Command line utility for assembling Kubernetes CD pipelines
* [gotk bootstrap github](gotk_bootstrap_github.md)	 - Bootstrap toolkit components in a GitHub repository
* [gotk bootstrap gitlab](gotk_bootstrap_gitlab.md)	 - Bootstrap toolkit components in a GitLab repository

