## gotk delete helmrelease

Delete a HelmRelease resource

### Synopsis

The delete helmrelease command removes the given HelmRelease from the cluster.

```
gotk delete helmrelease [name] [flags]
```

### Examples

```
  # Delete a Helm release and the Kubernetes resources created by it
  gotk delete hr podinfo

```

### Options

```
  -h, --help   help for helmrelease
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
  -s, --silent              delete resource without asking for confirmation
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk delete](gotk_delete.md)	 - Delete sources and resources

