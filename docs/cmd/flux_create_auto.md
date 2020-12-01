## flux create auto

Create or update resources dealing with automation

### Synopsis

The create auto sub-cmmands works with automation objects; that is,
object controlling updates to git based on e.g., new container images
being available.

### Options

```
  -h, --help   help for auto
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
* [flux create auto image-repository](flux_create_auto_image-repository.md)	 - Create or update an ImageRepository object

