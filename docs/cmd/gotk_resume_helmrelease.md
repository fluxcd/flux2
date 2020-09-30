## gotk resume helmrelease

Resume a suspended HelmRelease

### Synopsis

The resume command marks a previously suspended HelmRelease resource for reconciliation and waits for it to
finish the apply.

```
gotk resume helmrelease [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing Helm release
  gotk resume hr podinfo

```

### Options

```
  -h, --help   help for helmrelease
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk resume](gotk_resume.md)	 - Resume suspended resources

