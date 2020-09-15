## gotk delete source git

Delete a GitRepository source

### Synopsis

The delete source git command deletes the given GitRepository from the cluster.

```
gotk delete source git [name] [flags]
```

### Examples

```
  # Delete a Git repository
  gotk delete source git podinfo

```

### Options

```
  -h, --help   help for git
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

* [gotk delete source](gotk_delete_source.md)	 - Delete sources

