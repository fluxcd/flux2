## flux delete alert

Delete a Alert resource

### Synopsis

The delete alert command removes the given Alert from the cluster.

```
flux delete alert [name] [flags]
```

### Examples

```
  # Delete an Alert and the Kubernetes resources created by it
  flux delete alert main

```

### Options

```
  -h, --help   help for alert
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
  -s, --silent              delete resource without asking for confirmation
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux delete](flux_delete.md)	 - Delete sources and resources

