## gotk delete alert

Delete a Alert resource

### Synopsis

The delete alert command removes the given Alert from the cluster.

```
gotk delete alert [name] [flags]
```

### Examples

```
  # Delete an Alert and the Kubernetes resources created by it
  gotk delete alert main

```

### Options

```
  -h, --help   help for alert
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

