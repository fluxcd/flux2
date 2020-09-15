## gotk delete kustomization

Delete a Kustomization resource

### Synopsis

The delete kustomization command deletes the given Kustomization from the cluster.

```
gotk delete kustomization [name] [flags]
```

### Examples

```
  # Delete a kustomization and the Kubernetes resources created by it
  gotk delete kustomization podinfo

```

### Options

```
  -h, --help   help for kustomization
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

