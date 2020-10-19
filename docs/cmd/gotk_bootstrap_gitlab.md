## gotk bootstrap gitlab

Bootstrap toolkit components in a GitLab repository

### Synopsis

The bootstrap gitlab command creates the GitLab repository if it doesn't exists and
commits the toolkit components manifests to the master branch.
Then it configures the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.

```
gotk bootstrap gitlab [flags]
```

### Examples

```
  # Create a GitLab API token and export it as an env var
  export GITLAB_TOKEN=<my-token>

  # Run bootstrap for a private repo using HTTPS token authentication 
  gotk bootstrap gitlab --owner=<group> --repository=<repo name>

  # Run bootstrap for a private repo using SSH authentication
  gotk bootstrap gitlab --owner=<group> --repository=<repo name> --ssh-hostname=gitlab.com

  # Run bootstrap for a repository path
  gotk bootstrap gitlab --owner=<group> --repository=<repo name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  gotk bootstrap gitlab --owner=<user> --repository=<repo name> --private=false --personal=true

  # Run bootstrap for a private repo hosted on a GitLab server 
  gotk bootstrap gitlab --owner=<group> --repository=<repo name> --hostname=<domain>

  # Run bootstrap for a an existing repository with a branch named main
  gotk bootstrap gitlab --owner=<organization> --repository=<repo name> --branch=main

```

### Options

```
  -h, --help                  help for gitlab
      --hostname string       GitLab hostname (default "gitlab.com")
      --interval duration     sync interval (default 1m0s)
      --owner string          GitLab user or group name
      --path string           repository path, when specified the cluster sync will be scoped to this path
      --personal              is personal repository
      --private               is private repository (default true)
      --repository string     GitLab repository name
      --ssh-hostname string   GitLab SSH hostname, when specified a deploy key will be added to the repository
```

### Options inherited from parent commands

```
      --arch arch                  cluster architecture, available options are: (amd64, arm, arm64) (default amd64)
      --branch string              default branch (for GitHub this must match the default branch setting for the organization) (default "main")
      --components strings         list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
      --image-pull-secret string   Kubernetes secret name used for pulling the toolkit images from a private registry
      --kubeconfig string          path to the kubeconfig file (default "~/.kube/config")
      --log-level logLevel         log level, available options are: (debug, info, error) (default info)
  -n, --namespace string           the namespace scope for this operation (default "gotk-system")
      --network-policy             deny ingress access to the toolkit controllers from other namespaces using network policies (default true)
      --registry string            container registry where the toolkit images are published (default "ghcr.io/fluxcd")
      --timeout duration           timeout for this operation (default 5m0s)
      --verbose                    print generated objects
  -v, --version string             toolkit version (default "latest")
      --watch-all-namespaces       watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed (default true)
```

### SEE ALSO

* [gotk bootstrap](gotk_bootstrap.md)	 - Bootstrap toolkit components

