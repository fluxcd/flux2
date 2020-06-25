## tk bootstrap

Bootstrap toolkit components

### Synopsis

The bootstrap sub-commands bootstrap the toolkit components on the targeted Git provider.

### Options

```
  -h, --help             help for bootstrap
      --version string   toolkit tag or branch (default "master")
```

### Options inherited from parent commands

```
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller])
      --kubeconfig string    path to the kubeconfig file (default "~/.kube/config")
      --namespace string     the namespace scope for this operation (default "gitops-system")
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
```

### SEE ALSO

* [tk](tk.md)	 - Command line utility for assembling Kubernetes CD pipelines
* [tk bootstrap github](tk_bootstrap_github.md)	 - Bootstrap toolkit components in a GitHub repository
* [tk bootstrap gitlab](tk_bootstrap_gitlab.md)	 - Bootstrap toolkit components in a GitLab repository

