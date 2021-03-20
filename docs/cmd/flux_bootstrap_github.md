---
title: "flux bootstrap github command"
---
## flux bootstrap github

Bootstrap toolkit components in a GitHub repository

### Synopsis

The bootstrap github command creates the GitHub repository if it doesn't exists and
commits the toolkit components manifests to the main branch.
Then it configures the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.

```
flux bootstrap github [flags]
```

### Examples

```
  # Create a GitHub personal access token and export it as an env var
  export GITHUB_TOKEN=<my-token>

  # Run bootstrap for a private repository owned by a GitHub organization
  flux bootstrap github --owner=<organization> --repository=<repository name>

  # Run bootstrap for a private repository and assign organization teams to it
  flux bootstrap github --owner=<organization> --repository=<repository name> --team=<team1 slug> --team=<team2 slug>

  # Run bootstrap for a repository path
  flux bootstrap github --owner=<organization> --repository=<repository name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  flux bootstrap github --owner=<user> --repository=<repository name> --private=false --personal=true

  # Run bootstrap for a private repository hosted on GitHub Enterprise using SSH auth
  flux bootstrap github --owner=<organization> --repository=<repository name> --hostname=<domain> --ssh-hostname=<domain>

  # Run bootstrap for a private repository hosted on GitHub Enterprise using HTTPS auth
  flux bootstrap github --owner=<organization> --repository=<repository name> --hostname=<domain> --token-auth

  # Run bootstrap for an existing repository with a branch named main
  flux bootstrap github --owner=<organization> --repository=<repository name> --branch=main
```

### Options

```
  -h, --help                    help for github
      --hostname string         GitHub hostname (default "github.com")
      --interval duration       sync interval (default 1m0s)
      --owner string            GitHub user or organization name
      --path safeRelativePath   path relative to the repository root, when specified the cluster sync will be scoped to this path
      --personal                if true, the owner is assumed to be a GitHub user; otherwise an org
      --private                 if true, the repository is assumed to be private (default true)
      --read-write-key          if true, the deploy key is configured with read/write permissions
      --repository string       GitHub repository name
      --team stringArray        GitHub team to be given maintainer access
```

### Options inherited from parent commands

```
      --author-email string                    author email for Git commits
      --author-name string                     author name for Git commits (default "Flux")
      --branch string                          default branch (for GitHub this must match the default branch setting for the organization) (default "main")
      --ca-file string                         path to TLS CA file used for validating self-signed certificates
      --cluster-domain string                  internal cluster domain (default "cluster.local")
      --components strings                     list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
      --components-extra strings               list of components in addition to those supplied or defaulted, accepts comma-separated values
      --context string                         kubernetes context to use
      --image-pull-secret string               Kubernetes secret name used for pulling the toolkit images from a private registry
      --kubeconfig string                      absolute path to the kubeconfig file
      --log-level logLevel                     log level, available options are: (debug, info, error) (default info)
  -n, --namespace string                       the namespace scope for this operation (default "flux-system")
      --network-policy                         deny ingress access to the toolkit controllers from other namespaces using network policies (default true)
      --private-key-file string                path to a private key file used for authenticating to the Git SSH server
      --registry string                        container registry where the toolkit images are published (default "ghcr.io/fluxcd")
      --secret-name string                     name of the secret the sync credentials can be found in or stored to (default "flux-system")
      --ssh-ecdsa-curve ecdsaCurve             SSH ECDSA public key curve (p256, p384, p521) (default p384)
      --ssh-hostname string                    SSH hostname, to be used when the SSH host differs from the HTTPS one
      --ssh-key-algorithm publicKeyAlgorithm   SSH public key algorithm (rsa, ecdsa, ed25519) (default rsa)
      --ssh-rsa-bits rsaKeyBits                SSH RSA public key bit size (multiplies of 8) (default 2048)
      --timeout duration                       timeout for this operation (default 5m0s)
      --token-auth                             when enabled, the personal access token will be used instead of SSH deploy key
      --toleration-keys strings                list of toleration keys used to schedule the components pods onto nodes with matching taints
      --verbose                                print generated objects
  -v, --version string                         toolkit version, when specified the manifests are downloaded from https://github.com/fluxcd/flux2/releases
      --watch-all-namespaces                   watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed (default true)
```

### SEE ALSO

* [flux bootstrap](../flux_bootstrap/)	 - Bootstrap toolkit components

