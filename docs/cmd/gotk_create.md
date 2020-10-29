## gotk create

Create or update sources and resources

### Synopsis

The create sub-commands generate sources and resources.

### Options

```
      --export              export in YAML format to stdout
  -h, --help                help for create
      --interval duration   source sync interval (default 1m0s)
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk](gotk.md)	 - Command line utility for assembling Kubernetes CD pipelines
* [gotk create alert](gotk_create_alert.md)	 - Create or update a Alert resource
* [gotk create alert-provider](gotk_create_alert-provider.md)	 - Create or update a Provider resource
* [gotk create helmrelease](gotk_create_helmrelease.md)	 - Create or update a HelmRelease resource
* [gotk create kustomization](gotk_create_kustomization.md)	 - Create or update a Kustomization resource
* [gotk create receiver](gotk_create_receiver.md)	 - Create or update a Receiver resource
* [gotk create source](gotk_create_source.md)	 - Create or update sources

