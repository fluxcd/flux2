## flux create secret

Create or update Kubernetes secrets

### Synopsis

The create source sub-commands generate Kubernetes secrets specific to Flux.

### Options

```
  -h, --help   help for secret
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux create](flux_create.md)	 - Create or update sources and resources
* [flux create secret git](flux_create_secret_git.md)	 - Create or update a Kubernetes secret for Git authentication
* [flux create secret helm](flux_create_secret_helm.md)	 - Create or update a Kubernetes secret for Helm repository authentication

