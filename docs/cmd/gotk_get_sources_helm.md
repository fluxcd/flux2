## gotk get sources helm

Get HelmRepository source statuses

### Synopsis

The get sources helm command prints the status of the HelmRepository sources.

```
gotk get sources helm [flags]
```

### Examples

```
  # List all Helm repositories and their status
  gotk get sources helm

```

### Options

```
  -h, --help   help for helm
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk get sources](gotk_get_sources.md)	 - Get source statuses

