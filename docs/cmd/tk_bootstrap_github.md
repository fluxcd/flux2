## tk bootstrap github

Bootstrap toolkit components in a GitHub repository

### Synopsis

The bootstrap github command creates the GitHub repository if it doesn't exists and
commits the toolkit components manifests to the master branch.
Then it configures the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.

```
tk bootstrap github [flags]
```

### Examples

```
  # Create a GitHub personal access token and export it as an env var
  export GITHUB_TOKEN=<my-token>

  # Run bootstrap for a private repo owned by a GitHub organization
  bootstrap github --owner=<organization> --repository=<repo name>

  # Run bootstrap for a private repo and assign organization teams to it
  bootstrap github --owner=<organization> --repository=<repo name> --team=<team1 slug> --team=<team2 slug>

  # Run bootstrap for a repository path
  bootstrap github --owner=<organization> --repository=<repo name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  bootstrap github --owner=<user> --repository=<repo name> --private=false --personal=true 

  # Run bootstrap for a private repo hosted on GitHub Enterprise
  bootstrap github --owner=<organization> --repository=<repo name> --hostname=<domain>

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
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller])
      --kubeconfig string    path to the kubeconfig file (default "~/.kube/config")
      --namespace string     the namespace scope for this operation (default "gitops-system")
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
      --version string       toolkit tag or branch (default "master")
```

### SEE ALSO

* [tk bootstrap](tk_bootstrap.md)	 - Bootstrap toolkit components

