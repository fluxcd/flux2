## flux delete auto image-policy

Delete an ImagePolicy object

### Synopsis

The delete auto image-policy command deletes the given ImagePolicy from the cluster.

```
flux delete auto image-policy [name] [flags]
```

### Examples

```
  # Delete an image policy
  flux delete auto image-policy alpine3.x

```

### Options

```
  -h, --help   help for image-policy
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

* [flux delete auto](flux_delete_auto.md)	 - Delete automation objects

