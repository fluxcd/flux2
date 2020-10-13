## gotk delete receiver

Delete a Receiver resource

### Synopsis

The delete receiver command removes the given Receiver from the cluster.

```
gotk delete receiver [name] [flags]
```

### Examples

```
  # Delete an Receiver and the Kubernetes resources created by it
  gotk delete receiver main

```

### Options

```
  -h, --help   help for receiver
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

