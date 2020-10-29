## flux create source

Create or update sources

### Synopsis

The create source sub-commands generate sources.

### Options

```
  -h, --help   help for source
```

### Options inherited from parent commands

```
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
* [flux create source bucket](flux_create_source_bucket.md)	 - Create or update a Bucket source
* [flux create source git](flux_create_source_git.md)	 - Create or update a GitRepository source
* [flux create source helm](flux_create_source_helm.md)	 - Create or update a HelmRepository source

