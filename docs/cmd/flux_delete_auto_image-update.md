## flux delete auto image-update

Delete an ImageUpdateAutomation object

### Synopsis

The delete auto image-update command deletes the given ImageUpdateAutomation from the cluster.

```
flux delete auto image-update [name] [flags]
```

### Examples

```
  # Delete an image update automation
  flux delete auto image-update latest-images

```

### Options

```
  -h, --help   help for image-update
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

