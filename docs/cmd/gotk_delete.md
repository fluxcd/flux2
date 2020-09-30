## gotk delete

Delete sources and resources

### Synopsis

The delete sub-commands delete sources and resources.

### Options

```
  -h, --help     help for delete
  -s, --silent   delete resource without asking for confirmation
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk](gotk.md)	 - Command line utility for assembling Kubernetes CD pipelines
* [gotk delete helmrelease](gotk_delete_helmrelease.md)	 - Delete a HelmRelease resource
* [gotk delete kustomization](gotk_delete_kustomization.md)	 - Delete a Kustomization resource
* [gotk delete source](gotk_delete_source.md)	 - Delete sources

