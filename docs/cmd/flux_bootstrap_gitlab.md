---
title: "flux bootstrap gitlab command"
---
## flux bootstrap gitlab

Bootstrap toolkit components in a GitLab repository

### Synopsis

The bootstrap gitlab command creates the GitLab repository if it doesn't exists and
commits the toolkit components manifests to the master branch.
Then it configures the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.

```
flux bootstrap gitlab [flags]
```

### Examples

```
  # Create a GitLab API token and export it as an env var
  export GITLAB_TOKEN=<my-token>

  # Run bootstrap for a private repository using HTTPS token authentication
  flux bootstrap gitlab --owner=<group> --repository=<repository name> --token-auth

  # Run bootstrap for a private repository using SSH authentication
  flux bootstrap gitlab --owner=<group> --repository=<repository name>

  # Run bootstrap for a repository path
  flux bootstrap gitlab --owner=<group> --repository=<repository name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  flux bootstrap gitlab --owner=<user> --repository=<repository name> --private=false --personal --token-auth

  # Run bootstrap for a private repository hosted on a GitLab server
  flux bootstrap gitlab --owner=<group> --repository=<repository name> --hostname=<domain> --token-auth

  # Run bootstrap for a an existing repository with a branch named main
  flux bootstrap gitlab --owner=<organization> --repository=<repository name> --branch=main --token-auth
```

### Options

```
  -h, --help                    help for gitlab
      --hostname string         GitLab hostname (default "gitlab.com")
      --interval duration       sync interval (default 1m0s)
      --owner string            GitLab user or group name
      --path safeRelativePath   path relative to the repository root, when specified the cluster sync will be scoped to this path
      --personal                if true, the owner is assumed to be a GitLab user; otherwise a group
      --private                 if true, the repository is assumed to be private (default true)
      --repository string       GitLab repository name
      --ssh-hostname string     GitLab SSH hostname, to be used when the SSH host differs from the HTTPS one
```

### Options inherited from parent commands

```
      --branch string              default branch (for GitHub this must match the default branch setting for the organization) (default "main")
      --cluster-domain string      internal cluster domain (default "cluster.local")
      --components strings         list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
      --components-extra strings   list of components in addition to those supplied or defaulted, accepts comma-separated values
      --context string             kubernetes context to use
      --image-pull-secret string   Kubernetes secret name used for pulling the toolkit images from a private registry
      --kubeconfig string          absolute path to the kubeconfig file
      --log-level logLevel         log level, available options are: (debug, info, error) (default info)
  -n, --namespace string           the namespace scope for this operation (default "flux-system")
      --network-policy             deny ingress access to the toolkit controllers from other namespaces using network policies (default true)
      --registry string            container registry where the toolkit images are published (default "ghcr.io/fluxcd")
      --timeout duration           timeout for this operation (default 5m0s)
      --token-auth                 when enabled, the personal access token will be used instead of SSH deploy key
      --toleration-keys strings    list of toleration keys used to schedule the components pods onto nodes with matching taints
      --verbose                    print generated objects
  -v, --version string             toolkit version, when specified the manifests are downloaded from https://github.com/fluxcd/flux2/releases
      --watch-all-namespaces       watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed (default true)
```

### SEE ALSO

* [flux bootstrap](/cmd/flux_bootstrap/)	 - Bootstrap toolkit components

