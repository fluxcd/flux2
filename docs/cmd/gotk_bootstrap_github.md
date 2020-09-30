## gotk bootstrap github

Bootstrap toolkit components in a GitHub repository

### Synopsis

The bootstrap github command creates the GitHub repository if it doesn't exists and
commits the toolkit components manifests to the master branch.
Then it configures the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.

```
gotk bootstrap github [flags]
```

### Examples

```
  # Create a GitHub personal access token and export it as an env var
  export GITHUB_TOKEN=<my-token>

  # Run bootstrap for a private repo owned by a GitHub organization
  gotk bootstrap github --owner=<organization> --repository=<repo name>

  # Run bootstrap for a private repo and assign organization teams to it
  gotk bootstrap github --owner=<organization> --repository=<repo name> --team=<team1 slug> --team=<team2 slug>

  # Run bootstrap for a repository path
  gotk bootstrap github --owner=<organization> --repository=<repo name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  gotk bootstrap github --owner=<user> --repository=<repo name> --private=false --personal=true 

  # Run bootstrap for a private repo hosted on GitHub Enterprise
  gotk bootstrap github --owner=<organization> --repository=<repo name> --hostname=<domain>

  # Run bootstrap for a an existing repository with a branch named main
  gotk bootstrap github --owner=<organization> --repository=<repo name> --branch=main

```

### Options

```
  -h, --help                help for github
      --hostname string     GitHub hostname (default "github.com")
      --interval duration   sync interval (default 1m0s)
      --owner string        GitHub user or organization name
      --path string         repository path, when specified the cluster sync will be scoped to this path
      --personal            is personal repository
      --private             is private repository (default true)
      --repository string   GitHub repository name
      --team stringArray    GitHub team to be given maintainer access
```

### Options inherited from parent commands

```
      --arch string                arch can be amd64 or arm64 (default "amd64")
      --branch string              default branch (for GitHub this must match the default branch setting for the organization) (default "master")
      --components strings         list of components, accepts comma-separated values (default [source-controller,kustomize-controller,helm-controller,notification-controller])
      --image-pull-secret string   Kubernetes secret name used for pulling the toolkit images from a private registry
      --kubeconfig string          path to the kubeconfig file (default "~/.kube/config")
      --log-level string           set the controllers log level (default "info")
  -n, --namespace string           the namespace scope for this operation (default "gotk-system")
      --registry string            container registry where the toolkit images are published (default "ghcr.io/fluxcd")
      --timeout duration           timeout for this operation (default 5m0s)
      --verbose                    print generated objects
  -v, --version string             toolkit version (default "latest")
      --watch-all-namespaces       watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed (default true)
```

### SEE ALSO

* [gotk bootstrap](gotk_bootstrap.md)	 - Bootstrap toolkit components

