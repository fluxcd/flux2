## gotk create source

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
  -n, --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk create](gotk_create.md)	 - Create or update sources and resources
* [gotk create source git](gotk_create_source_git.md)	 - Create or update a GitRepository source
* [gotk create source helm](gotk_create_source_helm.md)	 - Create or update a HelmRepository source

