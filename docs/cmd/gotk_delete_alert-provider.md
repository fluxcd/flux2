## gotk delete alert-provider

Delete a Provider resource

### Synopsis

The delete alert-provider command removes the given Provider from the cluster.

```
gotk delete alert-provider [name] [flags]
```

### Examples

```
  # Delete a Provider and the Kubernetes resources created by it
  gotk delete alert-provider slack

```

### Options

```
  -h, --help   help for alert-provider
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

