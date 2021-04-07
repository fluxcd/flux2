---
title: "flux bootstrap command"
---
## flux bootstrap

Bootstrap toolkit components

### Synopsis

The bootstrap sub-commands bootstrap the toolkit components on the targeted Git provider.

### Options

```
      --author-email string                    author email for Git commits
      --author-name string                     author name for Git commits (default "Flux")
      --branch string                          Git branch (default "main")
      --ca-file string                         path to TLS CA file used for validating self-signed certificates
      --cluster-domain string                  internal cluster domain (default "cluster.local")
      --commit-message-appendix string         string to add to the commit messages, e.g. '[ci skip]'
      --components strings                     list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
      --components-extra strings               list of components in addition to those supplied or defaulted, accepts comma-separated values
  -h, --help                                   help for bootstrap
      --image-pull-secret string               Kubernetes secret name used for pulling the toolkit images from a private registry
      --log-level logLevel                     log level, available options are: (debug, info, error) (default info)
      --network-policy                         deny ingress access to the toolkit controllers from other namespaces using network policies (default true)
      --private-key-file string                path to a private key file used for authenticating to the Git SSH server
      --recurse-submodules                     when enabled, configures the GitRepository source to initialize and include Git submodules in the artifact it produces
      --registry string                        container registry where the toolkit images are published (default "ghcr.io/fluxcd")
      --secret-name string                     name of the secret the sync credentials can be found in or stored to (default "flux-system")
      --ssh-ecdsa-curve ecdsaCurve             SSH ECDSA public key curve (p256, p384, p521) (default p384)
      --ssh-hostname string                    SSH hostname, to be used when the SSH host differs from the HTTPS one
      --ssh-key-algorithm publicKeyAlgorithm   SSH public key algorithm (rsa, ecdsa, ed25519) (default rsa)
      --ssh-rsa-bits rsaKeyBits                SSH RSA public key bit size (multiplies of 8) (default 2048)
      --token-auth                             when enabled, the personal access token will be used instead of SSH deploy key
      --toleration-keys strings                list of toleration keys used to schedule the components pods onto nodes with matching taints
  -v, --version string                         toolkit version, when specified the manifests are downloaded from https://github.com/fluxcd/flux2/releases
      --watch-all-namespaces                   watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed (default true)
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

* [flux](../flux/)	 - Command line utility for assembling Kubernetes CD pipelines
* [flux bootstrap git](../flux_bootstrap_git/)	 - Bootstrap toolkit components in a Git repository
* [flux bootstrap github](../flux_bootstrap_github/)	 - Bootstrap toolkit components in a GitHub repository
* [flux bootstrap gitlab](../flux_bootstrap_gitlab/)	 - Bootstrap toolkit components in a GitLab repository

