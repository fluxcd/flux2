## tk resume helmrelease

Resume a suspended HelmRelease

### Synopsis

The resume command marks a previously suspended HelmRelease resource for reconciliation and waits for it to
finish the apply.

```
tk resume helmrelease [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing Helm release
  tk resume hr podinfo

```

### Options

```
  -h, --help   help for helmrelease
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --namespace string    the namespace scope for this operation (default "gitops-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [tk resume](tk_resume.md)	 - Resume suspended resources

